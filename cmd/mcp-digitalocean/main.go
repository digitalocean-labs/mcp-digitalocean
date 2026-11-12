package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	middleware "mcp-digitalocean/internal"
	"mcp-digitalocean/internal/oauthmeta"
	"mcp-digitalocean/internal/openaichallenge"
	"mcp-digitalocean/internal/resourceid"
	"mcp-digitalocean/internal/wslogging"
	"mcp-digitalocean/pkg/registry"
	"mcp-digitalocean/pkg/registry/docs"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/server"
	"golang.org/x/oauth2"
)

const (
	mcpName                 = "mcp-digitalocean"
	mcpVersion              = "1.6.6"
	wsLoggingContextTimeout = 15 * time.Second
	// mcpEndpointPath is the path the streamable HTTP server serves the MCP
	// protocol on. It matches mcp-go's default so existing clients are unaffected.
	mcpEndpointPath = "/mcp"
	// resourceIdentifierHeader carries the AES-256-GCM ciphertext of this
	// server's MCP resource URL so the backend can decrypt and validate the
	// OAuth audience / resource indicator.
	resourceIdentifierHeader = "X-Encrypted-Resource-Identifier"
	// oauthTokenPrefix marks a token issued through the OAuth flow. Only those
	// tokens carry an audience the backend can validate, so personal access
	// tokens must not be sent with the resource identifier header.
	oauthTokenPrefix = "doo_"
)

// publicContentServices may run with --allow-anonymous: their tools read
// published content and never construct a DigitalOcean API client. Any other
// service keeps the bearer guard, so a stray ALLOW_ANONYMOUS cannot drop
// authentication from an account-scoped deployment.
var publicContentServices = map[string]struct{}{
	"docs": {},
}

// checkAnonymousServices reports whether the enabled service set may run
// without the bearer guard. An empty set means every service is registered,
// which includes account-scoped ones, so it is rejected along with any service
// outside publicContentServices. Names are compared exactly as the registry
// receives them, so a name this accepts is always one the registry can resolve.
func checkAnonymousServices(services []string) error {
	if len(services) == 0 {
		return errors.New("--allow-anonymous requires --services to list only public-content services (docs); all services are enabled")
	}
	var rejected []string
	for _, s := range services {
		if _, ok := publicContentServices[s]; !ok {
			rejected = append(rejected, fmt.Sprintf("%q", s))
		}
	}
	if len(rejected) > 0 {
		return fmt.Errorf("--allow-anonymous supports only public-content services (docs), got %s", strings.Join(rejected, ", "))
	}
	return nil
}

// docsEnabled reports whether the docs service is registered. An empty set
// registers every service, docs included.
func docsEnabled(services []string) bool {
	return len(services) == 0 || slices.Contains(services, "docs")
}

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is empty or not present, it returns the fallback value.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvBool accepts the forms strconv.ParseBool does ("1", "true", "TRUE"),
// so an operator writing ALLOW_ANONYMOUS=1 gets what they intended. An
// unparseable value falls back rather than guessing; invalidBoolEnv reports it
// so startup can warn once the logger exists.
func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return parsed
}

// getEnvFloat and getEnvInt fall back on unparseable values, like getEnvBool.
func getEnvFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return v
}

// invalidBoolEnv returns the raw value of key when it is set to something
// getEnvBool cannot parse, otherwise the empty string.
func invalidBoolEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		return ""
	}
	if _, err := strconv.ParseBool(strings.TrimSpace(v)); err != nil {
		return v
	}
	return ""
}

func main() {
	logLevelFlag := flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level: debug, info, warn, error")
	serviceFlag := flag.String("services", getEnv("SERVICES", ""), "Comma-separated list of services to activate (e.g., apps,networking,droplets)")
	tokenFlag := flag.String("digitalocean-api-token", getEnv("DIGITALOCEAN_API_TOKEN", ""), "DigitalOcean API token")
	endpointFlag := flag.String("digitalocean-api-endpoint", getEnv("DIGITALOCEAN_API_ENDPOINT", "https://api.digitalocean.com"), "DigitalOcean API endpoint")
	transport := flag.String("transport", getEnv("TRANSPORT", "stdio"), "The transport protocol to use (http or stdio). Default is stdio.")
	bindAddr := flag.String("bind-addr", getEnv("BIND_ADDR", "127.0.0.1:8080"), "Bind address to bind to. Only used for http transport.")
	wsLoggingURL := flag.String("ws-logging-url", getEnv("WS_LOGGING_URL", ""), "WebSocket URL for WebSocket logging (optional)")
	wsLoggingToken := flag.String("ws-logging-token", getEnv("WS_LOGGING_TOKEN", ""), "Authentication token for WebSocket logging (optional)")
	enableToolErrorLogging := flag.Bool("enable-tool-error-logging", getEnv("ENABLE_TOOL_ERROR_LOGGING", "false") == "true", "Enable logging of tool errors")
	serverURLFlag := flag.String("mcp-resource-url", getEnv("MCP_RESOURCE_URL", ""), "This server's public base URL advertised in the OAuth protected resource metadata. When empty, it is derived from each request (remote transport only, optional)")
	openaiAppsVerificationTokenFlag := flag.String("openai-apps-verification-token", getEnv("OPENAI_APPS_VERIFICATION_TOKEN", ""), "Plain-text token served at /.well-known/openai-apps-challenge for OpenAI ChatGPT app domain verification (remote transport only, optional)")
	userAgent := flag.String("user-agent", getEnv("USER_AGENT", ""), "Indicate this server is running as a remote MCP ")
	resourceEncryptionKey := flag.String("encrypted-resource-key", getEnv("ENCRYPTED_RESOURCE_KEY", ""), "32-byte AES-256 key used to encrypt --mcp-resource-url for the X-Encrypted-Resource-Identifier header (remote MCP only, optional)")
	scopes := flag.String("scopes", getEnv("SCOPES", "read,write"), "Comma-separated list of scopes to request from the DigitalOcean API")
	allowAnonymous := flag.Bool("allow-anonymous", getEnvBool("ALLOW_ANONYMOUS", false), "Serve the MCP endpoint without the bearer-token guard or OAuth metadata. Requires --services to list only public-content services (docs); the server refuses to start otherwise (remote transport only)")
	docsSearchAPI := flag.Bool("docs-search-api", getEnvBool("DOCS_SEARCH_API", false), "Serve the plain-HTTP docs search endpoint at "+docs.SearchAPIPath+". Requires --allow-anonymous (remote transport only)")
	docsSearchRateMode := flag.String("docs-search-rate-limit-mode", getEnv("DOCS_SEARCH_RATE_LIMIT_MODE", string(docs.RateLimitShadow)), "Per-client rate limiting for the docs search endpoint: off, shadow (log only), or enforce")
	docsSearchRPS := flag.Float64("docs-search-rate-limit-rps", getEnvFloat("DOCS_SEARCH_RATE_LIMIT_RPS", 10), "Sustained requests per second allowed per client IP on the docs search endpoint")
	docsSearchBurst := flag.Int("docs-search-rate-limit-burst", getEnvInt("DOCS_SEARCH_RATE_LIMIT_BURST", 30), "Burst size per client IP on the docs search endpoint")
	docsSearchIPHeader := flag.String("docs-search-client-ip-header", getEnv("DOCS_SEARCH_CLIENT_IP_HEADER", ""), "Header carrying the client IP set by a trusted edge (e.g. CF-Connecting-IP). Leave empty unless all traffic arrives through that edge")
	flag.Parse()

	var level slog.Level
	switch strings.ToLower(*logLevelFlag) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// setup signal context for graceful shutdown
	// This context is cancelled when the user presses Ctrl+C or the process receives SIGTERM/SIGINT.
	// It is used to signal all long-running goroutines (like WebSocket handlers) to stop their work.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// create WebSocket logging handler (drop-in replacement for slog.NewJSONHandler)
	wsLoggingHandler := wslogging.NewHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	// configure WebSocket logging if URL is provided
	if *wsLoggingURL != "" {
		if err := wsLoggingHandler.ConfigureWebSocket(*wsLoggingURL, *wsLoggingToken); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to configure WebSocket logging: %v\n", err)
			os.Exit(1)
		}

		// start WebSocket logging with signal context for graceful shutdown
		// The context passed here controls when the background goroutines should stop.
		// When a signal is received, ctx is cancelled and the goroutines exit their loops.
		wsLoggingHandler.Start(ctx)

		defer func() {
			// give the handler time to flush remaining logs before shutdown
			// we are not reusing the signal context because:
			// 1. The signal context (ctx) is already cancelled at this point
			// 2. Close() needs time to flush buffered logs, so we give it a new wsLoggingContextTimeout (15-second)
			// which ensures logs are properly flushed even during shutdown
			closeCtx, cancel := context.WithTimeout(context.Background(), wsLoggingContextTimeout)
			defer cancel()
			if err := wsLoggingHandler.Close(closeCtx); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to close WebSocket handler: %v\n", err)
			}
		}()
	}

	var services []string
	if *serviceFlag != "" {
		services = strings.Split(*serviceFlag, ",")
	}

	// add enabled_services as persistent attribute for context/metrics
	// this helps with filtering and understanding server configuration
	if *serviceFlag != "" {
		wsLoggingHandler = wsLoggingHandler.WithAttrs([]slog.Attr{
			slog.String("enabled_services", *serviceFlag),
		}).(*wslogging.Handler)
	} else {
		wsLoggingHandler = wsLoggingHandler.WithAttrs([]slog.Attr{
			slog.String("enabled_services", "all"),
		}).(*wslogging.Handler)
	}

	// create logger after adding service attributes
	logger := slog.New(wsLoggingHandler)
	token := *tokenFlag
	if token == "" && *transport == "stdio" {
		logger.Error("DigitalOcean API token not provided. Use --digitalocean-api-token flag or set DIGITALOCEAN_API_TOKEN environment variable")
		os.Exit(1)
	}

	if v := invalidBoolEnv("ALLOW_ANONYMOUS"); v != "" {
		logger.Warn("ignoring ALLOW_ANONYMOUS: not a boolean", "value", v)
	}

	if *allowAnonymous {
		if *transport == "stdio" {
			logger.Warn("--allow-anonymous has no effect on stdio transport")
		} else if err := checkAnonymousServices(services); err != nil {
			logger.Error("refusing to start with anonymous access", "error", err)
			os.Exit(1)
		}
	}

	var extraRoutes map[string]http.Handler
	if *docsSearchAPI {
		if *transport == "stdio" || !*allowAnonymous {
			logger.Error("--docs-search-api requires --allow-anonymous and a remote transport")
			os.Exit(1)
		}
		mode, err := docs.ParseRateLimitMode(*docsSearchRateMode)
		if err != nil {
			logger.Error("invalid docs search rate limit configuration", "error", err)
			os.Exit(1)
		}
		rlCfg := docs.RateLimitConfig{
			Mode:            mode,
			RPS:             *docsSearchRPS,
			Burst:           *docsSearchBurst,
			TrustedIPHeader: strings.TrimSpace(*docsSearchIPHeader),
		}
		searchHandler, err := docs.RateLimit(docs.NewSearchHandler(logger), rlCfg, logger)
		if err != nil {
			logger.Error("invalid docs search rate limit configuration", "error", err)
			os.Exit(1)
		}
		if mode == docs.RateLimitShadow && rlCfg.TrustedIPHeader == "" {
			logger.Warn("docs search rate limiting keys on the connection address; behind an edge every client looks the same. Set --docs-search-client-ip-header")
		}
		extraRoutes = map[string]http.Handler{docs.SearchAPIPath: searchHandler}
		logger.Info("serving docs search api", "path", docs.SearchAPIPath, "rate_limit_mode", string(mode), "rps", rlCfg.RPS, "burst", rlCfg.Burst, "client_ip_header", rlCfg.TrustedIPHeader)
	}

	var opts []server.ServerOption
	opts = append(opts, server.WithPromptCapabilities(true))
	if docsEnabled(services) {
		opts = append(opts, server.WithInstructions(docs.ServerInstructions))
	}
	if *enableToolErrorLogging {
		toolLoggingMiddleware := middleware.ToolLoggingMiddleware{Logger: logger}
		opts = append(opts, server.WithToolHandlerMiddleware(toolLoggingMiddleware.ToolMiddleware))
	}

	svr := server.NewMCPServer(mcpName, mcpVersion, opts...)

	// For remote (non-stdio) transports, serve the OAuth protected resource
	// metadata document and challenge unauthenticated requests, unless
	// --allow-anonymous opts a public-content service (docs) out of both. The
	// resource is taken from --mcp-resource-url when set, otherwise derived from
	// each request's scheme and host.
	var (
		wellKnownHandler       http.HandlerFunc
		openaiChallengeHandler http.HandlerFunc
		requireAuth            func(http.Handler) http.Handler
	)
	if *transport != "stdio" {
		if *allowAnonymous {
			logger.Info("anonymous access enabled; not serving OAuth metadata or challenging requests")
		} else {
			authServer := oauthmeta.ProdAuthorizationServer
			serverURL := strings.TrimSpace(*serverURLFlag)

			oauthScopes := parseScopes(*scopes)
			wellKnownHandler = oauthmeta.Handler(oauthmeta.Config{
				Resource:               serverURL,
				AuthorizationServers:   []string{authServer},
				BearerMethodsSupported: []string{"header"},
				ScopesSupported:        oauthScopes,
			})

			challengeCfg := oauthmeta.ChallengeConfig{
				Resource: serverURL,
				Scopes:   oauthScopes,
			}
			requireAuth = func(next http.Handler) http.Handler {
				return oauthmeta.RequireBearer(next, challengeCfg)
			}

			logger.Info("serving OAuth protected resource metadata", "path", oauthmeta.WellKnownPath, "authorization_server", authServer, "scopes", oauthScopes)
		}

		if token := strings.TrimSpace(*openaiAppsVerificationTokenFlag); token != "" && token != "OPENAI_APPS_VERIFICATION_TOKEN" {
			openaiChallengeHandler = openaichallenge.Handler(token)
			logger.Info("serving OpenAI app domain verification", "path", openaichallenge.WellKnownPath)
		}
	}

	// When a resource encryption key is configured, encrypt this server's MCP
	// resource URL and attach the ciphertext as a header on every public API
	// request. The backend decrypts with the same key to validate the audience.
	var encryptedResourceHeader string
	if key := strings.TrimSpace(*resourceEncryptionKey); key != "" {
		resourceURL := strings.TrimSpace(*serverURLFlag)
		if resourceURL == "" {
			logger.Error("--mcp-resource-url / MCP_RESOURCE_URL is required when --encrypted-resource-key is set")
			os.Exit(1)
		}
		var err error
		encryptedResourceHeader, err = resourceid.EncryptResourceIdentifier([]byte(key), resourceURL)
		if err != nil {
			logger.Error("failed to encrypt MCP resource URL", "error", err)
			os.Exit(1)
		}
	}

	// by default, we create a new client per request.
	getClientFn := func(ctx context.Context) (*godo.Client, error) {
		return clientFromContext(ctx, *endpointFlag, *userAgent, encryptedResourceHeader)
	}

	// if using stdio, we can re-use the client.
	if *transport == "stdio" {
		godoClient, err := newGodoClientWithTokenAndEndpoint(context.Background(), token, *endpointFlag, *userAgent, "")
		if err != nil {
			logger.Error("Failed to create DigitalOcean client: " + err.Error())
			os.Exit(1)
		}
		getClientFn = func(ctx context.Context) (*godo.Client, error) {
			return godoClient, nil
		}
	}

	// register the tools.
	err := registry.Register(
		logger,
		svr,
		getClientFn,
		services...,
	)
	if err != nil {
		logger.Error("Failed to register tools: " + err.Error())
		os.Exit(1)
	}

	// start our server.
	err = runServer(ctx, svr, logger, *bindAddr, transport, wellKnownHandler, openaiChallengeHandler, requireAuth, extraRoutes)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Info("shutting down mcp server")
			os.Exit(0)
		} else {
			logger.Error("Failed to serve MCP server: " + err.Error())
			os.Exit(1)
		}
	}
}

func clientFromContext(ctx context.Context, endpoint string, userAgent string, encryptedResourceHeader string) (*godo.Client, error) {
	auth, ok := ctx.Value(middleware.AuthKey{}).(string)
	if !ok || strings.TrimSpace(auth) == "" {
		return nil, errors.New("no auth header found")
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return nil, errors.New("no bearer token found")
	}

	// When the request arrived through a trusted MCP router, the router forwards
	// the AES-256-GCM ciphertext of its own resource URL. Relay that value so the
	// backend validates the OAuth audience against the router (the resource the
	// client authenticated to) rather than this server's own URL. Fall back to
	// this server's precomputed identifier for direct clients.
	if routerResource := strings.TrimSpace(middleware.RouterResourceFromContext(ctx)); routerResource != "" {
		encryptedResourceHeader = routerResource
	}

	client, err := newGodoClientWithTokenAndEndpoint(ctx, token, endpoint, userAgent, encryptedResourceHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create godo client: %w", err)
	}

	return client, nil
}

// newGodoClientWithTokenAndEndpoint initializes a new godo client with a custom user agent and endpoint.
func newGodoClientWithTokenAndEndpoint(ctx context.Context, token string, endpoint string, userAgent string, encryptedResourceHeader string) (*godo.Client, error) {
	cleanToken := strings.Trim(strings.TrimSpace(token), "'")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cleanToken})
	oauthClient := oauth2.NewClient(ctx, ts)

	retry := godo.RetryConfig{
		RetryMax:     4,
		RetryWaitMin: godo.PtrTo(float64(1)),
		RetryWaitMax: godo.PtrTo(float64(30)),
	}

	mcpUserAgent := fmt.Sprintf("%s/%s", mcpName, mcpVersion)
	if userAgent != "" {
		mcpUserAgent = fmt.Sprintf("%s/%s", userAgent, mcpVersion)
	}

	opts := []godo.ClientOpt{
		godo.WithRetryAndBackoffs(retry),
		godo.SetBaseURL(endpoint),
		godo.SetUserAgent(mcpUserAgent),
	}

	// When running as a remote MCP server, attach the encrypted MCP resource URL
	// as a header on every request to the public API so the backend can decrypt
	// and validate the OAuth audience / resource indicator for this server.
	if encryptedResourceHeader != "" && strings.HasPrefix(cleanToken, oauthTokenPrefix) {
		opts = append(opts, godo.SetRequestHeaders(map[string]string{
			resourceIdentifierHeader: encryptedResourceHeader,
		}))
	}

	return godo.New(oauthClient, opts...)
}

// parseScopes splits a comma-separated SCOPES value into the list advertised
// in both the RFC 9728 scopes_supported field and the 401 WWW-Authenticate
// challenge.
func parseScopes(raw string) []string {
	var out []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// registerRoutes mounts the MCP endpoint (behind requireAuth when set), the
// well-known documents, and any extra routes. Extra routes are never wrapped
// by requireAuth: they exist only on anonymous, public-content deployments.
func registerRoutes(mux *http.ServeMux, mcpServer http.Handler, wellKnownHandler, openaiChallengeHandler http.HandlerFunc, requireAuth func(http.Handler) http.Handler, extraRoutes map[string]http.Handler) {
	mcpHandler := mcpServer
	if requireAuth != nil {
		mcpHandler = requireAuth(mcpServer)
	}
	mux.Handle(mcpEndpointPath, mcpHandler)
	if wellKnownHandler != nil {
		mux.HandleFunc(oauthmeta.WellKnownPath, wellKnownHandler)
	}
	if openaiChallengeHandler != nil {
		mux.HandleFunc(openaichallenge.WellKnownPath, openaiChallengeHandler)
	}
	for path, h := range extraRoutes {
		mux.Handle(path, h)
	}
}

func runServer(ctx context.Context, s *server.MCPServer, logger *slog.Logger, bindAddr string, transport *string, wellKnownHandler http.HandlerFunc, openaiChallengeHandler http.HandlerFunc, requireAuth func(http.Handler) http.Handler, extraRoutes map[string]http.Handler) error {
	logger.Info("starting MCP server", "name", mcpName, "version", mcpVersion, "transport", *transport)
	switch *transport {
	case "stdio":
		logger.Info("stdio server started")
		err := server.ServeStdio(s)
		if err != nil {
			return fmt.Errorf("failed to start http server: %w", err)
		}
	// fallback to http
	default:
		errC := make(chan error, 1)
		logger.Info("http server started", "bind_addr", bindAddr)

		// When extra routing is needed (the public OAuth metadata route and/or the
		// bearer-token guard on the MCP endpoint), we own the mux and hand it to the
		// streamable server via WithStreamableHTTPServer. The MCP protocol endpoint
		// is registered explicitly so its behavior is unchanged; the well-known
		// route is served alongside it and is intentionally left unauthenticated.
		var streamableOpts []server.StreamableHTTPOption
		streamableOpts = append(streamableOpts,
			server.WithHTTPContextFunc(middleware.AuthFromRequest),
			server.WithStateLess(true),
		)

		useCustomMux := wellKnownHandler != nil || openaiChallengeHandler != nil || requireAuth != nil || len(extraRoutes) > 0
		var mux *http.ServeMux
		if useCustomMux {
			mux = http.NewServeMux()
			streamableOpts = append(streamableOpts, server.WithStreamableHTTPServer(&http.Server{
				Handler: mux,
				// Bounds slow-header connections. No write timeout: MCP
				// responses can stream.
				ReadHeaderTimeout: 10 * time.Second,
				IdleTimeout:       120 * time.Second,
			}))
		}

		httpServer := server.NewStreamableHTTPServer(s, streamableOpts...)

		if mux != nil {
			// We provided a custom *http.Server, so routing is our responsibility.
			registerRoutes(mux, httpServer, wellKnownHandler, openaiChallengeHandler, requireAuth, extraRoutes)
		}

		go func() {
			errC <- httpServer.Start(bindAddr)
		}()

		select {
		case <-ctx.Done():

			// allow 15 seconds for graceful shutdown
			timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Second*15)
			defer cancelFunc()

			logger.Info("received shutdown signal")
			err := httpServer.Shutdown(timeoutCtx)
			if err != nil {
				// this happens if the clients still hold connections after the timeout.
				if errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("timeout waiting for server to shutdown: %w", err)
				}

				return fmt.Errorf("failed to gracefully shutdown http server: %w", err)
			}

			return nil
		case err := <-errC:
			if err != nil {
				logger.Error("http server error", "error", err)
				return fmt.Errorf("http server error: %w", err)
			}
		}
	}

	return nil
}

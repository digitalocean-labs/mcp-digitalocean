package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/browser"
	"github.com/zalando/go-keyring" // Import the keyring library
)

// --- Constants ---
// The 'service' name used to store credentials in the system's keyring.
const keyringService = "mcp-digitalocean-oauth-callback-go"
const keyringLastUsedTeam = "last-used-team"

// --- Environment Variables ---
var (
	thisEndpoint              = getEnv("THIS_ENDPOINT", "http://127.0.0.1:8080")
	upstreamAPIURL            = getEnv("UPSTREAM_API_URL", "https://api.digitalocean.com")
	digitalOceanOAuthClientID = getEnv("DIGITALOCEAN_OAUTH_CLIENT_ID", "61f2be08367bf0b0cd6142f66838f48d5729da42f33f300919b4a3f8a6152904")
	// The client secret is not used in the OAuth Implicit Grant Flow.
)

// --- Helper Functions ---

// getEnv gets an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// --- Core OAuth Logic ---

// createAuthorizeURL generates the DigitalOcean OAuth authorization URL for the Implicit Grant Flow.
func createAuthorizeURL() (string, error) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	baseURL := "https://cloud.digitalocean.com/v1/oauth/authorize"
	params := url.Values{}
	// Use "token" for the response_type to request the access token directly.
	params.Add("response_type", "token")
	params.Add("client_id", digitalOceanOAuthClientID)
	params.Add("scope", "read write")
	params.Add("state", state)
	params.Add("redirect_uri", thisEndpoint+"/v1/auth/digitalocean/callback")

	return fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil
}

// getTeamUUID retrieves the team UUID from the DigitalOcean account endpoint using a token.
func getTeamUUID(token string) (string, error) {
	req, err := http.NewRequest("GET", upstreamAPIURL+"/v2/account", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create account request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform account request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read account response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("account request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accountResponse struct {
		Account struct {
			Team struct {
				UUID string `json:"uuid"`
			} `json:"team"`
		} `json:"account"`
	}
	if err := json.Unmarshal(body, &accountResponse); err != nil {
		return "", fmt.Errorf("failed to parse account response: %w", err)
	}

	if accountResponse.Account.Team.UUID == "" {
		return "", fmt.Errorf("team UUID not found in account response")
	}

	return accountResponse.Account.Team.UUID, nil
}

// --- Secret Storage ---

// saveTokenToKeyring securely stores the token in the system's native keychain.
func saveTokenToKeyring(teamUUID, token string) error {
	err := keyring.Set(keyringService, teamUUID, token)
	if err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}
	err = keyring.Set(keyringService, keyringLastUsedTeam, teamUUID)
	if err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}
	slog.Info(fmt.Sprintf("Successfully saved token for team %s to the system keyring.", teamUUID))
	return nil
}

// GetLastUsedTeamUUID retrieves the last saved team UUID from the keyring.
func GetLastUsedTeamUUID() (string, error) {
	secret, err := keyring.Get(keyringService, keyringLastUsedTeam)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// GetTokenFromKeyring retrieves a token from the keyring for a given team UUID.
func GetTokenFromKeyring(teamUUID string) (string, error) {
	secret, err := keyring.Get(keyringService, teamUUID)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// --- Handlers ---

// callbackHandler handles the OAuth2 callback.
// It uses a two-step process for the implicit grant flow.
func callbackHandler(tokenChan chan string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 2: Check if the access_token is now in the query parameters
		// after being redirected by our own JavaScript.
		token := r.URL.Query().Get("access_token")

		// If the token is NOT in the query string, this is the first hit from DigitalOcean.
		// The token is in the URL fragment ('#'), which the server cannot see.
		if token == "" {
			// Step 1: Serve an HTML page with JavaScript.
			// This script will read the token from the fragment and reload the page,
			// moving the token into the query string ('?') for the server to read.
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head>
		<title>Authorizing...</title>
		<script>
				window.addEventListener('DOMContentLoaded', () => {
						const url = new URL(window.location.href);
						// Check if the URL has a hash component.
						const hash = url.hash.slice(1);

						// Only process if there is a hash.
						if (hash) {
								const searchParams = new URLSearchParams(hash);
								// If the access_token exists in the hash, move it to the query string.
								if (searchParams.has("access_token")) {
										url.search = searchParams.toString();
										url.hash = "";
										// Replace the URL to reload the page with the token in the query string.
										window.location.replace(url.toString());
								} else {
										// A hash exists, but it doesn't contain the token. This is an error.
										document.body.innerHTML = "<h1>Error</h1><p>Authentication failed. Invalid parameters found in the URL.</p>";
								}
						} else {
								// No hash. This means the process is complete and the token is stored.
								// Display the final success message.
								document.body.innerHTML = "<h1>Success!</h1><p>OAuth token has been successfully obtained and stored securely in your system's keychain.</p>";
						}
				});
		</script>
</head>
<body>
		<p>Please wait, processing authentication...</p>
</body>
</html>`)
			return // Stop further processing for this request.
		}

		// --- If we reach here, the token was successfully extracted and is in the query ---

		// Get account info to find the team UUID.
		teamUUID, err := getTeamUUID(token)
		if err != nil {
			http.Error(w, "Failed to get team information", http.StatusInternalServerError)
			slog.Error(fmt.Sprintf("Error getting team UUID: %v", err))
			return
		}

		// Save the token to the system keyring.
		if err := saveTokenToKeyring(teamUUID, token); err != nil {
			http.Error(w, "Failed to save token to system keyring", http.StatusInternalServerError)
			slog.Error(fmt.Sprintf("Error saving token: %v", err))
			return
		}

		// Signal that the token has been received.
		tokenChan <- token

		// Instead of writing HTML, redirect back to this same callback URL.
		// This clears the access_token from the browser's address bar. The page's
		// script will then see a URL with no hash and display the success message.
		http.Redirect(w, r, "/v1/auth/digitalocean/callback", http.StatusTemporaryRedirect)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// This handler should only respond to the root path.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	authURL, err := createAuthorizeURL()
	if err != nil {
		http.Error(w, "Failed to create authorization URL", http.StatusInternalServerError)
		slog.Error(fmt.Sprintf("Error creating authorization URL: %v", err))
		return
	}

	// Perform a 307 Temporary Redirect to the authorization URL.
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// LocalhostAuthorize starts the entire OAuth2 implicit grant flow. It starts a
// local web server, opens the browser for user authorization, and waits until
// the token is received and stored in the keyring. It returns the token.
func LocalhostAuthorize() (string, error) {
	// A channel to receive the token from the callback handler.
	tokenChan := make(chan string)
	// A channel to signal errors from the server.
	errChan := make(chan error, 1)

	// Create the HTTP server.
	server := &http.Server{Addr: "127.0.0.1:8080"}

	http.HandleFunc("/v1/auth/digitalocean/callback", callbackHandler(tokenChan))
	http.HandleFunc("/", rootHandler)

	// Start the server in a goroutine so it doesn't block.
	go func() {
		slog.Info(fmt.Sprintf("Starting OAuth server on %s...", thisEndpoint))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start server: %w", err)
		}
	}()

	// Open the user's browser to the root URL, which will redirect to DigitalOcean.
	slog.Info("Opening browser for authentication...")
	time.Sleep(1 * time.Second) // Give the server a moment to start.
	if err := browser.OpenURL(thisEndpoint); err != nil {
		slog.Warn(fmt.Sprintf("Warning: could not open browser automatically: %v", err))
		slog.Info(fmt.Sprintf("Please manually open %s in your browser to proceed.", thisEndpoint))
	}

	// Wait for a token from the callback or an error from the server.
	var token string
	select {
	case token = <-tokenChan:
		slog.Info("Token received successfully.")
	case err := <-errChan:
		return "", err
	case <-time.After(5 * time.Minute): // Timeout after 5 minutes.
		return "", fmt.Errorf("timed out waiting for OAuth token")
	}

	// Shutdown the server gracefully.
	slog.Info("Shutting down OAuth server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return "", fmt.Errorf("failed to shut down server gracefully: %w", err)
	}

	return token, nil
}

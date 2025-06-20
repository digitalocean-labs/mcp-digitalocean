package webapp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/zalando/go-keyring" // Import the keyring library
)

// --- Constants ---
// The 'service' name used to store credentials in the system's keyring.
const keyringService = "mcp-digitalocean-oauth-callback-go"

// --- Environment Variables ---
var (
	thisEndpoint                  = getEnv("THIS_ENDPOINT", "http://localhost:8080")
	upstreamAPIURL                = getEnv("UPSTREAM_API_URL", "https://api.digitalocean.com")
	digitalOceanOAuthClientID     = getEnv("DIGITALOCEAN_OAUTH_CLIENT_ID", "61f2be08367bf0b0cd6142f66838f48d5729da42f33f300919b4a3f8a6152904")
	digitalOceanOAuthClientSecret = getEnvOrDie("DIGITALOCEAN_OAUTH_CLIENT_SECRET")
)

// --- Helper Functions ---

// getEnv gets an environment variable or returns a default value.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// getEnvOrDie gets an environment variable or terminates the program if it's not set.
func getEnvOrDie(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Fatalf("Error: Environment variable %s not set.", key)
	return ""
}

// --- Core OAuth Logic ---

// createAuthorizeURL generates the DigitalOcean OAuth authorization URL.
func createAuthorizeURL() (string, error) {
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	baseURL := "https://cloud.digitalocean.com/v1/oauth/authorize"
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", digitalOceanOAuthClientID)
	params.Add("scope", "read write")
	params.Add("state", state)
	params.Add("redirect_uri", thisEndpoint+"/v1/auth/digitalocean/callback")

	return fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil
}

// exchangeCodeForToken makes a request to DigitalOcean to get an access token.
func exchangeCodeForToken(code string) (string, error) {
	tokenURL := "https://cloud.digitalocean.com/v1/oauth/token"
	params := url.Values{}
	params.Add("grant_type", "authorization_code")
	params.Add("code", code)
	params.Add("client_id", digitalOceanOAuthClientID)
	params.Add("client_secret", digitalOceanOAuthClientSecret)
	params.Add("redirect_uri", thisEndpoint+"/v1/auth/digitalocean/callback")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s?%s", tokenURL, params.Encode()), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResponse.AccessToken, nil
}

// getTeamUUID retrieves the team UUID from the DigitalOcean account endpoint.
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

// --- Secret Storage (MODIFIED) ---

// saveTokenToKeyring securely stores the token in the system's native keychain.
// The team's UUID is used as the 'user' or key for the secret.
func saveTokenToKeyring(teamUUID, token string) error {
	err := keyring.Set(keyringService, teamUUID, token)
	if err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}
	log.Printf("Successfully saved token for team %s to the system keyring.", teamUUID)
	return nil
}

// getTokenFromKeyring retrieves a token from the keyring for a given team UUID.
// This function is not used in the callback flow but is included for completeness.
func getTokenFromKeyring(teamUUID string) (string, error) {
	secret, err := keyring.Get(keyringService, teamUUID)
	if err != nil {
		return "", fmt.Errorf("failed to get token from keyring: %w", err)
	}
	return secret, nil
}

// --- Handlers ---

// callbackHandler handles the OAuth2 callback from DigitalOcean.
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	// If no code is present, redirect to the authorization URL.
	if code == "" {
		authURL, err := createAuthorizeURL()
		if err != nil {
			http.Error(w, "Failed to create authorization URL", http.StatusInternalServerError)
			log.Printf("Error creating authorize URL: %v", err)
			return
		}
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
		return
	}

	// Exchange authorization code for an access token.
	token, err := exchangeCodeForToken(code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		log.Printf("Error exchanging code for token: %v", err)
		return
	}

	// Get account info to find the team UUID.
	teamUUID, err := getTeamUUID(token)
	if err != nil {
		http.Error(w, "Failed to get team information", http.StatusInternalServerError)
		log.Printf("Error getting team UUID: %v", err)
		return
	}

	// MODIFIED: Save the token to the system keyring instead of a file.
	if err := saveTokenToKeyring(teamUUID, token); err != nil {
		http.Error(w, "Failed to save token to system keyring", http.StatusInternalServerError)
		log.Printf("Error saving token: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintln(w, "<h1>Success!</h1><p>OAuth token has been successfully obtained and stored securely in your system's keychain.</p>")
}

// cliHandler prints the authorization URL to the console.
func cliHandler() {
	authURL, err := createAuthorizeURL()
	if err != nil {
		log.Fatalf("Failed to create authorization URL: %v", err)
	}
	fmt.Println(authURL)
}

// --- Main Function ---

func main() {
	// Simple check for a "cli" argument to run in command-line mode.
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		cliHandler()
	} else {
		// Default to running as an HTTP server.
		http.HandleFunc("/v1/auth/digitalocean/callback", callbackHandler)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Redirect root to the callback path which will then trigger the auth flow.
			http.Redirect(w, r, "/v1/auth/digitalocean/callback", http.StatusTemporaryRedirect)
		})

		log.Println("Starting server on " + thisEndpoint)
		u, err := url.Parse(thisEndpoint)
		if err != nil {
			log.Fatalf("Invalid THIS_ENDPOINT URL: %v", err)
		}

		port := u.Port()
		if port == "" {
			port = "8080" // Default port if not specified in THIS_ENDPOINT
		}
		addr := ":" + port

		log.Printf("Server listening on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}
}

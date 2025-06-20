package main

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
	thisEndpoint              = getEnv("THIS_ENDPOINT", "http://localhost:8080")
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
	log.Printf("Successfully saved token for team %s to the system keyring.", teamUUID)
	return nil
}

// getTokenFromKeyring retrieves a token from the keyring for a given team UUID.
func getTokenFromKeyring(teamUUID string) (string, error) {
	secret, err := keyring.Get(keyringService, teamUUID)
	if err != nil {
		return "", fmt.Errorf("failed to get token from keyring: %w", err)
	}
	return secret, nil
}

// --- Handlers ---

// callbackHandler handles the OAuth2 callback.
// It uses a two-step process for the implicit grant flow.
func callbackHandler(w http.ResponseWriter, r *http.Request) {
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
        // Create a URL object from the current window location.
        const url = new URL(window.location.href);
        // The token is in the hash part of the URL (e.g., #access_token=...).
        // Create a URLSearchParams object from the hash, skipping the leading '#'.
        const searchParams = new URLSearchParams(url.hash.slice(1));

        // If the access_token exists in the params from the hash, process it.
        if (searchParams.has("access_token")) {
            // Set the URL's search string (query part) to the params from the hash.
            url.search = searchParams.toString();
            // Clear the hash from the URL.
            url.hash = "";
            // Replace the current URL in the browser's history with the new one.
            // This reloads the page, and the server can now read the token.
            window.location.replace(url.toString());
        } else {
            // If no token is found, an error likely occurred during authorization.
            document.body.innerHTML = "<h1>Error</h1><p>Authentication failed. No access token found in the URL.</p>";
        }
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
		log.Printf("Error getting team UUID: %v", err)
		return
	}

	// Save the token to the system keyring.
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
	// Run in "cli" mode to get the auth URL, or run as a server to handle the callback.
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		cliHandler()
	} else {
		// Default to running as an HTTP server.
		http.HandleFunc("/v1/auth/digitalocean/callback", callbackHandler)

		// Add a root handler that redirects to the DigitalOcean auth page to start the flow.
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// This handler should only respond to the root path.
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}

			authURL, err := createAuthorizeURL()
			if err != nil {
				http.Error(w, "Failed to create authorization URL", http.StatusInternalServerError)
				log.Printf("Error creating authorization URL: %v", err)
				return
			}

			// Perform a 307 Temporary Redirect to the authorization URL.
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
		})

		u, err := url.Parse(thisEndpoint)
		if err != nil {
			log.Fatalf("Invalid THIS_ENDPOINT URL: %v", err)
		}

		port := u.Port()
		if port == "" {
			port = "8080" // Default port if not specified in THIS_ENDPOINT
		}
		addr := ":" + port

		log.Printf("Server starting. Visit http://localhost%s to begin authentication.", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}
}

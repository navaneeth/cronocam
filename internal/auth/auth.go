package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// Scopes for Google Photos API
	// See: https://developers.google.com/photos/library/guides/authorization
	// We need:
	// - photoslibrary.appendonly: for uploading photos
	// - photoslibrary.readonly: for reading albums and photos
	// - photoslibrary: for full access including creating albums
	// - photoslibrary.sharing: for sharing photos
	photosScope        = "https://www.googleapis.com/auth/photoslibrary"
	photosAppendScope  = "https://www.googleapis.com/auth/photoslibrary.appendonly"
	photosReadScope    = "https://www.googleapis.com/auth/photoslibrary.readonly"
	photosSharingScope = "https://www.googleapis.com/auth/photoslibrary.sharing"
)

type Authenticator struct {
	config    *oauth2.Config
	tokenPath string
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	default: // "linux", "freebsd", etc.
		cmd = "xdg-open"
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func New(credentialsPath string) (*Authenticator, error) {
	// Check if file exists
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials file does not exist at %s - please create it first", credentialsPath)
	}

	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, photosScope, photosAppendScope, photosReadScope, photosSharingScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	// Set the redirect URL to our local server
	config.RedirectURL = "http://localhost:8080/callback"

	// Set token path in same directory as credentials
	tokenPath := filepath.Join(filepath.Dir(credentialsPath), "token.json")

	return &Authenticator{config: config, tokenPath: tokenPath}, nil
}

func (a *Authenticator) GetClient(ctx context.Context) (*http.Client, error) {
	tok, err := a.GetTokenFromFile()
	if err != nil {
		tok, err = a.getTokenFromWeb(ctx)
		if err != nil {
			return nil, err
		}
		if err := a.saveToken(tok); err != nil {
			return nil, err
		}
	}
	return a.config.Client(ctx, tok), nil
}

func (a *Authenticator) GetTokenFromFile() (*oauth2.Token, error) {
	f, err := os.Open(a.tokenPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func (a *Authenticator) getTokenFromWeb(ctx context.Context) (*oauth2.Token, error) {
	// Create a channel to receive the auth code
	codeChan := make(chan string)

	// Create an HTTP server for the callback
	server := &http.Server{Addr: ":8080"}

	// Handle the OAuth callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Error: no code in callback")
			codeChan <- ""
			return
		}

		// Send success message to browser
		fmt.Fprintf(w, "Authorization successful! You can close this window.")

		// Send the code to the waiting goroutine
		codeChan <- code

		// Shutdown the server
		go func() {
			server.Shutdown(ctx)
		}()
	})

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
			codeChan <- ""
		}
	}()

	// Generate the auth URL with the correct redirect URI
	authURL := a.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening the following link in your browser: \n%v\n", authURL)

	// Open the URL in the default browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\nPlease open the URL manually.\n", err)
	}

	// Wait for the code
	authCode := <-codeChan
	if authCode == "" {
		return nil, fmt.Errorf("failed to get authorization code")
	}

	// Exchange the code for a token
	tok, err := a.config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}

	return tok, nil
}

func (a *Authenticator) saveToken(token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(a.tokenPath), 0755); err != nil {
		return fmt.Errorf("unable to create token directory: %v", err)
	}
	f, err := os.OpenFile(a.tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

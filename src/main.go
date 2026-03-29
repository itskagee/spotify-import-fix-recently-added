package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jdcukier/spotify/v2"
	spotifyauth "github.com/jdcukier/spotify/v2/auth"
)

func main() {
	cfg, needsSaving, err := loadOrPromptConfig()
	if err != nil {
		exitWithError("Failed to load config: %v", err)
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%s/callback", cfg.Port)

	// 1. Authenticate
	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
			spotifyauth.ScopeUserReadPrivate,
		),
		spotifyauth.WithClientID(cfg.ClientID),
		spotifyauth.WithClientSecret(cfg.ClientSecret),
	)

	// Start a local server to handle the callback
	ch := make(chan *spotify.Client)

	// Generate a cryptographically secure random state
	state, err := generateState(16)
	if err != nil {
		exitWithError("Generating state: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(r.Context(), state, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			exitWithError("Auth: %v", err)
		}

		if st := r.FormValue("state"); st != state {
			http.NotFound(w, r)
			exitWithError("State mismatch: %s != %s\n", st, state)
		}

		// use the token to get an authenticated client
		client := spotify.New(auth.Client(r.Context(), token))
		fmt.Fprintf(w, "Login Completed! You can close this window.")
		ch <- client
	})

	// Create an explicit server for safety and control
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Assign the global cleanup hook so the server ALWAYS shuts down cleanly even during unexpected exits
	onExit = func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutown the local server gracefully
		if err := server.Shutdown(shutdownCtx); err != nil {
			if err != context.DeadlineExceeded { // don't care about timeout errors during shutdown, we're already exiting anyway
				fmt.Printf("Error shutting down server: %v\n", err)
			}
		}
	}

	// Channel to catch immediate startup errors
	serverError := make(chan error, 1)

	go func() {
		err := server.ListenAndServe()
		// Ignoring http.ErrServerClosed as it is an expected error upon shutdown
		if err != nil && err != http.ErrServerClosed {
			serverError <- err
		}
	}()

	// Wait briefly to see if the server fails to start (e.g., port already in use)
	select {
	case err := <-serverError:
		exitWithError("Could not start local server (is port %s in use? Try a different one): %v", cfg.Port, err)
	case <-time.After(1 * time.Second):
		// No error caught even after 1 second of waiting, assume the server started successfully
	}

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser.")
	fmt.Println("(make sure you are logging in to the account that has the playlists you want to fix)")
	fmt.Println()
	fmt.Println("Use CTRL + Left Click (Windows) or CMD + Double Left Click (Mac) to open the link:")
	fmt.Println()
	fmt.Println(url)

	// Wait for client
	client := <-ch

	ctx := context.Background()

	// 2. Get Current User
	user, err := client.CurrentUser(ctx)
	if err != nil {
		exitWithError("Could not get user info: %v", err)
	}

	// If the config was just created, save it for future runs
	if needsSaving {
		saveConfig(cfg)
	}

	fmt.Printf("\nLogged in as: %s\n", user.DisplayName)

	// 3. List Playlists
	fmt.Println("\nFetching playlists")
	playlists, err := fetchPlaylists(ctx, client)
	if err != nil {
		spotifyAPIError(err)
		exitWithError("Fetching Playlists: %v\n", err)
	}

	for i, p := range playlists {
		fmt.Printf("[%d] %s (Tracks: %d)\n", i+1, p.Name, p.Items.Total)
	}

	// 4. User Selection
	fmt.Print("\nEnter the numbers of the playlists you want to fix (comma separated, e.g. 1,3,7): ")
	scanner := bufio.NewScanner(os.Stdin)
	var input string
	if scanner.Scan() {
		input = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		exitWithError("Reading input: %v", err)
	}

	selectedIndices, err := parseInput(input, len(playlists))
	if err != nil {
		exitWithError("Input error: %v", err)
	}

	for _, index := range selectedIndices {
		originalPlaylist := playlists[index]

		err := processPlaylist(ctx, client, originalPlaylist)
		if err != nil {
			fmt.Println("Don't worry, your progress is safe. Re-run the script later.")
			spotifyAPIError(err)
			exitWithError("Processing Playlists: %v\n", err)
		}
	}

	fmt.Println("\nDone!")
	exitGracefully()
}

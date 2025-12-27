package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var (
	clientID     = os.Getenv("SPOTIFY_ID")
	clientSecret = os.Getenv("SPOTIFY_SECRET")
	redirectURI  = "http://127.0.0.1:8080/callback"
)

func main() {
	if clientID == "" || clientSecret == "" {
		log.Fatal("Please set SPOTIFY_ID and SPOTIFY_SECRET environment variables.")
	}

	// 1. Authenticate
	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
			spotifyauth.ScopeUserReadPrivate,
		),
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
	)

	// Start a local server to handle the callback
	ch := make(chan *spotify.Client)

	// Generate a cryptographically secure random state
	state, err := generateState(16)
	if err != nil {
		log.Fatalf("Fatal error generating state: %v", err)
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(r.Context(), state, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			log.Fatal(err)
		}

		if st := r.FormValue("state"); st != state {
			http.NotFound(w, r)
			log.Fatalf("State mismatch: %s != %s\n", st, state)
		}

		// use the token to get an authenticated client
		client := spotify.New(auth.Client(r.Context(), token), spotify.WithRetry(true))
		fmt.Fprintf(w, "Login Completed! You can close this window.")
		ch <- client
	})

	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:\n", url)

	// Wait for client
	client := <-ch

	ctx := context.Background()

	// 2. Get Current User
	user, err := client.CurrentUser(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nLogged in as: %s\n", user.DisplayName)

	// 3. List Playlists
	fmt.Println("\nFetching playlists")
	playlistPage, err := client.CurrentUsersPlaylists(ctx)
	if err != nil {
		log.Fatal(err)
	}

	playlists := playlistPage.Playlists

	for i, p := range playlists {
		fmt.Printf("[%d] %s (Tracks: %d)\n", i+1, p.Name, p.Tracks.Total)
	}

	// 4. User Selection
	fmt.Print("\nEnter the numbers of the playlists you want to fix (comma separated, e.g. 1,3,7): ")
	scanner := bufio.NewScanner(os.Stdin)
	var input string
	if scanner.Scan() {
		input = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading input: %v", err)
	}

	selectedIndices, err := parseInput(input, len(playlists))
	if err != nil {
		log.Fatalf("Input error: %v", err)
	}

	for _, index := range selectedIndices {
		originalPlaylist := playlists[index]
		processPlaylist(ctx, client, user.ID, originalPlaylist)
	}

	fmt.Println("\nDone!")
}

func processPlaylist(ctx context.Context, client *spotify.Client, userID string, p spotify.SimplePlaylist) {
	fmt.Printf("\nProcessing playlist: %s\n", p.Name)

	// A. Get all tracks
	var allTracks []spotify.ID
	offset := 0
	limit := 50
	for {
		opts := []spotify.RequestOption{spotify.Limit(limit), spotify.Offset(offset)}
		trackPage, err := client.GetPlaylistItems(ctx, p.ID, opts...)
		if err != nil {
			log.Printf("Error fetching tracks for %s: %v\n", p.Name, err)
			return
		}

		for _, item := range trackPage.Items {
			// Ensure it's a track and not an episode or empty
			if item.Track.Track != nil {
				allTracks = append(allTracks, item.Track.Track.ID)
			}
		}

		if len(trackPage.Items) < limit {
			break
		}

		offset += limit
	}

	if len(allTracks) == 0 {
		fmt.Println("No tracks found.")
		return
	}

	// B. Reverse the slice
	allTracks = reverseIDs(allTracks)

	// C. Create new playlist
	newName := fmt.Sprintf("%s Fixed", p.Name)
	newPlaylist, err := client.CreatePlaylistForUser(
		ctx,
		userID,
		newName,
		"Fixed copy of "+p.Name,
		p.IsPublic,
		p.Collaborative,
	)
	if err != nil {
		log.Printf("Error creating playlist %s: %v\n", newName, err)
		return
	}

	fmt.Printf("Created new playlist: %s\n", newName)

	// D. Add tracks one by one with delay
	total := len(allTracks)
	fmt.Printf("Starting transfer of %d tracks (this will take about %d seconds)...\n", total, total)

	for i, trackID := range allTracks {
		_, err := client.AddTracksToPlaylist(ctx, newPlaylist.ID, trackID)
		if err != nil {
			log.Printf("Failed to add track %s to playlist %s: %v\n", trackID, newPlaylist.Name, err)
			fmt.Printf("\nFailed to add track %s\n", trackID)
		}

		// Progress updates
		remaining := total - (i + 1)

		// Use \r to return to the start of the line
		// Overwrite the line with the new count
		// Add trailing spaces "   " to clean up any leftover characters from wider numbers (e.g., going from 100 to 99)
		fmt.Printf("\rTracks remaining: %d   ", remaining)

		// Sleep for 1 second to ensure "Recently Added" works
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nPlaylist processing complete.")
}

// generateState creates a cryptographically secure random string of n bytes
func generateState(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	// Return as a URL-safe base64 string
	return base64.URLEncoding.EncodeToString(b), nil
}

// reverseIDs takes a slice of Spotify IDs and returns a new slice in reverse order.
func reverseIDs(input []spotify.ID) []spotify.ID {
	if len(input) == 0 {
		return input
	}

	output := make([]spotify.ID, len(input))
	copy(output, input)

	for i, j := 0, len(output)-1; i < j; i, j = i+1, j-1 {
		output[i], output[j] = output[j], output[i]
	}

	return output
}

// parseInput parses and validates the user input for playlist selection.
// Returns a slice of 0-based indices.
func parseInput(input string, totalPlaylists int) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("No input provided")
	}

	parts := strings.Split(input, ",")
	var indices []int

	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		var val int
		_, err := fmt.Sscanf(s, "%d", &val)
		if err != nil {
			return nil, fmt.Errorf("Invalid number: %s", s)
		}

		// User sees 1-based list, we convert it back to 0-based
		if val < 1 || val > totalPlaylists {
			return nil, fmt.Errorf("Number out of range: %d", val)
		}

		indices = append(indices, val-1)
	}

	return indices, nil
}

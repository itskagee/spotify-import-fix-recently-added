package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/zmb3/spotify/v2"
)

// --- UNIT TESTS FOR LOGIC ---

func TestReverseIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []spotify.ID
		expected []spotify.ID
	}{
		{
			name:     "Reverse 3 IDs",
			input:    []spotify.ID{"1", "2", "3"},
			expected: []spotify.ID{"3", "2", "1"},
		},
		{
			name:     "Reverse 1 ID",
			input:    []spotify.ID{"1"},
			expected: []spotify.ID{"1"},
		},
		{
			name:     "Reverse Empty",
			input:    []spotify.ID{},
			expected: []spotify.ID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reverseIDs(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ReverseIDs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		total     int
		expected  []int
		expectErr bool
	}{
		{"Valid Single", "1", 5, []int{0}, false},
		{"Valid Multiple", "1, 3", 5, []int{0, 2}, false},
		{"Valid Unordered", "3, 1", 5, []int{2, 0}, false},
		{"Out of Bounds High", "6", 5, nil, true},
		{"Out of Bounds Low", "0", 5, nil, true},
		{"Garbage Input", "abc", 5, nil, true},
		{"Empty Input", "", 5, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInput(tt.input, tt.total)
			if (err != nil) != tt.expectErr {
				t.Errorf("ParseInput() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !tt.expectErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseInput() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// --- MOCK INTEGRATION TEST ---

// TestProcessPlaylistFlow verifies that we can fetch tracks, reverse them,
// create a playlist, and ADD them one by one.
func TestProcessPlaylistFlow(t *testing.T) {
	// 1. Setup a Mock Spotify Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// MOCK: Get Playlist Items
		if r.Method == "GET" && r.URL.Path == "/playlists/old-playlist/tracks" {
			w.WriteHeader(http.StatusOK)
			// Return 2 tracks
			// Using Raw JSON string to guarantee correct JSON structure.
			// There were some Go struct nesting issues in the mock otherwise.
			fmt.Fprintln(w, `{
							"items": [
								{
									"track": {
										"id": "track-A",
										"name": "Track A",
										"type": "track",
										"uri": "spotify:track:track-A"
									}
								},
								{
									"track": {
										"id": "track-B",
										"name": "Track B",
										"type": "track",
										"uri": "spotify:track:track-B"
									}
								}
							],
							"total": 2,
							"limit": 100,
							"offset": 0
						}`)

			return
		}

		// MOCK: Create Playlist
		if r.Method == "POST" && r.URL.Path == "/users/test-user/playlists" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(spotify.FullPlaylist{
				SimplePlaylist: spotify.SimplePlaylist{ID: spotify.ID("new-playlist")},
			})

			return
		}

		// MOCK: Add Tracks
		if r.Method == "POST" && r.URL.Path == "/playlists/new-playlist/tracks" {
			w.WriteHeader(http.StatusCreated)
			// Mimic the JSON response {"snapshot_id": "..."}
			json.NewEncoder(w).Encode(map[string]string{
				"snapshot_id": "random-snapshot-id",
			})

			return
		}

		// Fallback for unexpected calls
		t.Logf("Unexpected call: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))

	defer mockServer.Close()

	// 2. Configure Client to use Mock Server
	client := spotify.New(mockServer.Client(), spotify.WithBaseURL(mockServer.URL+"/"))

	// 3. Define the data to test
	originalPlaylist := spotify.SimplePlaylist{
		ID:   spotify.ID("old-playlist"),
		Name: "My Jam",
	}

	ctx := context.Background()

	// A. Get Tracks
	trackPage, err := client.GetPlaylistItems(ctx, originalPlaylist.ID)
	if err != nil {
		t.Fatalf("Failed to get playlist items: %v", err)
	}

	var tracks []spotify.ID
	for _, item := range trackPage.Items {
		if item.Track.Track != nil {
			tracks = append(tracks, item.Track.Track.ID)
		}
	}

	// Verify we got the tracks
	if len(tracks) != 2 {
		t.Fatalf("Expected 2 tracks, got %d", len(tracks))
	}

	// B. Reverse Logic
	tracks = reverseIDs(tracks)

	// Verify reversal (compare against spotify.ID type)
	if tracks[0] != spotify.ID("track-B") {
		t.Errorf("Expected reversal! First track should be track-B, got %s", tracks[0])
	}

	// C. Create Playlist
	newPl, err := client.CreatePlaylistForUser(ctx, "test-user", "New Name", "Desc", false, false)
	if err != nil {
		t.Fatalf("Failed to create playlist: %v", err)
	}

	// D. Add Tracks Loop
	for _, trackID := range tracks {
		_, err := client.AddTracksToPlaylist(ctx, newPl.ID, trackID)
		if err != nil {
			t.Errorf("Failed to add track %s: %v", trackID, err)
		}
	}
}

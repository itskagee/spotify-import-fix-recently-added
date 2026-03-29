package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jdcukier/spotify/v2"
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

func TestGenerateState(t *testing.T) {
	// Test 1: Does it successfully generate a string without error?
	state1, err := generateState(16)
	if err != nil {
		t.Fatalf("Failed to generate state: %v", err)
	}

	// Test 2: Is the length correct?
	// 16 bytes of raw data becomes 24 characters when Base64 encoded.
	// Formula: 4 * math.Ceil(n / 3) -> 4 * 6 = 24
	if len(state1) != 24 {
		t.Errorf("Expected Base64 string of length 24, got %d", len(state1))
	}

	// Test 3: Is it actually random?
	state2, err := generateState(16)
	if err != nil {
		t.Fatalf("Failed to generate second state: %v", err)
	}

	if state1 == state2 {
		t.Errorf("Security Flaw: generateState produced identical strings: %s", state1)
	}
}

// --- UNIT TEST FOR SAVE STATE ---

func TestSaveAndLoadState(t *testing.T) {
	// Create a safe, temporary directory for the test file
	tempDir := t.TempDir()
	testFileName := filepath.Join(tempDir, ".progress_test_playlist.json")

	// Define the exact state we expect to be saved
	expectedState := SyncState{
		Phase:          "adding",
		NewPlaylistID:  spotify.ID("new-playlist-123"),
		ReversedTracks: []spotify.ID{"track-C", "track-B", "track-A"},
		CurrentIndex:   2,
	}

	// Run the saveState function (function being tested)
	err := saveState(testFileName, expectedState)
	if err != nil {
		t.Fatalf("saveState failed unexpectedly: %v", err)
	}

	// Mimic the loading logic from main.go to verify the file was written correctly
	data, err := os.ReadFile(testFileName)
	if err != nil {
		t.Fatalf("saveState failed to write the file or it cannot be read: %v", err)
	}

	var loadedState SyncState
	err = json.Unmarshal(data, &loadedState)
	if err != nil {
		t.Fatalf("saveState wrote invalid JSON: %v", err)
	}

	// Compare what we loaded with what we saved
	if !reflect.DeepEqual(expectedState, loadedState) {
		t.Errorf("Loaded state does not match saved state.\nGot: %+v\nExpected: %+v", loadedState, expectedState)
	}
}

func TestSaveStateErrorHandling(t *testing.T) {
	// We pass a deliberately illegal file path to force an error.
	// A path containing null bytes (\x00) is universally rejected by file systems.
	invalidFileName := "invalid\x00file.json"

	state := SyncState{
		Phase: "fetching",
	}

	err := saveState(invalidFileName, state)

	if err == nil {
		t.Errorf("Expected saveState to return an error for an invalid file path, but got nil")
	}
}

// --- MOCK INTEGRATION TESTS ---

// TestFetchPlaylistsPagination verifies that we loop correctly to fetch all pages of a user's playlists.
func TestFetchPlaylistsPagination(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// MOCK: Get Current User's Playlists
		if r.Method == "GET" && r.URL.Path == "/me/playlists" {
			w.WriteHeader(http.StatusOK)

			offset := r.URL.Query().Get("offset")
			if offset == "" || offset == "0" {
				// PAGE 1: Return item 1 and provide a "Next" URL
				fmt.Fprintf(w, `{
					"items": [{"id": "playlist-1", "name": "My First Playlist"}],
					"total": 2,
					"limit": 1,
					"offset": 0,
					"next": "http://%s/me/playlists?offset=1&limit=1"
				}`, r.Host)
			} else {
				// PAGE 2: Return item 2 and set "Next" to null
				fmt.Fprintf(w, `{
					"items": [{"id": "playlist-2", "name": "My Second Playlist"}],
					"total": 2,
					"limit": 1,
					"offset": 1,
					"next": null
				}`)
			}
			return
		}

		http.NotFound(w, r)
	}))

	defer mockServer.Close()

	client := spotify.New(mockServer.Client(), spotify.WithBaseURL(mockServer.URL+"/"))
	ctx := context.Background()

	playlists, err := fetchPlaylists(ctx, client)
	if err != nil {
		t.Fatalf("Failed to fetch playlists: %v", err)
	}

	if len(playlists) != 2 {
		t.Fatalf("Expected exactly 2 playlists after pagination, got %d", len(playlists))
	}

	if playlists[0].ID != "playlist-1" || playlists[1].ID != "playlist-2" {
		t.Errorf("Fetched playlists are not in the expected order or missing.")
	}
}

// TestProcessPlaylistFlow verifies that we can fetch tracks, reverse them,
// create a playlist, and ADD them one by one.
func TestProcessPlaylistFlow(t *testing.T) {
	// 1. Sandbox Environment
	// We change the working directory to a temporary folder so the test
	// doesn't write/delete files in our actual project directory.
	tempDir := t.TempDir()
	originalWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWD) // Restore working directory after test

	// 2. Setup a Mock Spotify Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// MOCK: Get Playlist Items
		if r.Method == "GET" && r.URL.Path == "/playlists/old-playlist/items" {
			w.WriteHeader(http.StatusOK)
			// Using Raw JSON string to guarantee correct JSON structure.
			fmt.Fprintln(w, `{
				"items": [
					{
						"item": {
							"id": "track-A",
							"name": "Track A",
							"type": "track",
							"uri": "spotify:track:track-A"
						}
					},
					{
						"item": {
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
		if r.Method == "POST" && r.URL.Path == "/me/playlists" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(spotify.FullPlaylist{
				SimplePlaylist: spotify.SimplePlaylist{ID: spotify.ID("new-playlist")},
			})
			return
		}

		// MOCK: Add Tracks
		if r.Method == "POST" && r.URL.Path == "/playlists/new-playlist/items" {
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

	// 3. Configure Client to use Mock Server
	client := spotify.New(mockServer.Client(), spotify.WithBaseURL(mockServer.URL+"/"))
	ctx := context.Background()

	// 4. Define the data to test
	originalPlaylist := spotify.SimplePlaylist{
		ID:   spotify.ID("old-playlist"),
		Name: "Test Jam",
	}

	// 5. Run the ACTUAL function
	// This will hit the mocks, create a temporary .progress file,
	// sleep for 2 seconds total (1s per track), delete the file, and return nil.
	err := processPlaylist(ctx, client, originalPlaylist)
	if err != nil {
		t.Fatalf("processPlaylist failed unexpectedly: %v", err)
	}
}

// TestProcessPlaylistResumeFlow verifies that if a progress file exists,
// the script skips fetching/creating and resumes adding exactly where it left off.
func TestProcessPlaylistResumeFlow(t *testing.T) {
	// 1. Sandbox Environment
	// We change the working directory to a temporary folder so the test
	// doesn't write/delete files in our actual project directory.
	tempDir := t.TempDir()
	originalWD, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(originalWD) // Restore working directory after test

	originalPlaylistID := spotify.ID("test-resume-playlist")
	stateFile := fmt.Sprintf(".progress_%s.json", originalPlaylistID)

	expectedNewPlaylistID := spotify.ID("new-resumed-playlist")

	// 2. Create a fake pre-existing state file
	// We pretend the playlist has 3 tracks, and track-3 was already successfully added.
	initialState := SyncState{
		Phase:          "adding",
		NewPlaylistID:  expectedNewPlaylistID,
		ReversedTracks: []spotify.ID{"track-3", "track-2", "track-1"},
		CurrentIndex:   1,
	}

	err := saveState(stateFile, initialState)
	if err != nil {
		t.Fatalf("Failed to create initial state file for mock test: %v", err)
	}

	// We will use this slice to track which tracks the script actually tries to add
	var addedTracks []string

	// 3. Setup Mock Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// MOCK: Add Tracks
		if r.Method == "POST" && r.URL.Path == fmt.Sprintf("/playlists/%s/items", expectedNewPlaylistID) {

			// Decode the JSON request body to see which track URI it is trying to add
			var payload struct {
				URIs []string `json:"uris"`
			}
			json.NewDecoder(r.Body).Decode(&payload)
			if len(payload.URIs) > 0 {
				addedTracks = append(addedTracks, payload.URIs[0])
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"snapshot_id": "snap-resume"})
			return
		}

		// STRICT CHECKS: If the script tries to fetch tracks or create a playlist, FAIL the test.
		// It should be reading from the save file, not the API!
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/items") {
			t.Errorf("FAIL: Script attempted to fetch original tracks instead of using save state!")
		}
		if r.Method == "POST" && r.URL.Path == "/me/playlists" {
			t.Errorf("FAIL: Script attempted to create a new playlist instead of using the saved one!")
		}

		w.WriteHeader(http.StatusOK)
	}))

	defer mockServer.Close()

	client := spotify.New(mockServer.Client(), spotify.WithBaseURL(mockServer.URL+"/"))
	ctx := context.Background()

	// 4. Run the function
	p := spotify.SimplePlaylist{
		ID:   originalPlaylistID,
		Name: "Resume Test Jam",
	}

	err = processPlaylist(ctx, client, p)
	if err != nil {
		t.Fatalf("processPlaylist failed unexpectedly during resume: %v", err)
	}

	// 5. Verify Results
	// Check A: Did it only add the remaining 2 tracks?
	if len(addedTracks) != 2 {
		t.Fatalf("Expected exactly 2 tracks to be added on resume, got %d", len(addedTracks))
	}

	// Check B: Did it add the correct tracks in the correct order?
	expectedURI1 := "spotify:track:track-2"
	expectedURI2 := "spotify:track:track-1"
	if addedTracks[0] != expectedURI1 || addedTracks[1] != expectedURI2 {
		t.Errorf("Added wrong tracks or wrong order.\nExpected: [%s, %s]\nGot: %v", expectedURI1, expectedURI2, addedTracks)
	}

	// Check C: Did it clean up after itself?
	// The state file should be deleted upon 100% completion.
	if _, err := os.Stat(stateFile); !os.IsNotExist(err) {
		t.Errorf("State file was not deleted after successful completion")
	}
}

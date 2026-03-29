package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jdcukier/spotify/v2"
)

// SyncState keeps track of our progress in case of rate limits in two distinct phases: fetching and adding
type SyncState struct {
	Phase          string       `json:"phase"` // "fetching" or "adding"
	FetchOffset    int          `json:"fetch_offset"`
	OriginalTracks []spotify.ID `json:"original_tracks"`
	NewPlaylistID  spotify.ID   `json:"new_playlist_id"`
	ReversedTracks []spotify.ID `json:"reversed_tracks"`
	CurrentIndex   int          `json:"current_index"`
}

// saveState saves the current synchronization state to a JSON file for later resumption.
func saveState(filename string, state SyncState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("Could not format save data: %w", err)
	}

	// Write to a temporary file first
	tempFile := filename + ".tmp"
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		return fmt.Errorf("Could not write temporary save file: %w", err)
	}

	// Rename it to the real filename (atomic, prevents corruption)
	err = os.Rename(tempFile, filename)
	if err != nil {
		return fmt.Errorf("Could not finalize save file: %w", err)
	}

	return nil
}

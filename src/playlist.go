package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jdcukier/spotify/v2"
)

// fetchPlaylists retrieves all playlists of the current user, handling pagination if necessary.
func fetchPlaylists(ctx context.Context, client *spotify.Client) ([]spotify.SimplePlaylist, error) {
	var playlists []spotify.SimplePlaylist

	playlistPage, err := client.CurrentUsersPlaylists(ctx)
	if err != nil {
		return nil, err
	}

	playlists = append(playlists, playlistPage.Playlists...)

	for playlistPage.Next != "" {
		err = client.NextPage(ctx, playlistPage)
		if err != nil {
			return nil, err
		}

		playlists = append(playlists, playlistPage.Playlists...)
	}

	return playlists, nil
}

// processPlaylist handles the logic of fetching tracks, creating a new playlist, and transferring tracks while saving progress.
func processPlaylist(ctx context.Context, client *spotify.Client, p spotify.SimplePlaylist) error {
	fmt.Printf("\nProcessing playlist: %s\n", p.Name)

	stateFile := fmt.Sprintf(".progress_%s.json", p.ID)

	// Default state
	state := SyncState{
		Phase: "fetching",
	}

	// 1. Try to load existing progress
	if data, err := os.ReadFile(stateFile); err == nil {
		if err := json.Unmarshal(data, &state); err == nil {
			switch state.Phase {
			case "fetching":
				fmt.Printf(">> Found saved progress! Resuming fetch from offset %d.\n", state.FetchOffset)
			case "adding":
				fmt.Printf(">> Found saved progress! Resuming track additions from %d.\n", state.CurrentIndex+1)
			default:
				return fmt.Errorf("Unknown state.Phase: %s", state.Phase)
			}
		}
	}

	// 2. Fetching Tracks
	if state.Phase == "fetching" {
		limit := 50
		for {
			opts := []spotify.RequestOption{spotify.Limit(limit), spotify.Offset(state.FetchOffset)}
			trackPage, err := client.GetPlaylistItems(ctx, p.ID, opts...)
			if err != nil {
				fmt.Printf("\n[!] ERROR fetching tracks at offset %d.\n", state.FetchOffset)
				return err // Halts and allows for resume later
			}

			for _, item := range trackPage.Items {
				// Ignore Local MP3 files
				if item.IsLocal {
					continue
				}

				// Ensure it's a track, not a podcast episode
				if item.Item.Track != nil {
					state.OriginalTracks = append(state.OriginalTracks, item.Item.Track.ID)
				}
			}

			state.FetchOffset += limit
			// Save progress after EVERY page fetch
			if err := saveState(stateFile, state); err != nil {
				return fmt.Errorf("failed to save state during fetch: %w", err)
			}

			// UI Update for fetching
			fmt.Printf("\rFetched %d valid tracks", len(state.OriginalTracks))

			if len(trackPage.Items) < limit {
				break
			}
		}

		fmt.Println()

		if len(state.OriginalTracks) == 0 {
			fmt.Println("No playable tracks found (or they were all local MP3s/Podcasts).")
			if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
				// no error because the sync has technically succeeded, but print a warning for manual intervention
				fmt.Printf("\n[!] Warning: Could not delete progress file. Error: %v.\nYou may need to manually delete '%s' (make sure your file explorer shows hidden files).\n", err, stateFile)
			}

			return nil
		}

		// Transition State to Adding Phase
		fmt.Println("Reversing tracks and creating new playlist")
		state.ReversedTracks = reverseIDs(state.OriginalTracks)
		newName := fmt.Sprintf("%s Fixed", p.Name)
		newPlaylist, err := client.CreatePlaylist(ctx, newName, "Fixed copy of "+p.Name, p.IsPublic, p.Collaborative)
		if err != nil {
			fmt.Printf("Error creating playlist %s\n", newName)
			return err
		}

		fmt.Printf("Created new playlist: %s\n", newName)

		state.NewPlaylistID = newPlaylist.ID
		state.Phase = "adding"
		state.CurrentIndex = 0
		state.OriginalTracks = nil // Free up memory, we don't need this array anymore
		if err := saveState(stateFile, state); err != nil {
			return fmt.Errorf("failed to save state during transition: %w", err)
		}
	}

	// 3. Adding Tracks
	if state.Phase == "adding" {
		const secondsToWaitBetweenAdding = 1
		total := len(state.ReversedTracks)
		remainingAtStart := total - state.CurrentIndex
		fmt.Printf("Transferring %d tracks, should be done in about %s\n", remainingAtStart, time.Duration(remainingAtStart)*secondsToWaitBetweenAdding*time.Second)

		for i := state.CurrentIndex; i < total; i++ {
			trackID := state.ReversedTracks[i]
			_, err := client.AddTracksToPlaylist(ctx, state.NewPlaylistID, trackID)
			if err != nil {
				fmt.Printf("\n[!] ERROR: Failed to add track.\n")
				return err
			}

			// Save state after successful addition
			state.CurrentIndex = i + 1
			if err := saveState(stateFile, state); err != nil {
				return fmt.Errorf("failed to save state after adding track: %w", err)
			}

			// UI update
			remaining := total - state.CurrentIndex
			fmt.Printf("\rTracks remaining: %d      ", remaining) // extra space for clearing longer previous messages

			time.Sleep(secondsToWaitBetweenAdding * time.Second)
		}
	}

	// 4. Cleanup on success
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		// no error because the sync has technically succeeded, but print a warning for manual intervention
		fmt.Printf("\n[!] Warning: Could not delete progress file. Error: %v.\nYou may need to manually delete '%s' (make sure your file explorer shows hidden files).\n", err, stateFile)
	}

	fmt.Println("\nPlaylist processing complete.")
	return nil
}

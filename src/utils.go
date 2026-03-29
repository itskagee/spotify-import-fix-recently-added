package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jdcukier/spotify/v2"
)

// Cleanup on Exit: If the user exits the program while a local server is running, we want to make sure to shut it down gracefully.
var onExit func()

// generateState creates a random string of the specified length using crypto/rand for secure randomness.
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
//
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

		val, err := strconv.Atoi(s)
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

// spotifyAPIError checks whether the error is a Spotify API error and prints relevant information, especially for rate limits.
func spotifyAPIError(err error) {
	spotErr, ok := errors.AsType[spotify.Error](err)
	if !ok {
		return
	}

	fmt.Println("\nSpotify API Error.")

	if spotErr.RetryAfter.IsZero() {
		fmt.Println("Spotify did not specify a time limit to wait for. Please wait for 24 hours before proceeding to be on the safe side.")
		fmt.Println("Note the time logged below for reference.")
		return
	}

	fmt.Println("You are probably being rate-limited.")
	waitDuration := time.Until(spotErr.RetryAfter).Round(time.Second)
	fmt.Printf("API HTTP Status Code: %v\n", spotErr.Status)

	// Print the wait duration (e.g., "45m30s")
	fmt.Printf("Time to wait: %v\n", waitDuration)

	// Print the exact clock time to resume (e.g., "3:15PM")
	fmt.Printf("Safe to resume at: %v\n", spotErr.RetryAfter.Format(time.Kitchen))
}

// exitWithError prints a formatted error, waits for the user to press Enter, and exits.
func exitWithError(format string, a ...any) {
	fmt.Println()
	log.Printf("\n[!] ERROR: "+format, a...)

	// gracefully shutdown the local server if it's running
	if onExit != nil {
		onExit()
	}

	fmt.Println("\nNote all the errors above. Then press Enter to exit.")
	bufio.NewScanner(os.Stdin).Scan()
	os.Exit(1)
}

// exitGracefully waits for the user to press Enter before closing the application successfully.
func exitGracefully() {
	fmt.Println()

	// Wipe the configuration file after the script finishes successfully
	fmt.Println("Cleaning up saved credentials.")
	err := os.Remove(configFileName)
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("[!] Warning: Could not remove config file. Error: %v\nPlease delete '%s' manually if it exists (make sure your file explorer shows hidden files).", err, configFileName)
	}

	// gracefully shutdown the local server if it's running
	if onExit != nil {
		onExit()
	}

	fmt.Println("\nPress Enter to close this window.")
	bufio.NewScanner(os.Stdin).Scan()
	os.Exit(0)
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config stores the user's inputs
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Port         string `json:"port"`
}

const configFileName = ".spotify-fixer-config.json"

// loadOrPromptConfig tries to load the configuration from disk, and if it fails, prompts the user for input.
//
// Returns the configuration, a boolean indicating whether the config needs to be saved to disk, and an error if something went wrong.
func loadOrPromptConfig() (Config, bool, error) {
	var cfg Config

	// Try to load existing config
	if data, err := os.ReadFile(configFileName); err == nil {
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.Port != "" {
				return cfg, false, nil
			}
		}
	}

	// If no config exists (or it's broken), prompt the user
	// Get Port
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter port for local server (press enter for default: 8080): ")
	cfg.Port = "8080"
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			cfg.Port = input
		}
	}

	fmt.Println()
	fmt.Println("If you have changed the port above, make sure to change it in the Spotify developer dashboard as well, under Redirect URIs.")
	fmt.Println("And if you're using Docker, change the ports in the 'compose.yml' file as well.")
	fmt.Println("Use CTRL + C (Windows) or CMD + . (Mac) to exit. Re-run the script once you've made the changes.")

	// Get Spotify API Credentials
	fmt.Println("\nPlease paste your app credentials below (Right Click (Windows) or Right Click > Paste (Mac))")

	fmt.Print("Spotify Client ID: ")
	if scanner.Scan() {
		cfg.ClientID = strings.TrimSpace(scanner.Text())
	}

	fmt.Print("Spotify Client Secret: ")
	if scanner.Scan() {
		cfg.ClientSecret = strings.TrimSpace(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return Config{}, false, fmt.Errorf("Reading configuration input: %w", err)
	}

	fmt.Println() // extra line break for better formatting

	return cfg, true, nil
}

// saveConfig writes the verified credentials to the disk
func saveConfig(cfg Config) {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Printf("\n[!] Warning: Could not format config file for future runs: %v\n", err)
		return
	}

	// We use 0600 permissions so ONLY the current computer user can read/write this file
	err = os.WriteFile(configFileName, data, 0600)
	if err != nil {
		fmt.Printf("\n[!] Warning: Could not save config file for future runs: %v\n", err)
	} else {
		fmt.Println("\n[+] Configuration verified and saved locally! You won't have to enter it again.")
	}
}

# Spotify Import Fix Recently Added
This script fixes the random listing of items in playlists that happens when songs are imported to Spotify using services like TuneMyMusic or Soundiiz.

## Note
This script was made and tested on Windows.
Should be fine for Linux as well. 
Not sure about MacOS, but I don't think there should be any issues, especially if using Docker.

## What it does, and how
This script takes one or more playlists present in your account and makes a copy of those with items that maintain the existing ordering when sorted by recently added.

It does this by first scanning the existing playlist, then adding the songs to a new playlist one by one in reverse order with a 1 second delay in between. While this makes the process a bit long for huge playlists, it is the only way we can be assured that the sorting works as intended.

Currently there is no provision for saving state, so if your internet dies, or you experience a power cut, just delete the playlist copy made by the script from your account and run the script again. Since it does not modify the existing playlist, the process is deterministic. I have not tested it for idempotence (although I think Spotify allows for playlists with duplicate names, so the script should be idempotent as well).

I have tested this with playlists up to 700 items long and it works fine.

## Prerequisites
- Install one of the following on your system:
    - Docker (easier)
        - Can be either Docker Desktop (easier) or Docker CE.
    - Golang
- Get the Spotify Client ID and Token for your app:
    - Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard/).
    - Create an app and get your Client ID and Client Secret.
        - Copy them over for later, but make sure to remove all references when done. These can be harmful in the wrong hands.
    - Add http://127.0.0.1:8080/callback as a Redirect URI in your app settings.

## How to Run
### Using Docker

- Clone the repository, or download/copy the contents of `main.go`, `go.mod`, `go.sum`, `compose.yml`, and `Dockerfile` into a folder of your choice.
- Make a new file in the folder with the name `.env`
- Edit the file, paste the following into it, and save:
    ```
        SPOTIFY_ID=paste_your_client_id_from_earlier_here
        SPOTIFY_SECRET=paste_your_client_secret_from_earlier_here
    ```
- Using any Terminal, `cd` into the folder (or open the folder and right click -> 'Open in Terminal', if available).
- Run this command: `docker compose run --build --service-ports --rm spotify-import-fix-recently-added`
    - Click Allow/Accept on any Firewall requests that pop-up. It is required for the script to connect to the internet.
- Let the script run (it can take a while to download and run everything depending on the speed of your internet).
- Follow the instructions displayed. DO NOT CLOSE THE SCRIPT UNTIL IT FINISHES.
- The docker container will automatically stop and remove itself once the script is done running.
- Once you are done, you can delete this script and uninstall Docker if you don't need it anymore. You can also delete the app you created in the Spotify Developer Dashboard.
- You can now delete the old playlists in your account and rename the new ones to whatever you want.

### Using Golang
- Clone the repository, or download/copy the contents of `main.go`, `go.mod`, and `go.sum` into a folder of your choice.
- Do one of the following:
    - Change `clientID` and `clientSecret` in the script:
        - Open `main.go`.
        - In Lines 20 and 21,  change:
            - `clientID = os.Getenv("SPOTIFY_ID")` to `clientID = "your_client_id_from_earlier"`
            - `clientSecret = os.Getenv("SPOTIFY_SECRET")` to `clientSecret = "your_client_secret_from_earlier"`
        - Save and Exit.
    - If you're not comfortable with changing the script file, then you can set Environment Variables as follows using your Terminal:
        - Mac/Linux: `export SPOTIFY_ID="your_client_id_from_earlier" && export SPOTIFY_SECRET="your_client_secret_from_earlier"`
        - Windows (PowerShell): `$env:SPOTIFY_ID="your_client_id_from_earlier"; $env:SPOTIFY_SECRET="your_client_secret_from_earlier"`
- Open a Terminal and `cd` to the folder with the script files (or open the folder and right click -> 'Open in Terminal', if available).
- Run this command: `go run main.go`
    - Click Allow/Accept on any Firewall requests that pop-up. It is required for the script to connect to the internet.
- Let the script run (it can take a while for stuff to download and run depending on the speed of your internet).
- Follow the instructions displayed. DO NOT CLOSE THE SCRIPT UNTIL IT FINISHES.
- Once you are done, you can delete the script and its folder if you don't need it anymore. You can also delete the app you created in the Spotify Developer Dashboard.
- You can now delete the old playlists in your account and rename the new ones to whatever you want.

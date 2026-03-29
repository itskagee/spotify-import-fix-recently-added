# Spotify Import Fix Recently Added
This script fixes the random listing of items in playlists that happens when songs are imported to Spotify using services like TuneMyMusic or Soundiiz.

## Note
This script was made and tested on Windows, but should run fine on other systems as well.

## What it does, and How
This script takes one or more playlists present in your account and makes a copy of those with items that maintain the existing ordering when sorted by recently added.

It does this by first scanning the existing playlist, then adding the songs to a new playlist one by one in reverse order with a 1 second delay in between. While this makes the process a bit long for huge playlists, it is the only way we can be assured that the sorting works as intended.

The script saves its current working state whenever there is any kind of error. So if it crashes, hits an API limit, your internet dies, or whatever happens, just re-run it (make sure to note the error it shows though). 

The script does not modify your existing playlist, and makes a note of the new one it creates, so this re-run is deterministic and idempotent.

For playlists up to 700 items long, the script works in a single go. For playlists longer than 750-800 items, you may run into Spotify's daily API limits. Just wait 24h (or however long the script tells you), and start the script back up to continue from where you left off.

Unfortunately there is no way to bypass this limit except to wait.

## Prerequisites
- Download the relevant binary for your system from the 'Releases' tab.
- Get the Spotify Client ID and Secret for your app:
    - Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard/).
    - Create an app and get your `Client ID` and `Client Secret`.
        - Copy them over for later, but make sure to remove all references when done. These can be harmful in the wrong hands.
    - Add `http://127.0.0.1:8080/callback` as the Redirect URI in your app settings.
        - If port `8080` is already in use on your system, make sure to change it before adding it.
        - The script will ask you for your choice when it starts up, don't worry.
        - To check the availability of the port, open your 'Terminal' and enter these commands (they can take a while to run, be patient):
            - Windows: `netstat -a | findstr 8080` 
            - Linux: `sudo ss -tlnp | grep :8080`
            - Mac: `sudo lsof -i :8080`
        - If you see an output, then the port is in use. Choose a different one (and please test it again).
- (Optional / Advanced) Install one of the following on your system:
    - Docker (easier)
        - Can be either Docker Desktop (easier) or Docker CE.
    - Golang
    - This step is useful if you're not comfortable with executing binaries which you haven't compiled yourself. That said, the release binaries are built using Github Actions, which you can verify, so they won't be running any code that isn't already in the repo.

## How to Run
### Using the Downloaded Binaries
- Make a new folder and copy the downloaded binary over to it.
    - This is because the script creates additional files that you might want to isolate (and delete later if not automatically done).
- Double click on the binary to run it.
- Click Allow/Accept on any Firewall requests that pop-up. It is required for the script to connect to the internet.
- Follow the instructions displayed. DO NOT CLOSE THE SCRIPT UNTIL IT ASKS YOU TO EXIT.
- Once you are done, you can delete the folder you created (and everything in it) if you don't need it anymore. You can also delete the app you created in the Spotify Developer Dashboard.
    - DO NOT DELETE ANYTHING IF THE SCRIPT SAYS IT ENCOUNTERED AN ERROR.
    - If you do that, you'll delete the saved state as well!
    - Let the script print "Done!" without showing any errors. Then and only then you can be assured that it completed successfully and you can proceed with the deletions.
- If the script completed successfully, you can now delete the old playlists in your account and rename the new ones to whatever you want.

### Using Docker

- Clone the repository, or download/copy the contents of the `src` folder, the `docker` folder, `go.mod` and `go.sum` into a new folder.
- Make sure to open the `compose.yml` file and change the Time Zone.
    - Also change the port number if it wasn't available when you checked it earlier.
- Using any Terminal, `cd` into the newly created folder (or open the folder and right click -> 'Open in Terminal', if available).
- Run this command: `docker compose -f ./docker/compose.yml run --build --service-ports --rm spotify-import-fix-recently-added`
    - Click Allow/Accept on any Firewall requests that pop-up. It is required for the script to connect to the internet.
- Let the script run (it can take a while to download and run everything depending on the speed of your internet).
- Follow the instructions displayed. DO NOT CLOSE THE SCRIPT UNTIL IT ASKS YOU TO EXIT.
- The docker container will automatically stop and remove itself once the script is done running.
- Once you are done, you can delete the folder (and everything in it) and uninstall Docker if you don't need it anymore. You can also delete the app you created in the Spotify Developer Dashboard.
    - DO NOT DELETE ANYTHING IF THE SCRIPT SAYS IT ENCOUNTERED AN ERROR.
    - If you do that, you'll delete the saved state as well!
    - Let the script print "Done!" without showing any errors. Then and only then you can be assured that it completed successfully and you can proceed with the deletions.
- If the script completed successfully, you can now delete the old playlists in your account and rename the new ones to whatever you want.

### Using Golang
- Clone the repository, or download/copy the contents of the `src` folder, `go.mod`, and `go.sum` into a new folder.
- Open a Terminal and `cd` into the newly created folder (or open the folder and right click -> 'Open in Terminal', if available).
- Run this command: `go run ./src`
    - Click Allow/Accept on any Firewall requests that pop-up. It is required for the script to connect to the internet.
- Let the script run (it can take a while for stuff to download and run depending on the speed of your internet).
- Follow the instructions displayed. DO NOT CLOSE THE SCRIPT UNTIL IT ASKS YOU TO EXIT.
- Once you are done, you can delete the folder (and everything in it) if you don't need it anymore. You can also delete the app you created in the Spotify Developer Dashboard.
    - DO NOT DELETE ANYTHING IF THE SCRIPT SAYS IT ENCOUNTERED AN ERROR.
    - If you do that, you'll delete the saved state as well!
    - Let the script print "Done!" without showing any errors. Then and only then you can be assured that it completed successfully and you can proceed with the deletions.
- If the script completed successfully, you can now delete the old playlists in your account and rename the new ones to whatever you want.

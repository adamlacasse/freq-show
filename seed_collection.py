import json
import time
import urllib.request
import urllib.parse
import sys
import ssl

# Paths and URLs
SYNC_JSON_PATH = "../record-collection/sync_progress.json"
MB_SEARCH_URL = "https://musicbrainz.org/ws/2/release-group/?query={}&fmt=json"
BASE_URL = sys.argv[1].rstrip('/') if len(sys.argv) > 1 else "http://localhost:8080"
FREQSHOW_ALBUM_URL = f"{BASE_URL}/albums/{{}}"
FREQSHOW_COLLECTION_URL = f"{BASE_URL}/collections/adam/albums/{{}}"
USER_AGENT = "FreqShowSeeder/1.0 ( adamlacasse@example.com )"

def main():
    try:
        with open(SYNC_JSON_PATH, "r") as f:
            data = json.load(f)
    except Exception as e:
        print(f"Failed to read sync progress: {e}")
        return

    albums_to_add = []
    for key, val in data.items():
        if val.get("status") == "added":
            albums_to_add.append(key)
    
    print(f"Found {len(albums_to_add)} albums to seed.")

    added_count = 0
    not_found_count = 0
    error_count = 0

    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE

    for idx, key in enumerate(albums_to_add):
        print(f"[{idx+1}/{len(albums_to_add)}] Processing '{key}'...")
        
        # Parse artist and title (format is usually "Artist - Title")
        parts = key.split(" - ", 1)
        if len(parts) == 2:
            artist, title = parts[0], parts[1]
        else:
            artist, title = "", key

        # 1. Search MusicBrainz for the Release Group
        query = f'release-group:"{title}"'
        if artist:
            query += f' AND artist:"{artist}"'
            
        search_url = MB_SEARCH_URL.format(urllib.parse.quote(query))
        
        req = urllib.request.Request(search_url, headers={"User-Agent": USER_AGENT})
        
        mbid = None
        try:
            time.sleep(1.2) # Respect MB rate limit (1 req / sec)
            with urllib.request.urlopen(req, context=ctx) as response:
                resp_data = json.loads(response.read().decode())
                rgs = resp_data.get("release-groups", [])
                if rgs:
                    # Prefer album/ep primary types if possible, else just take the first one
                    best_rg = rgs[0]
                    for rg in rgs:
                        if rg.get("primary-type") in ["Album", "EP"]:
                            best_rg = rg
                            break
                    mbid = best_rg.get("id")
        except Exception as e:
            print(f"  -> Error searching MusicBrainz: {e}")
            error_count += 1
            continue

        if not mbid:
            print("  -> Not found in MusicBrainz.")
            not_found_count += 1
            continue

        # 2. Fetch the album from Freq Show to cache it
        req = urllib.request.Request(FREQSHOW_ALBUM_URL.format(mbid))
        try:
            with urllib.request.urlopen(req, context=ctx) as response:
                pass # Success
        except Exception as e:
            print(f"  -> Error caching album in Freq Show: {e}")
            # we'll continue anyway, maybe it's just a track fetch failure
            
        # 3. Add to Collection
        data_to_send = json.dumps({"format": "Vinyl"}).encode('utf-8')
        req = urllib.request.Request(
            FREQSHOW_COLLECTION_URL.format(mbid), 
            data=data_to_send, 
            headers={'Content-Type': 'application/json'},
            method='POST'
        )
        try:
            with urllib.request.urlopen(req, context=ctx) as response:
                added_count += 1
                print(f"  -> Added {mbid} to collection!")
        except Exception as e:
            print(f"  -> Error adding to collection: {e}")
            error_count += 1

    print("\nSeeding Complete!")
    print(f"Successfully added: {added_count}")
    print(f"Not found in MB: {not_found_count}")
    print(f"Errors: {error_count}")

if __name__ == "__main__":
    main()

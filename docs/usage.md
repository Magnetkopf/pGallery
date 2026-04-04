# pGallery Documentation

## Commands

### 1. Sync

Download artworks from your Pixiv bookmarks.

~~~bash
pGallery sync -user <userid> -cookie <cookiefile> -base <dir> [-downloader <type>]
~~~

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-user` | Yes | - | The Pixiv user ID whose bookmarks to sync |
| `-cookie` | Yes | `cookie.txt` | Path to the cookie file |
| `-base` | No | `downloads` | Base directory to save artworks |
| `-downloader` | No | - | You can choose `aria2c` |

**Getting your Cookie:**
1. Log in to Pixiv in your browser
2. Open Developer Tools (F12)
3. Go to Application/Storage → Cookies → pixiv.net
4. Copy the cookie value

**Getting your User ID:**
Your user ID is the number in your Pixiv profile URL:
`https://www.pixiv.net/users/<USER_ID>`

**Directory Structure:**
~~~
<base>/
├── <artist_id>/
│   ├── artist.yaml      # Artist metadata
│   ├── folder.jpg       # Artist pfp
│   └── <artwork_id>/
│       ├── artwork.yaml # Artwork metadata
│       ├── folder.jpg  # Artwork thumbnail
│       ├── p0.jpg      # First page
│       ├── p1.jpg      # Second page (if multi-page)
│       └── ...
├── downloaded.json     # Download record
└── index.json          # Built index
~~~

---

### 2. Build

Build the search index from downloaded artworks.

~~~bash
pGallery build -base <dir>
~~~

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-base` | No | `downloads` | Base directory to scan |

This command scans the base directory and creates an `index.json` file containing:
- All artworks with their metadata
- Tag index for filtering
- Artist index for browsing

---

### 3. WebUI

Start the web interface to browse your gallery.

~~~bash
pGallery webui -base <dir> -port <port>
~~~

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-base` | No | `downloads` | Base directory containing artworks |
| `-port` | No | `8080` | Port to listen on |

**Features:**
- Browse all artworks
- Filter by artist
- Filter by tag
- View artwork details and metadata

---

### 4. Check
Validate the downloaded artwork folders against the metadata in `downloaded.json`.

~~~bash
pGallery check -base <directory>
~~~ 

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `-base` | Yes | `downloads` | Base directory containing artworks |

The command scans each artwork ID recorded in `downloaded.json`, reads its
`artwork.yaml` to get the expected page count, and verifies that the required
files exist:
* `folder.*` – thumbnail of the artwork
* `p0.*`, `p1.*`, …, `p{pages‑1}.*` – each page image
If any file is missing, the entire artwork folder is removed and the ID is
deleted from `downloaded.json`. After the scan, a refreshed `downloaded.json`
containing only the valid IDs is written.

---

## Quick Start

1. **Sync your bookmarks:**
   ~~~bash
   ./pGallery sync -user 12345678 -cookie cookie.txt -base my_gallery
   ~~~

2. **Build the index:**
   ~~~bash
   ./pGallery build -base my_gallery
   ~~~

3. **Start the web UI:**
   ~~~bash
   ./pGallery webui -base my_gallery -port 8080
   ~~~

4. Open http://localhost:8080 in your browser

---

## Workflow

### First Time Setup

1. Get your Pixiv cookie and user ID
2. Run `sync` to download all bookmarked artworks
3. Run `build` to create the search index
4. Run `webui` to start browsing

### Regular Usage

1. Run `sync` again to download new bookmarks (skips already downloaded)
2. Run `build` to update the index
3. Run `webui` to start browsing


---

## Troubleshooting

**Sync fails with API error:**
- Your cookie may have expired - refresh it and try again

**Web UI shows no artworks:**
- Make sure you've run `build` after syncing
- Check that `index.json` exists in your base directory

**Download speed is slow:**
- Switch downloader 🤓

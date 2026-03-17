![pGallery](images/pgallery.svg)

Self-hosted pixiv gallery with built-in sync.

- 🌐 Web UI
- 🔭 Filter by artist and tag
- 📄 Standalone metadata(yaml)
- 🪼 Friendly for Jellyfin
- 📁 No database required

## Usage

Read [📖 Docs](docs/usage.md)

~~~bash
#Sync bookmark
./pGallery sync -user <userid> -cookie <cookiefile> -base <dir>

#Build index
./pGallery build -base <dir>

#Start WebUI
./pGallery webui -base <dir> -port <port>
~~~

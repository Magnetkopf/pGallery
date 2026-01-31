# pGallery

Self-hosted pixiv gallery with built-in sync.

- 🪼 Friendly for Jellyfin
- 📄 Standalone metadata(yaml)
- 📁 No database required 

## Usage

~~~bash
#Sync bookmark
./pGallery sync -user <userid> -cookie <cookiefile> -base <dir>

#Build index
./pGallery build -base <dir>
~~~

# m3u_gen_acestream

## What is this?

> M3U playlist generator for Ace Stream.

## How it works?

It fetches available channels from ace stream engine using API, filters and transforms them and writes
M3U playlists using templates defined in config.

## Requirements

[Ace Stream](https://docs.acestream.net/products/) running locally or remotely.

## Command line flags

| Command argument     | Description                                                                               |
| -------------------- | ----------------------------------------------------------------------------------------- |
| -h, --help           | Print help message                                                                        |
| -v, --version        | Print the program version                                                                 |
| -u, --update         | Check for updates and update                                                              |
| -l, --logLevel       | Logging level. Can be from `1` (most verbose) to `7` (least verbose) [default: `3`]       |
| -f, --logFile        | Log file. If set, writes structured log to a file at the specified path                   |
| -c, --cfgPath        | Config file path to read from or initialize a default [default: `m3u_gen_acestream.yaml`] |

Unless `--cfgPath` is specified, on first run it creates default config in current directory and terminates.
Tweak it to suit your needs and start the program again.

## Downloads

See [releases page](https://github.com/SCP002/m3u_gen_acestream/releases)

## Default config

```yaml
# Ace Stream Engine address in format of host:port.
engineAddr: 127.0.0.1:6878
#
# Playlists to generate.
playlists:
#
# MPEG-TS format, alive.
# Change any non-default category to 'other'.
- #
  # Destination filepath to write playlist to.
  outputPath: ./out/playlist_mpegts_all.m3u8
  #
  # Template for the header of M3U file.
  headerTemplate: |
    #EXTM3U url-tvg="http://epg.one/epg2.xml.gz" tvg-shift=0 deinterlace=1 m3uautoload=1
  #
  # Template for each channel. Available variables are:
  # {{.Name}}
  # {{.Infohash}}
  # {{.Categories}}
  # {{.Countries}}
  # {{.Languages}}
  # {{.EngineAddr}}
  # {{.TVGName}}
  # {{.IconURL}}
  entryTemplate: |
    #EXTINF:-1 group-title="{{.Categories}}",{{.Name}}
    http://{{.EngineAddr}}/ace/getstream?infohash={{.Infohash}}
  #
  # Change categories by category regular expressions (keys) to strings (values).
  # Use "^$" regular expression to match unset categories.
  # Example:
  # categoryRxToCategoryMap:
  #   "^category regexp A$": "becomes category B"
  #   "^category regexp C$": "becomes category D"
  categoryRxToCategoryMap:
    ^(?!.*(informational|entertaining|educational|movies|documentaries|sport|fashion|music|regional|ethnic|religion|teleshop|erotic_18_plus|other_18_plus|cyber_games|amateur|webcam)).*: other
  #
  # Set categories by name regular expressions (keys) to list of strings (values).
  # Example:
  # nameRxToCategoriesMap:
  #   "^name regexp A$":
  #   - "will have category B"
  #   - "and category C"
  #   "^name regexp D$":
  #   - "will have category E"
  #   - "and category F"
  nameRxToCategoriesMap: {}
  #
  # Only keep channels which name matches any of these regular expressions.
  # Example:
  # nameRxFilter:
  # - ".*keep channels matching name A.*"
  # - ".*keep channels matching name B.*"
  nameRxFilter: []
  #
  # Remove channels which name matches any of these regular expressions.
  # Example:
  # nameRxBlacklist:
  # - ".*remove channels matching name A.*"
  # - ".*remove channels matching name B.*"
  nameRxBlacklist: []
  #
  # Only keep channels which category equals to any of these.
  # See https://docs.acestream.net/developers/knowledge-base/list-of-categories/
  # for known (but not all possible) categories list.
  # Use empty string to include results with unset category.
  # Example:
  # categoriesFilter:
  # - "keep channels with category A"
  # - "keep channels with category B"
  categoriesFilter: []
  #
  # If true, only keep channels with categories that are in filter, but not any other.
  categoriesFilterStrict: false
  #
  # Remove channels which category equals to any of these.
  # See https://docs.acestream.net/developers/knowledge-base/list-of-categories/
  # for known (but not all possible) categories list.
  # Use empty string to exclude results with unset category.
  # Example:
  # categoriesBlacklist:
  # - "remove channels with category A"
  # - "remove channels with category B"
  categoriesBlacklist: []
  #
  # Only keep channels which language equals to any of these.
  # Languages are 3-character, lower case strings, such as 'eng', 'rus' etc.
  # Use empty string to include results with unset language.
  # Example:
  # languagesFilter:
  # - "keep channels with language A"
  # - "keep channels with language B"
  languagesFilter: []
  #
  # If true, only keep channels with languages that are in filter, but not any other.
  languagesFilterStrict: false
  #
  # Remove channels which language equals to any of these.
  # Languages are 3-character, lower case strings, such as 'eng', 'rus' etc.
  # Use empty string to exclude results with unset language.
  # Example:
  # languagesBlacklist:
  # - "remove channels with language A"
  # - "remove channels with language B"
  languagesBlacklist: []
  #
  # Only keep channels which country equals to any of these.
  # Countries are 2-character, lower case strings, such as 'us', 'ru' etc. and 'int' for international.
  # Use empty string to include results with unset country.
  # Example:
  # countriesFilter:
  # - "keep channels with country A"
  # - "keep channels with country B"
  countriesFilter: []
  #
  # If true, only keep channels with countries that are in filter, but not any other.
  countriesFilterStrict: false
  #
  # Remove channels which country equals to any of these.
  # Countries are 2-character, lower case strings, such as 'us', 'ru' etc. and 'int' for international.
  # Use empty string to exclude results with unset country.
  # Example:
  # countriesBlacklist:
  # - "remove channels with country A"
  # - "remove channels with country B"
  countriesBlacklist: []
  #
  # Only keep channels which status equals to any of these.
  # Can be 1 (no guaranty that channel is working) or 2 (channel is available).
  # Example:
  # statusFilter:
  # - 1
  # - 2
  statusFilter:
  - 2
  #
  # Only keep channels which availability equals to or more than this.
  # Can be between 0.0 (zero availability) and 1.0 (full availability).
  # The lower this value is, the less channels gets removed.
  availabilityThreshold: 1.0
  #
  # Only keep channels which availability was updated that much time ago or sooner.
  # The lower this value is, the more channels gets removed.
  availabilityUpdatedThreshold: 36h0m0s
  #
  # Remove sources that does not respond with any content.
  removeDeadSources: false
  #
  # Try to read TS packets when removing dead sources.
  useMpegTsAnalyzer: false
  #
  # Timeout for reading Ace Stream Engine response when removing dead sources.
  checkRespTimeout: 20s
#
# HLS format, alive, only keep tv, music and empty category.
# Change category 'tv' to 'television' and empty category to 'unknown'.
- outputPath: ./out/playlist_hls_tv_+_music_+_no_category.m3u8
  headerTemplate: |
    #EXTM3U url-tvg="http://epg.one/epg2.xml.gz" tvg-shift=0 deinterlace=1 m3uautoload=1
  entryTemplate: |
    #EXTINF:-1 group-title="{{.Categories}}",{{.Name}}
    http://{{.EngineAddr}}/ace/manifest.m3u8?infohash={{.Infohash}}
  categoryRxToCategoryMap:
    (?i)^tv$: television
    ^$: unknown
  nameRxToCategoriesMap: {}
  nameRxFilter: []
  nameRxBlacklist: []
  categoriesFilter:
  - tv
  - music
  - unknown
  categoriesFilterStrict: false
  categoriesBlacklist: []
  languagesFilter: []
  languagesFilterStrict: false
  languagesBlacklist: []
  countriesFilter: []
  countriesFilterStrict: false
  countriesBlacklist: []
  statusFilter:
  - 2
  availabilityThreshold: 1.0
  availabilityUpdatedThreshold: 36h0m0s
  removeDeadSources: false
  useMpegTsAnalyzer: false
  checkRespTimeout: 20s
#
# https://github.com/pepsik-kiev/HTTPAceProxy format, alive, all but erotic channels.
- outputPath: ./out/playlist_httpaceproxy_all_but_porn.m3u8
  headerTemplate: |
    #EXTM3U url-tvg="http://epg.one/epg2.xml.gz" tvg-shift=0 deinterlace=1 m3uautoload=1
  entryTemplate: |
    #EXTINF:-1 group-title="{{.Categories}}",{{.Name}}
    http://127.0.0.1:8000/infohash/{{.Infohash}}/stream.mp4
  categoryRxToCategoryMap: {}
  nameRxToCategoriesMap: {}
  nameRxFilter: []
  nameRxBlacklist:
  - (?i).*erotic.*
  - (?i).*porn.*
  - '(?i).*18\+.*'
  categoriesFilter: []
  categoriesFilterStrict: false
  categoriesBlacklist:
  - erotic_18_plus
  - 18+
  languagesFilter: []
  languagesFilterStrict: false
  languagesBlacklist: []
  countriesFilter: []
  countriesFilterStrict: false
  countriesBlacklist: []
  statusFilter:
  - 2
  availabilityThreshold: 1.0
  availabilityUpdatedThreshold: 36h0m0s
  removeDeadSources: false
  useMpegTsAnalyzer: false
  checkRespTimeout: 20s
```

## Build from source code [Go / Golang]

1. Install [Golang](https://golang.org/) 1.24 or newer.

2. Download the source code:  

    ```sh
    git clone https://github.com/SCP002/m3u_gen_acestream.git
    ```

3. Install dependencies:

    ```sh
    cd src
    go mod tidy
    ```

    Or

    ```sh
    cd src
    go get ./...
    ```

4. Update dependencies (optional):

    ```sh
    go get -u ./...
    ```

5. To build a binary for current OS / architecture into `../build/` folder:

    ```sh
    go build -o ../build/ m3u_gen_acestream.go
    ```

    Or run `/src/build.sh` to build binaries for every OS / architecture pair.

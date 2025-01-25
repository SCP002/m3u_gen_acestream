package config

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-yaml"
	"github.com/samber/lo"

	"m3u_gen_acestream/util/logger"
)

// Config represents program configuration.
type Config struct {
	EngineAddr string     `yaml:"engineAddr"`
	Playlists  []Playlist `yaml:"playlists"`
}

// Playlist represents set of parameters for M3U playlist generation such as output path, template and filter criterias.
type Playlist struct {
	OutputPath                   string            `yaml:"outputPath"`
	HeaderTemplate               template.Template `yaml:"headerTemplate"`
	EntryTemplate                template.Template `yaml:"entryTemplate"`
	NameRegexpFilter             regexp.Regexp     `yaml:"nameRegexpFilter"`
	CategoriesFilter             []string          `yaml:"categoriesFilter"`
	LanguagesFilter              []string          `yaml:"languagesFilter"`
	CountriesFilter              []string          `yaml:"countriesFilter"`
	StatusFilter                 []int             `yaml:"statusFilter"`
	AvailabilityThreshold        float32           `yaml:"availabilityThreshold"`
	AvailabilityUpdatedThreshold time.Duration     `yaml:"availabilityUpdatedThreshold"`
}

// Init returns config instance and false if config at `filePath` already exist.
//
// If config does not exist, creates a default, returns empty instance and true.
func Init(log *logger.Logger, filePath string) (*Config, bool, error) {
	log.Info("Reading config")

	var cfg Config
	defCfg, defCommentMap := newDefCfg()

	readConfig := func() error {
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		return yaml.UnmarshalWithOptions(bytes, &cfg,
			yaml.CustomUnmarshaler(func(t *regexp.Regexp, b []byte) error {
				rx, err := regexp.Compile(string(b))
				*t = *rx
				return err
			}),
			yaml.CustomUnmarshaler(func(t *template.Template, b []byte) error {
				lines := strings.Split(string(b), "\n")
				if lines[0] == ">" || lines[0] == "|" {
					lines = lines[1:]
				}
				lines = lo.Map(lines, func(line string, _ int) string {
					return strings.TrimPrefix(line, "  ")
				})
				chunk := strings.Join(lines, "\n")
				templ, err := t.Parse(chunk)
				*t = *templ
				return err
			}),
		)
	}

	writeDefConfig := func() error {
		bytes, err := yaml.MarshalWithOptions(defCfg, yaml.WithComment(defCommentMap),
			yaml.CustomMarshaler(func(t regexp.Regexp) ([]byte, error) {
				return []byte(t.String()), nil
			}),
			yaml.CustomMarshaler(func(t template.Template) ([]byte, error) {
				lines := strings.Split(t.Root.String(), "\n")
				lines = lo.Map(lines, func(line string, _ int) string {
					return "  " + line
				})
				chunk := strings.Join(lines, "\n")
				return []byte(">\n" + chunk), nil
			}),
		)
		if err != nil {
			return err
		}
		return os.WriteFile(filePath, bytes, 0644)
	}

	// Read config or create a new if not exist.
	if err := readConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Info("Config file not found, creating a default")
			if err := writeDefConfig(); err != nil {
				return &cfg, false, errors.Wrap(err, "Write default config")
			}
			return &cfg, true, nil
		} else {
			return &cfg, false, errors.Wrap(err, "Read config")
		}
	}

	return &cfg, false, nil
}

// newDefCfg returns new default config and comment map
func newDefCfg() (*Config, yaml.CommentMap) {
	headerLine := `#EXTM3U url-tvg="https://iptvx.one/epg/epg.xml.gz" tvg-shift=0 deinterlace=1 m3uautoload=1`
	entryLine1 := `#EXTINF:-1 group-title="{{.Categories}}" tvg-name="{{.TVGName}}" tvg-logo="{{.TVGLogo}}",{{.Name}}`
	entryMpegtsLink := `http://{{.EngineAddr}}/ace/getstream?infohash={{.Infohash}}`
	entryHlsLink := `http://{{.EngineAddr}}/ace/manifest.m3u8?infohash={{.Infohash}}`
	entryHttpAceProxyLink := `http://127.0.0.1:8000/pid/{{.Infohash}}/stream.mp4`

	headerTemplate := template.Must(template.New("headerTemplate").Parse(headerLine))
	mpegTsTemplate := template.Must(template.New("mpegTsTemplate").Parse(entryLine1 + "\n" + entryMpegtsLink))
	hlsTemplate := template.Must(template.New("hlsTemplate").Parse(entryLine1 + "\n" + entryHlsLink))
	httpAceProxyTemplate := template.Must(template.New("httpAceProxy").Parse(entryLine1 + "\n" + entryHttpAceProxyLink))

	cfg := &Config{
		EngineAddr: "127.0.0.1:6878",
		Playlists: []Playlist{
			{
				OutputPath:                   "./out/playlist_all_mpegts.m3u8",
				HeaderTemplate:               *headerTemplate,
				EntryTemplate:                *mpegTsTemplate,
				NameRegexpFilter:             *regexp.MustCompile(".*"),
				CategoriesFilter:             []string{},
				LanguagesFilter:              []string{},
				CountriesFilter:              []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        0.8,
				AvailabilityUpdatedThreshold: time.Hour * 24 * 8,
			},
			{
				OutputPath:                   "./out/playlist_tv_and_music_hls.m3u8",
				HeaderTemplate:               *headerTemplate,
				EntryTemplate:                *hlsTemplate,
				NameRegexpFilter:             *regexp.MustCompile(".*"),
				CategoriesFilter:             []string{"tv", "music"},
				LanguagesFilter:              []string{},
				CountriesFilter:              []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        0.8,
				AvailabilityUpdatedThreshold: time.Hour * 24 * 8,
			},
			{
				OutputPath:                   "./out/playlist_fm_httpaceproxy.m3u8",
				HeaderTemplate:               *headerTemplate,
				EntryTemplate:                *httpAceProxyTemplate,
				NameRegexpFilter:             *regexp.MustCompile(".* FM$"),
				CategoriesFilter:             []string{},
				LanguagesFilter:              []string{},
				CountriesFilter:              []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        0.8,
				AvailabilityUpdatedThreshold: time.Hour * 24 * 8,
			},
		},
	}

	commentMap := yaml.CommentMap{
		"$.engineAddr": []*yaml.Comment{
			yaml.HeadComment(" Acestream engine address in format of host:port."),
		},
		"$.playlists": []*yaml.Comment{
			yaml.HeadComment("", " Playlists to generate."),
		},
		"$.playlists[0]": []*yaml.Comment{
			yaml.HeadComment("", " MPEG-TS format, no filtering by name, category or country."),
		},
		"$.playlists[0].outputPath": []*yaml.Comment{
			yaml.HeadComment("", " Destination filepath to write playlist to."),
		},
		"$.playlists[0].headerTemplate": []*yaml.Comment{
			yaml.HeadComment("", " Template for the first line of M3U file."),
		},
		"$.playlists[0].entryTemplate": []*yaml.Comment{
			yaml.HeadComment("", " Template for each channel."),
		},
		"$.playlists[0].nameRegexpFilter": []*yaml.Comment{
			yaml.HeadComment("", " Only keep channels which name matches this regular expression."),
		},
		"$.playlists[0].categoriesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which category equals to any of these.",
				" See https://docs.acestream.net/developers/knowledge-base/list-of-categories/ for categories list",
			),
		},
		"$.playlists[0].languagesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which language equals to any of these.",
				" Languages are 3-character, lower case strings, such as 'eng', 'rus' etc.",
				" Use empty string to include results with unset language.",
			),
		},
		"$.playlists[0].countriesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which country equals to any of these.",
				" Countries are 2-character, lower case strings, such as 'us', 'ru' etc. and 'int' for international.",
				" Use empty string to include results with unset country.",
			),
		},
		"$.playlists[0].statusFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which status equals to any of these.",
				" Can be 1 (no guaranty that channel is working) or 2 (channel is available).",
			),
		},
		"$.playlists[0].availabilityThreshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which availability equals to or more than this.",
				" Can be between 0.0 (zero availability) and 1.0 (full availability).",
			),
		},
		"$.playlists[0].availabilityUpdatedThreshold": []*yaml.Comment{
			yaml.HeadComment("", " Only keep channels which availability was updated that much time ago or sooner."),
		},
		"$.playlists[1]": []*yaml.Comment{
			yaml.HeadComment("", " HLS format, only keep tv and music category."),
		},
		"$.playlists[2]": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" https://github.com/pepsik-kiev/HTTPAceProxy format, only keep channels which names ends with ' FM'.",
			),
		},
	}

	return cfg, commentMap
}

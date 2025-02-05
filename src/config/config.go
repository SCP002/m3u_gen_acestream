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

// BlockStr represents YAML block string.
type BlockStr string

// Playlist represents set of parameters for M3U playlist generation such as output path, template and filter criterias.
type Playlist struct {
	OutputPath                   string             `yaml:"outputPath"`
	HeaderTemplate               BlockStr           `yaml:"headerTemplate"`
	EntryTemplate                *template.Template `yaml:"entryTemplate"`
	NameRegexpFilter             []*regexp.Regexp   `yaml:"nameRegexpFilter"`
	NameRegexpBlacklist          []*regexp.Regexp   `yaml:"nameRegexpBlacklist"`
	CategoriesFilter             []string           `yaml:"categoriesFilter"`
	CategoriesBlacklist          []string           `yaml:"categoriesBlacklist"`
	LanguagesFilter              []string           `yaml:"languagesFilter"`
	CountriesFilter              []string           `yaml:"countriesFilter"`
	StatusFilter                 []int              `yaml:"statusFilter"`
	AvailabilityThreshold        float64            `yaml:"availabilityThreshold"`
	AvailabilityUpdatedThreshold time.Duration      `yaml:"availabilityUpdatedThreshold"`
	// TODO: Add category mapping
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

		blockBytesToString := func(b []byte) string {
			lines := strings.Split(string(b), "\n")
			if lines[0] == ">" || lines[0] == "|" {
				lines = lines[1:]
			}
			lines = lo.Map(lines, func(line string, _ int) string {
				return strings.TrimPrefix(line, "    ")
			})
			return strings.Join(lines, "\n")
		}

		err = yaml.UnmarshalWithOptions(bytes, &cfg,
			yaml.CustomUnmarshaler(func(t *regexp.Regexp, b []byte) error {
				rx, err := regexp.Compile(string(b))
				*t = *rx
				return err
			}),
			yaml.CustomUnmarshaler(func(t *template.Template, b []byte) error {
				templ, err := t.Parse(blockBytesToString(b))
				*t = *templ
				return err
			}),
			yaml.CustomUnmarshaler(func(t *BlockStr, b []byte) error {
				*t = BlockStr(blockBytesToString(b))
				return err
			}),
		)

		return errors.Wrap(err, "Decode config file")
	}

	writeDefConfig := func() error {
		stringToBlockBytes := func(s string) []byte {
			lines := strings.Split(s, "\n")
			lines = lo.Map(lines, func(line string, _ int) string {
				return "  " + line
			})
			chunk := strings.Join(lines, "\n")
			return []byte("|\n" + chunk)
		}

		bytes, err := yaml.MarshalWithOptions(defCfg, yaml.WithComment(defCommentMap),
			yaml.CustomMarshaler(func(t regexp.Regexp) ([]byte, error) {
				return []byte(t.String()), nil
			}),
			yaml.CustomMarshaler(func(t template.Template) ([]byte, error) {
				return stringToBlockBytes(t.Root.String()), nil
			}),
			yaml.CustomMarshaler(func(t BlockStr) ([]byte, error) {
				return stringToBlockBytes(string(t)), nil
			}),
		)
		if err != nil {
			return errors.Wrap(err, "Encode config file")
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

// newDefCfg returns new default config and comment map.
func newDefCfg() (*Config, yaml.CommentMap) {
	headerLine := BlockStr(`#EXTM3U url-tvg="https://iptvx.one/epg/epg.xml.gz" tvg-shift=0 deinterlace=1 m3uautoload=1`)
	entryLine1 := `#EXTINF:-1 group-title="{{.Categories}}",{{.Name}}`
	entryMpegtsLink := `http://{{.EngineAddr}}/ace/getstream?infohash={{.Infohash}}`
	entryHlsLink := `http://{{.EngineAddr}}/ace/manifest.m3u8?infohash={{.Infohash}}`
	entryHttpAceProxyLink := `http://127.0.0.1:8000/pid/{{.Infohash}}/stream.mp4`

	mpegTsTemplate := template.Must(template.New("mpegTsTemplate").Parse(entryLine1 + "\n" + entryMpegtsLink))
	hlsTemplate := template.Must(template.New("hlsTemplate").Parse(entryLine1 + "\n" + entryHlsLink))
	httpAceProxyTemplate := template.Must(template.New("httpAceProxy").Parse(entryLine1 + "\n" + entryHttpAceProxyLink))

	regexpsAll := []*regexp.Regexp{regexp.MustCompile(`.*`)}
	regexpsPorn := []*regexp.Regexp{
		regexp.MustCompile(`(?i).*erotic.*`),
		regexp.MustCompile(`(?i).*porn.*`),
		regexp.MustCompile(`(?i).*18\+.*`),
	}

	cfg := &Config{
		EngineAddr: "127.0.0.1:6878",
		Playlists: []Playlist{
			{
				OutputPath:                   "./out/playlist_all_mpegts.m3u8",
				HeaderTemplate:               headerLine,
				EntryTemplate:                mpegTsTemplate,
				NameRegexpFilter:             regexpsAll,
				NameRegexpBlacklist:          []*regexp.Regexp{},
				CategoriesFilter:             []string{},
				CategoriesBlacklist:          []string{},
				LanguagesFilter:              []string{},
				CountriesFilter:              []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        0.8,
				AvailabilityUpdatedThreshold: time.Hour * 24 * 8,
			},
			{
				OutputPath:                   "./out/playlist_tv_and_music_hls.m3u8",
				HeaderTemplate:               headerLine,
				EntryTemplate:                hlsTemplate,
				NameRegexpFilter:             regexpsAll,
				NameRegexpBlacklist:          []*regexp.Regexp{},
				CategoriesFilter:             []string{"tv", "music"},
				CategoriesBlacklist:          []string{},
				LanguagesFilter:              []string{},
				CountriesFilter:              []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        0.8,
				AvailabilityUpdatedThreshold: time.Hour * 24 * 8,
			},
			{
				OutputPath:                   "./out/playlist_all_but_porn_httpaceproxy.m3u8",
				HeaderTemplate:               headerLine,
				EntryTemplate:                httpAceProxyTemplate,
				NameRegexpFilter:             regexpsAll,
				NameRegexpBlacklist:          regexpsPorn,
				CategoriesFilter:             []string{},
				CategoriesBlacklist:          []string{"erotic_18_plus", "18+"},
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
			yaml.HeadComment("", " Template for the header of M3U file."),
		},
		"$.playlists[0].entryTemplate": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Template for each channel.",
				" Available variables are:",
				" {{.Name}}",
				" {{.Infohash}}",
				" {{.Categories}}",
				" {{.EngineAddr}}",
				" {{.TVGName}}",
				" {{.IconURL}}",
			),
		},
		"$.playlists[0].nameRegexpFilter": []*yaml.Comment{
			yaml.HeadComment("", " Only keep channels which name matches any of these regular expressions."),
		},
		"$.playlists[0].nameRegexpBlacklist": []*yaml.Comment{
			yaml.HeadComment("", " Remove channels which name matches any of these regular expressions."),
		},
		"$.playlists[0].categoriesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which category equals to any of these.",
				" See https://docs.acestream.net/developers/knowledge-base/list-of-categories/",
				" for known (but not all possible) categories list.",
			),
		},
		"$.playlists[0].categoriesBlacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove channels which category equals to any of these.",
				" See https://docs.acestream.net/developers/knowledge-base/list-of-categories/",
				" for known (but not all possible) categories list.",
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
				" https://github.com/pepsik-kiev/HTTPAceProxy format, all but erotic channels.",
			),
		},
	}

	return cfg, commentMap
}

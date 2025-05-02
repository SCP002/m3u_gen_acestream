package config

import (
	"fmt"
	"io/fs"
	"os"
	"text/template"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dlclark/regexp2"
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
	OutputPath                   string              `yaml:"outputPath"`
	HeaderTemplate               string              `yaml:"headerTemplate"`
	EntryTemplate                string              `yaml:"entryTemplate"`
	CategoryRxToCategoryMap      map[string]string   `yaml:"categoryRxToCategoryMap"`
	NameRxToCategoriesMap        map[string][]string `yaml:"nameRxToCategoriesMap"`
	NameRxFilter                 []string            `yaml:"nameRxFilter"`
	NameRxBlacklist              []string            `yaml:"nameRxBlacklist"`
	CategoriesFilter             []string            `yaml:"categoriesFilter"`
	CategoriesFilterStrict       bool                `yaml:"categoriesFilterStrict"`
	CategoriesBlacklist          []string            `yaml:"categoriesBlacklist"`
	LanguagesFilter              []string            `yaml:"languagesFilter"`
	LanguagesFilterStrict        bool                `yaml:"languagesFilterStrict"`
	LanguagesBlacklist           []string            `yaml:"languagesBlacklist"`
	CountriesFilter              []string            `yaml:"countriesFilter"`
	CountriesFilterStrict        bool                `yaml:"countriesFilterStrict"`
	CountriesBlacklist           []string            `yaml:"countriesBlacklist"`
	StatusFilter                 []int               `yaml:"statusFilter"`
	AvailabilityThreshold        float64             `yaml:"availabilityThreshold"`
	AvailabilityUpdatedThreshold time.Duration       `yaml:"availabilityUpdatedThreshold"`
	RemoveDeadChannels           *bool               `yaml:"removeDeadChannels"`
	UseMpegTsAnalyzer            *bool               `yaml:"useMpegTsAnalyzer"`
	CheckRespTimeout             *time.Duration      `yaml:"checkRespTimeout"`
}

// Init returns config instance and false if config at `filePath` already exist.
//
// If config does not exist, creates a default, returns empty instance and true.
func Init(log *logger.Logger, filePath string) (*Config, bool, error) {
	log.Info("Reading config")

	var cfg Config
	commentMap := yaml.CommentMap{}
	defCfg, defCommentMap := newDefCfg()

	readConfig := func() error {
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		err = yaml.UnmarshalWithOptions(bytes, &cfg, yaml.CommentToMap(commentMap))
		return errors.Wrap(err, "Decode config file")
	}

	writeConfig := func(cfg *Config, comments yaml.CommentMap) error {
		bytes, err := yaml.MarshalWithOptions(cfg, yaml.WithComment(comments),
			yaml.UseLiteralStyleIfMultiline(true), yaml.UseSingleQuote(true))
		if err != nil {
			return errors.Wrap(err, "Encode config file")
		}
		return os.WriteFile(filePath, bytes, 0644)
	}

	validateConfig := func() error {
		for _, playlist := range cfg.Playlists {
			for rx := range playlist.CategoryRxToCategoryMap {
				if _, err := regexp2.Compile(rx, regexp2.RE2); err != nil {
					return errors.Wrapf(err, "Can not compile regular expression:\n%v\nin categoryRxToCategoryMap", rx)
				}
			}
			for rx := range playlist.NameRxToCategoriesMap {
				if _, err := regexp2.Compile(rx, regexp2.RE2); err != nil {
					return errors.Wrapf(err, "Can not compile regular expression:\n%v:\nin nameRxToCategoriesMap", rx)
				}
			}
			for _, rx := range playlist.NameRxFilter {
				if _, err := regexp2.Compile(rx, regexp2.RE2); err != nil {
					return errors.Wrapf(err, "Can not compile regular expression:\n%v\nin nameRxFilter", rx)
				}
			}
			for _, rx := range playlist.NameRxBlacklist {
				if _, err := regexp2.Compile(rx, regexp2.RE2); err != nil {
					return errors.Wrapf(err, "Can not compile regular expression:\n%v\nin nameRxBlacklist", rx)
				}
			}
			if _, err := template.New("").Parse(playlist.EntryTemplate); err != nil {
				return errors.Wrapf(err, "Can not parse template:\n%v\nin entryTemplate", playlist.EntryTemplate)
			}
		}
		return nil
	}

	addNewOprions := func() error {
		modified := false
		for idx, playlist := range cfg.Playlists {
			if playlist.RemoveDeadChannels == nil {
				defVal := lo.ToPtr(false)
				path := fmt.Sprintf("$.playlists[%v].removeDeadChannels", idx)
				log.InfoFi("Adding new config option", "path", path, "value", defVal, "playlist", playlist.OutputPath)
				cfg.Playlists[idx].RemoveDeadChannels = defVal
				commentMap[path] = defCommentMap[path]
				modified = true
			}
			if playlist.UseMpegTsAnalyzer == nil {
				defVal := lo.ToPtr(false)
				path := fmt.Sprintf("$.playlists[%v].useMpegTsAnalyzer", idx)
				log.InfoFi("Adding new config option", "path", path, "value", defVal, "playlist", playlist.OutputPath)
				cfg.Playlists[idx].UseMpegTsAnalyzer = defVal
				commentMap[path] = defCommentMap[path]
				modified = true
			}
			if playlist.CheckRespTimeout == nil {
				defVal := lo.ToPtr(time.Second * 20)
				path := fmt.Sprintf("$.playlists[%v].checkRespTimeout", idx)
				log.InfoFi("Adding new config option", "path", path, "value", defVal, "playlist", playlist.OutputPath)
				cfg.Playlists[idx].CheckRespTimeout = defVal
				commentMap[path] = defCommentMap[path]
				modified = true
			}
		}
		if modified {
			return errors.Wrap(writeConfig(&cfg, commentMap), "Write config")
		}
		return nil
	}

	// Read config or create a new if not exist.
	if err := readConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Info("Config file not found, creating a default")
			if err := writeConfig(defCfg, defCommentMap); err != nil {
				return &cfg, false, errors.Wrap(err, "Write default config")
			}
			return &cfg, true, nil
		} else {
			return &cfg, false, errors.Wrap(err, "Read config")
		}
	}

	if err := validateConfig(); err != nil {
		return &cfg, false, errors.Wrap(err, "Validate config")
	}

	if err := addNewOprions(); err != nil {
		return &cfg, false, errors.Wrap(err, "Add new options")
	}

	return &cfg, false, nil
}

// newDefCfg returns new default config and comment map.
func newDefCfg() (*Config, yaml.CommentMap) {
	headerLine := `#EXTM3U url-tvg="https://iptvx.one/epg/epg.xml.gz" tvg-shift=0 deinterlace=1 m3uautoload=1` + "\n"
	entryLine1 := `#EXTINF:-1 group-title="{{.Categories}}",{{.Name}}` + "\n"
	entryMpegtsLink := `http://{{.EngineAddr}}/ace/getstream?infohash={{.Infohash}}` + "\n"
	entryHlsLink := `http://{{.EngineAddr}}/ace/manifest.m3u8?infohash={{.Infohash}}` + "\n"
	entryHttpAceProxyLink := `http://127.0.0.1:8000/infohash/{{.Infohash}}/stream.mp4` + "\n"

	regexpNonDefault := `^(?!.*(informational|entertaining|educational|movies|documentaries|sport|fashion|music|` +
		`regional|ethnic|religion|teleshop|erotic_18_plus|other_18_plus|cyber_games|amateur|webcam)).*`

	cfg := &Config{
		EngineAddr: "127.0.0.1:6878",
		Playlists: []Playlist{
			{
				OutputPath:                   "./out/playlist_mpegts_all.m3u8",
				HeaderTemplate:               headerLine,
				EntryTemplate:                entryLine1 + entryMpegtsLink,
				CategoryRxToCategoryMap:      map[string]string{regexpNonDefault: "other"},
				NameRxToCategoriesMap:        map[string][]string{},
				NameRxFilter:                 []string{},
				NameRxBlacklist:              []string{},
				CategoriesFilter:             []string{},
				CategoriesFilterStrict:       false,
				CategoriesBlacklist:          []string{},
				LanguagesFilter:              []string{},
				LanguagesFilterStrict:        false,
				LanguagesBlacklist:           []string{},
				CountriesFilter:              []string{},
				CountriesFilterStrict:        false,
				CountriesBlacklist:           []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        1.0,
				AvailabilityUpdatedThreshold: time.Hour * 12 * 3,
				RemoveDeadChannels:           lo.ToPtr(false),
				UseMpegTsAnalyzer:            lo.ToPtr(false),
				CheckRespTimeout:             lo.ToPtr(time.Second * 20),
			},
			{
				OutputPath:                   "./out/playlist_hls_tv_+_music_+_no_category.m3u8",
				HeaderTemplate:               headerLine,
				EntryTemplate:                entryLine1 + entryHlsLink,
				CategoryRxToCategoryMap:      map[string]string{`(?i)^tv$`: "television", `^$`: "unknown"},
				NameRxToCategoriesMap:        map[string][]string{},
				NameRxFilter:                 []string{},
				NameRxBlacklist:              []string{},
				CategoriesFilter:             []string{"tv", "music", "unknown"},
				CategoriesFilterStrict:       false,
				CategoriesBlacklist:          []string{},
				LanguagesFilter:              []string{},
				LanguagesFilterStrict:        false,
				LanguagesBlacklist:           []string{},
				CountriesFilter:              []string{},
				CountriesFilterStrict:        false,
				CountriesBlacklist:           []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        1.0,
				AvailabilityUpdatedThreshold: time.Hour * 12 * 3,
				RemoveDeadChannels:           lo.ToPtr(false),
				UseMpegTsAnalyzer:            lo.ToPtr(false),
				CheckRespTimeout:             lo.ToPtr(time.Second * 20),
			},
			{
				OutputPath:                   "./out/playlist_httpaceproxy_all_but_porn.m3u8",
				HeaderTemplate:               headerLine,
				EntryTemplate:                entryLine1 + entryHttpAceProxyLink,
				CategoryRxToCategoryMap:      map[string]string{},
				NameRxToCategoriesMap:        map[string][]string{},
				NameRxFilter:                 []string{},
				NameRxBlacklist:              []string{`(?i).*erotic.*`, `(?i).*porn.*`, `(?i).*18\+.*`},
				CategoriesFilter:             []string{},
				CategoriesFilterStrict:       false,
				CategoriesBlacklist:          []string{"erotic_18_plus", "18+"},
				LanguagesFilter:              []string{},
				LanguagesFilterStrict:        false,
				LanguagesBlacklist:           []string{},
				CountriesFilter:              []string{},
				CountriesFilterStrict:        false,
				CountriesBlacklist:           []string{},
				StatusFilter:                 []int{2},
				AvailabilityThreshold:        1.0,
				AvailabilityUpdatedThreshold: time.Hour * 12 * 3,
				RemoveDeadChannels:           lo.ToPtr(false),
				UseMpegTsAnalyzer:            lo.ToPtr(false),
				CheckRespTimeout:             lo.ToPtr(time.Second * 20),
			},
		},
	}

	commentMap := yaml.CommentMap{
		"$.engineAddr": []*yaml.Comment{
			yaml.HeadComment(" Ace Stream Engine address in format of host:port."),
		},
		"$.playlists": []*yaml.Comment{
			yaml.HeadComment("", " Playlists to generate."),
		},
		"$.playlists[0]": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" MPEG-TS format, alive.",
				" Change any non-default category to 'other'.",
			),
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
				" Template for each channel. Available variables are:",
				" {{.Name}}",
				" {{.Infohash}}",
				" {{.Categories}}",
				" {{.Countries}}",
				" {{.Languages}}",
				" {{.EngineAddr}}",
				" {{.TVGName}}",
				" {{.IconURL}}",
			),
		},
		"$.playlists[0].categoryRxToCategoryMap": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Change categories by category regular expressions (keys) to strings (values).",
				" Use \"^$\" regular expression to match unset categories.",
				" Example:",
				" categoryRxToCategoryMap:",
				"   \"^category regexp A$\": \"becomes category B\"",
				"   \"^category regexp C$\": \"becomes category D\"",
			),
		},
		"$.playlists[0].nameRxToCategoriesMap": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Set categories by name regular expressions (keys) to list of strings (values).",
				" Example:",
				" nameRxToCategoriesMap:",
				"   \"^name regexp A$\":",
				"   - \"will have category B\"",
				"   - \"and category C\"",
				"   \"^name regexp D$\":",
				"   - \"will have category E\"",
				"   - \"and category F\"",
			),
		},
		"$.playlists[0].nameRxFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which name matches any of these regular expressions.",
				" Example:",
				" nameRxFilter:",
				" - \".*keep channels matching name A.*\"",
				" - \".*keep channels matching name B.*\"",
			),
		},
		"$.playlists[0].nameRxBlacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove channels which name matches any of these regular expressions.",
				" Example:",
				" nameRxBlacklist:",
				" - \".*remove channels matching name A.*\"",
				" - \".*remove channels matching name B.*\"",
			),
		},
		"$.playlists[0].categoriesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which category equals to any of these.",
				" See https://docs.acestream.net/developers/knowledge-base/list-of-categories/",
				" for known (but not all possible) categories list.",
				" Use empty string to include results with unset category.",
				" Example:",
				" categoriesFilter:",
				" - \"keep channels with category A\"",
				" - \"keep channels with category B\"",
			),
		},
		"$.playlists[0].categoriesFilterStrict": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" If true, only keep channels with categories that are in filter, but not any other.",
			),
		},
		"$.playlists[0].categoriesBlacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove channels which category equals to any of these.",
				" See https://docs.acestream.net/developers/knowledge-base/list-of-categories/",
				" for known (but not all possible) categories list.",
				" Use empty string to exclude results with unset category.",
				" Example:",
				" categoriesBlacklist:",
				" - \"remove channels with category A\"",
				" - \"remove channels with category B\"",
			),
		},
		"$.playlists[0].languagesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which language equals to any of these.",
				" Languages are 3-character, lower case strings, such as 'eng', 'rus' etc.",
				" Use empty string to include results with unset language.",
				" Example:",
				" languagesFilter:",
				" - \"keep channels with language A\"",
				" - \"keep channels with language B\"",
			),
		},
		"$.playlists[0].languagesFilterStrict": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" If true, only keep channels with languages that are in filter, but not any other.",
			),
		},
		"$.playlists[0].languagesBlacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove channels which language equals to any of these.",
				" Languages are 3-character, lower case strings, such as 'eng', 'rus' etc.",
				" Use empty string to exclude results with unset language.",
				" Example:",
				" languagesBlacklist:",
				" - \"remove channels with language A\"",
				" - \"remove channels with language B\"",
			),
		},
		"$.playlists[0].countriesFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which country equals to any of these.",
				" Countries are 2-character, lower case strings, such as 'us', 'ru' etc. and 'int' for international.",
				" Use empty string to include results with unset country.",
				" Example:",
				" countriesFilter:",
				" - \"keep channels with country A\"",
				" - \"keep channels with country B\"",
			),
		},
		"$.playlists[0].countriesFilterStrict": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" If true, only keep channels with countries that are in filter, but not any other.",
			),
		},
		"$.playlists[0].countriesBlacklist": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove channels which country equals to any of these.",
				" Countries are 2-character, lower case strings, such as 'us', 'ru' etc. and 'int' for international.",
				" Use empty string to exclude results with unset country.",
				" Example:",
				" countriesBlacklist:",
				" - \"remove channels with country A\"",
				" - \"remove channels with country B\"",
			),
		},
		"$.playlists[0].statusFilter": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which status equals to any of these.",
				" Can be 1 (no guaranty that channel is working) or 2 (channel is available).",
				" Example:",
				" statusFilter:",
				" - 1",
				" - 2",
			),
		},
		"$.playlists[0].availabilityThreshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which availability equals to or more than this.",
				" Can be between 0.0 (zero availability) and 1.0 (full availability).",
				" The lower this value is, the less channels gets removed.",
			),
		},
		"$.playlists[0].availabilityUpdatedThreshold": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Only keep channels which availability was updated that much time ago or sooner.",
				" The lower this value is, the more channels gets removed.",
			),
		},
		"$.playlists[0].removeDeadChannels": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Remove channels that does not respond with any content.",
			),
		},
		"$.playlists[0].useMpegTsAnalyzer": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Try to read TS packets when removing dead channels.",
			),
		},
		"$.playlists[0].engineRespTimeout": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" Timeout for reading Ace Stream Engine response when removing dead channels.",
			),
		},
		"$.playlists[1]": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" HLS format, alive, only keep tv, music and empty category.",
				" Change category 'tv' to 'television' and empty category to 'unknown'.",
			),
		},
		"$.playlists[2]": []*yaml.Comment{
			yaml.HeadComment(
				"",
				" https://github.com/pepsik-kiev/HTTPAceProxy format, alive, all but erotic channels.",
			),
		},
	}

	return cfg, commentMap
}

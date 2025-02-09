package m3u

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/config"
	"m3u_gen_acestream/util/logger"
)

type FilterTest struct {
	input     []acestream.SearchResult
	playlist  config.Playlist
	expected  []acestream.SearchResult
	logOutput string
}

var timeRx = `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`

func TestFilterByCategories(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]FilterTest{
		"filter and blacklist are nil": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesFilter:    nil,
				CategoriesBlacklist: nil,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`,
		},
		"filter and blacklist are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesFilter:    []string{},
				CategoriesBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`,
		},
		"filter is empty string, categories have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"eng", "rus", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:       "file.m3u8",
				CategoriesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"eng", "rus", ""}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`,
		},
		"blacklist is empty string, categories have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"eng", "rus", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`,
		},
		"filter is empty string, categories does not have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			playlist: config.Playlist{
				OutputPath:       "file.m3u8",
				CategoriesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`,
		},
		"blacklist is empty string, categories does not have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`,
		},
		"filter has empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			playlist: config.Playlist{
				OutputPath:       "file.m3u8",
				CategoriesFilter: []string{"", "movies"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`,
		},
		"blacklist has empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport"}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesBlacklist: []string{"", "movies"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`,
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"movies", "sport"}},
					{Name: "name 2", Categories: []string{"regional", "movies"}},
					{Name: "name 3", Categories: []string{"regional"}},
					{Name: "name 4", Categories: []string{"sport", "documentaries"}},
					{Name: "name 5", Categories: []string{"regional", "fashion"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesFilter:    []string{"movies", "regional"},
				CategoriesBlacklist: []string{"fashion"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"movies", "sport"}},
					{Name: "name 2", Categories: []string{"regional", "movies"}},
					{Name: "name 3", Categories: []string{"regional"}},
				}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "2", by "categories", playlist "file.m3u8"`,
		},
	}

	for name, test := range tests {
		actual := filterByCategories(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		assert.Regexp(t, regexp.MustCompile(test.logOutput), consoleBuff.String(), msg)
		consoleBuff.Reset()
	}
}

func TestFilterByLanguages(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]FilterTest{
		"filter and blacklist are nil": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesFilter:    nil,
				LanguagesBlacklist: nil,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`,
		},
		"filter and blacklist are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesFilter:    []string{},
				LanguagesBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`,
		},
		"filter is empty string, languages have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				LanguagesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus", ""}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`,
		},
		"blacklist is empty string, languages have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`,
		},
		"filter is empty string, languages does not have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				LanguagesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`,
		},
		"blacklist is empty string, languages does not have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`,
		},
		"filter has empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				LanguagesFilter: []string{"", "eng"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`,
		},
		"blacklist has empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesBlacklist: []string{"", "eng"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`,
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Languages: []string{"eng", "rus"}},
					{Name: "name 2", Languages: []string{"kaz", "eng"}},
					{Name: "name 3", Languages: []string{"kaz"}},
					{Name: "name 4", Languages: []string{"rus", "ron"}},
					{Name: "name 5", Languages: []string{"kaz", "kor"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesFilter:    []string{"eng", "kaz"},
				LanguagesBlacklist: []string{"kor"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Languages: []string{"eng", "rus"}},
					{Name: "name 2", Languages: []string{"kaz", "eng"}},
					{Name: "name 3", Languages: []string{"kaz"}},
				}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "2", by "languages", playlist "file.m3u8"`,
		},
	}

	for name, test := range tests {
		actual := filterByLanguages(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		assert.Regexp(t, regexp.MustCompile(test.logOutput), consoleBuff.String(), msg)
		consoleBuff.Reset()
	}
}

func TestFilterByCountries(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]FilterTest{
		"filter and blacklist are nil": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesFilter:    nil,
				CountriesBlacklist: nil,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`,
		},
		"filter and blacklist are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesFilter:    []string{},
				CountriesBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`,
		},
		"filter is empty string, countries have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				CountriesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru", ""}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`,
		},
		"blacklist is empty string, countries have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`,
		},
		"filter is empty string, countries does not have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				CountriesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`,
		},
		"blacklist is empty string, countries does not have empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`,
		},
		"filter has empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				CountriesFilter: []string{"", "us"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`,
		},
		"blacklist has empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru"}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesBlacklist: []string{"", "us"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`,
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Countries: []string{"us", "ru"}},
					{Name: "name 2", Countries: []string{"kz", "us"}},
					{Name: "name 3", Countries: []string{"kz"}},
					{Name: "name 4", Countries: []string{"ru", "md"}},
					{Name: "name 5", Countries: []string{"kz", "ko"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesFilter:    []string{"us", "kz"},
				CountriesBlacklist: []string{"ko"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Countries: []string{"us", "ru"}},
					{Name: "name 2", Countries: []string{"kz", "us"}},
					{Name: "name 3", Countries: []string{"kz"}},
				}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "2", by "countries", playlist "file.m3u8"`,
		},
	}

	for name, test := range tests {
		actual := filterByCountries(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		assert.Regexp(t, regexp.MustCompile(test.logOutput), consoleBuff.String(), msg)
		consoleBuff.Reset()
	}
}

func TestFilterByName(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]FilterTest{
		"regular expression lists are nil": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    nil,
				NameRegexpBlacklist: nil,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`,
		},
		"regular expressions in lists are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{},
				NameRegexpBlacklist: []*regexp.Regexp{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`,
		},
		"regular expressions in lists are nil": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{nil},
				NameRegexpBlacklist: []*regexp.Regexp{nil},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`,
		},
		"filter is empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{regexp.MustCompile("")},
				NameRegexpBlacklist: []*regexp.Regexp{nil},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`,
		},
		"blacklist is empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{nil},
				NameRegexpBlacklist: []*regexp.Regexp{regexp.MustCompile("")},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "1", by "name", playlist "file.m3u8"`,
		},
		"filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}, {Name: "xxx skip xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}, {Name: "xxx skip xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{regexp.MustCompile(`.*keep1.*`), regexp.MustCompile(`.*keep2.*`)},
				NameRegexpBlacklist: []*regexp.Regexp{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "2", by "name", playlist "file.m3u8"`,
		},
		"blacklist is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{},
				NameRegexpBlacklist: []*regexp.Regexp{regexp.MustCompile(`.*skip1.*`), regexp.MustCompile(`.*skip2.*`)},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "4", by "name", playlist "file.m3u8"`,
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "other"}}},
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{regexp.MustCompile(`xxx .* xxx`)},
				NameRegexpBlacklist: []*regexp.Regexp{regexp.MustCompile(`.*skip1.*`), regexp.MustCompile(`.*skip2.*`)},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "5", by "name", playlist "file.m3u8"`,
		},
	}

	for name, test := range tests {
		actual := filterByName(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		assert.Regexp(t, regexp.MustCompile(test.logOutput), consoleBuff.String(), msg)
		consoleBuff.Reset()
	}
}

package m3u

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/config"
	"m3u_gen_acestream/util/logger"
)

type TransformTest struct {
	input    []acestream.SearchResult
	playlist config.Playlist
	expected []acestream.SearchResult
	logLines []string
}

var timeRx = `[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}`

func TestRemapCategoryToCategory(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
		"change 9 categories for 6 items": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"tv", "movies", "music"}},
					{Name: "name 2", Categories: []string{"music", "tv"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 3", Categories: []string{"movies", ""}},
					{Name: "name 4", Categories: []string{"audio", "unknown"}},
					{Name: "name 5", Categories: []string{"music", "documentary"}},
					{Name: "name 6", Categories: []string{"tv", "tv"}},
					{Name: "name 7", Categories: []string{""}},
					{Name: "name 8", Categories: []string{}},
				}},
			},
			playlist: config.Playlist{
				OutputPath: "file.m3u8",
				CategoryRxToCategoryMap: map[string]string{
					"(?i)^tv$":    "television",
					"(?i)^music$": "audio",
					"^$":          "unknown",
				},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"television", "movies", "audio"}},
					{Name: "name 2", Categories: []string{"audio", "television"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 3", Categories: []string{"movies", "unknown"}},
					{Name: "name 4", Categories: []string{"audio", "unknown"}},
					{Name: "name 5", Categories: []string{"audio", "documentary"}},
					{Name: "name 6", Categories: []string{"television", "television"}},
					{Name: "name 7", Categories: []string{"unknown"}},
					{Name: "name 8", Categories: []string{"unknown"}},
				}},
			},
			logLines: []string{
				timeRx + ` DEBUG Changed: category "tv", to "television", playlist "file.m3u8"`,
				timeRx + ` INFO Changed: categories "10", by "category to category map", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := remapCategoryToCategory(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestRemapNameToCategories(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
		"change 3 categories for 2 items": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"tv", "movies"}},
					{Name: "name 2", Categories: []string{"music", "tv"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 3", Categories: []string{"movies", ""}},
					{Name: "name 4", Categories: []string{}},
					{Name: ""},
				}},
			},
			playlist: config.Playlist{
				OutputPath: "file.m3u8",
				NameRxToCategoriesMap: map[string][]string{
					"^name 2$": {"category 1", "category 2"},
					"^name 4$": {"category 3"},
				},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"tv", "movies"}},
					{Name: "name 2", Categories: []string{"category 1", "category 2"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 3", Categories: []string{"movies", ""}},
					{Name: "name 4", Categories: []string{"category 3"}},
					{Name: ""},
				}},
			},
			logLines: []string{
				timeRx + ` DEBUG Changed: categories "\["music","tv"\]", to ` +
					`"\["category 1","category 2"\]", by name "name 2", playlist "file.m3u8"`,
				timeRx + ` INFO Changed: categories "3", by "name to categories map", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := remapNameToCategories(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByStatus(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
		"two items with bad status": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Status: 2}, {Name: "name 2", Status: 3}}},
				{Items: []acestream.Item{{Name: "name 3", Status: 1}, {Name: "name 4", Status: -1}}},
			},
			playlist: config.Playlist{
				OutputPath:   "file.m3u8",
				StatusFilter: []int{1, 2},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Status: 2}}},
				{Items: []acestream.Item{{Name: "name 3", Status: 1}}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "name 2", status "3", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "2", by "status", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByStatus(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByAvailability(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
		"two items exceed threshold": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Availability: 1.0}, {Name: "name 2", Availability: 0.8}}},
				{Items: []acestream.Item{{Name: "name 3", Availability: 0.5}, {Name: "name 4", Availability: 0.5}}},
			},
			playlist: config.Playlist{
				OutputPath:            "file.m3u8",
				AvailabilityThreshold: 0.8,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Availability: 1.0}, {Name: "name 2", Availability: 0.8}}},
				{Items: []acestream.Item{}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "name 3", availability "0.5", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "2", by "availability", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByAvailability(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByAvailabilityUpdateTime(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	now := time.Now().Unix()

	tests := map[string]TransformTest{
		"two items exceed threshold": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", AvailabilityUpdatedAt: now - 100},
					{Name: "name 2", AvailabilityUpdatedAt: now - 300},
				}},
				{Items: []acestream.Item{
					{Name: "name 3", AvailabilityUpdatedAt: now - 400},
				}},
			},
			playlist: config.Playlist{
				OutputPath:                   "file.m3u8",
				AvailabilityUpdatedThreshold: time.Second * 200,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", AvailabilityUpdatedAt: now - 100}}},
				{Items: []acestream.Item{}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "name 2", availability updated at "` +
					fmt.Sprint(now-300) + `", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "2", by "availability update time", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByAvailabilityUpdateTime(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByCategories(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`},
		},
		"filter and blacklist are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesFilter:    []string{},
				CategoriesBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{"movies", "sport", ""}}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`},
		},
		"filter is empty string, categories are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{}}}},
			},
			playlist: config.Playlist{
				OutputPath:       "file.m3u8",
				CategoriesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{}}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`},
		},
		"blacklist is empty string, categories are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Categories: []string{}}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				CategoriesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "categories", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`},
		},
		"soft filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"movies", "sport"}},
					{Name: "name 2", Categories: []string{"regional", "movies", "documentaries"}},
					{Name: "name 3", Categories: []string{"regional"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Categories: []string{"sport", "documentaries"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:       "file.m3u8",
				CategoriesFilter: []string{"movies", "regional"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"movies", "sport"}},
					{Name: "name 2", Categories: []string{"regional", "movies", "documentaries"}},
					{Name: "name 3", Categories: []string{"regional"}},
				}},
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "categories", playlist "file.m3u8"`},
		},
		"strict filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"movies", "sport"}},
					{Name: "name 2", Categories: []string{"documentaries", "movies"}},
					{Name: "name 3", Categories: []string{"regional", "movies"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Categories: []string{"movies", "regional"}},
					{Name: "name 5", Categories: []string{"movies", "regional", "sport"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:             "file.m3u8",
				CategoriesFilter:       []string{"movies", "regional", "documentaries"},
				CategoriesFilterStrict: true,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 2", Categories: []string{"documentaries", "movies"}},
					{Name: "name 3", Categories: []string{"regional", "movies"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Categories: []string{"movies", "regional"}},
				}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "2", by "categories", playlist "file.m3u8"`},
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Categories: []string{"movies", "sport"}},
					{Name: "name 2", Categories: []string{"regional", "movies"}},
				}},
				{Items: []acestream.Item{
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
				}},
				{Items: []acestream.Item{
					{Name: "name 3", Categories: []string{"regional"}},
				}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "name 4", ` +
					`categories "\["sport","documentaries"\]", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "2", by "categories", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByCategories(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByLanguages(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`},
		},
		"filter and blacklist are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesFilter:    []string{},
				LanguagesBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{"eng", "rus", ""}}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`},
		},
		"filter is empty string, languages are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				LanguagesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{}}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`},
		},
		"blacklist is empty string, languages are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Languages: []string{}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				LanguagesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "languages", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`},
		},
		"soft filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Languages: []string{"eng", "rus"}},
					{Name: "name 2", Languages: []string{"kaz", "eng", "ron"}},
					{Name: "name 3", Languages: []string{"kaz"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Languages: []string{"rus", "ron"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				LanguagesFilter: []string{"eng", "kaz"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Languages: []string{"eng", "rus"}},
					{Name: "name 2", Languages: []string{"kaz", "eng", "ron"}},
					{Name: "name 3", Languages: []string{"kaz"}},
				}},
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "languages", playlist "file.m3u8"`},
		},
		"strict filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Languages: []string{"eng", "rus"}},
					{Name: "name 2", Languages: []string{"md", "eng"}},
					{Name: "name 3", Languages: []string{"kaz", "eng"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Languages: []string{"eng", "kaz"}},
					{Name: "name 5", Languages: []string{"eng", "kaz", "rus"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:            "file.m3u8",
				LanguagesFilter:       []string{"eng", "kaz", "md"},
				LanguagesFilterStrict: true,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 2", Languages: []string{"md", "eng"}},
					{Name: "name 3", Languages: []string{"kaz", "eng"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Languages: []string{"eng", "kaz"}},
				}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "2", by "languages", playlist "file.m3u8"`},
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Languages: []string{"eng", "rus"}},
					{Name: "name 2", Languages: []string{"kaz", "eng"}},
				}},
				{Items: []acestream.Item{
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
				}},
				{Items: []acestream.Item{
					{Name: "name 3", Languages: []string{"kaz"}},
				}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "name 4", languages "\["rus","ron"\]", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "2", by "languages", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByLanguages(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByCountries(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`},
		},
		"filter and blacklist are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru", ""}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesFilter:    []string{},
				CountriesBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{"us", "ru", ""}}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`},
		},
		"filter is empty string, countries are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{}}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				CountriesFilter: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{}}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`},
		},
		"blacklist is empty string, countries are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1", Countries: []string{}}}},
			},
			playlist: config.Playlist{
				OutputPath:         "file.m3u8",
				CountriesBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "countries", playlist "file.m3u8"`},
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
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`},
		},
		"soft filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Countries: []string{"us", "ru"}},
					{Name: "name 2", Countries: []string{"kz", "us", "md"}},
					{Name: "name 3", Countries: []string{"kz"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Countries: []string{"ru", "md"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				CountriesFilter: []string{"us", "kz"},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Countries: []string{"us", "ru"}},
					{Name: "name 2", Countries: []string{"kz", "us", "md"}},
					{Name: "name 3", Countries: []string{"kz"}},
				}},
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "countries", playlist "file.m3u8"`},
		},
		"strict filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Countries: []string{"us", "ru"}},
					{Name: "name 2", Countries: []string{"md", "us"}},
					{Name: "name 3", Countries: []string{"kz", "us"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Countries: []string{"us", "kz"}},
					{Name: "name 5", Countries: []string{"us", "kz", "ru"}},
				}},
			},
			playlist: config.Playlist{
				OutputPath:            "file.m3u8",
				CountriesFilter:       []string{"us", "kz", "md"},
				CountriesFilterStrict: true,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 2", Countries: []string{"md", "us"}},
					{Name: "name 3", Countries: []string{"kz", "us"}},
				}},
				{Items: []acestream.Item{
					{Name: "name 4", Countries: []string{"us", "kz"}},
				}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "2", by "countries", playlist "file.m3u8"`},
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1", Countries: []string{"us", "ru"}},
					{Name: "name 2", Countries: []string{"kz", "us"}},
					{Name: "name 3", Countries: []string{"kz"}},
				}},
				{Items: []acestream.Item{
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
				{Items: []acestream.Item{}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "name 4", countries "\["ru","md"\]", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "2", by "countries", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByCountries(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestFilterByName(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]TransformTest{
		"regular expression lists are nil": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    nil,
				NameRxBlacklist: nil,
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`},
		},
		"regular expressions in lists are empty": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    []string{},
				NameRxBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`},
		},
		"filter is empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    []string{""},
				NameRxBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "0", by "name", playlist "file.m3u8"`},
		},
		"blacklist is empty string": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    []string{},
				NameRxBlacklist: []string{""},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "1", by "name", playlist "file.m3u8"`},
		},
		"filter is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}, {Name: "xxx skip xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}, {Name: "xxx skip xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    []string{`.*keep1.*`, `.*keep2.*`},
				NameRxBlacklist: []string{},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep1 xxx"}, {Name: "xxx keep2 xxx"}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "2", by "name", playlist "file.m3u8"`},
		},
		"blacklist is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    []string{},
				NameRxBlacklist: []string{`.*skip1.*`, `.*skip2.*`},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
			},
			logLines: []string{timeRx + ` INFO Rejected: sources "4", by "name", playlist "file.m3u8"`},
		},
		"filter and blacklist are set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "other"}}},
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:      "file.m3u8",
				NameRxFilter:    []string{`xxx .* xxx`},
				NameRxBlacklist: []string{`.*skip1.*`, `.*skip2.*`},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{}},
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
			},
			logLines: []string{
				timeRx + ` DEBUG Rejected: name "xxx skip1 xxx", playlist "file.m3u8"`,
				timeRx + ` INFO Rejected: sources "5", by "name", playlist "file.m3u8"`,
			},
		},
	}

	for name, test := range tests {
		actual := filterByName(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}
}

func TestRemoveDead(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	hashAlive := "a7c19473d3389a3d9c9d1e268ce6e0550fea3192"
	hashDead := "9ddda51034375eb93505c076d4437064abdf2dcd"
	linkRxFmt := `http:\/\/127\.0\.0\.1:8080\/ace\/getstream\?infohash=%v`
	linkTempl := "http://127.0.0.1:8080/ace/getstream?infohash={{.Infohash}}"

	tests := map[string]TransformTest{
		"2 alive, 2 dead sources": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{
					{Name: "name 1 alive", Infohash: hashAlive},
					{Name: "name 2 dead", Infohash: hashDead},
				}},
				{Items: []acestream.Item{
					{Name: "name 3 alive", Infohash: hashAlive},
					{Name: "name 4 dead", Infohash: hashDead},
				}},
			},
			playlist: config.Playlist{
				OutputPath:        "file.m3u8",
				RemoveDeadSources: lo.ToPtr(true),
				UseMpegTsAnalyzer: lo.ToPtr(true),
				CheckRespTimeout:  lo.ToPtr(time.Second * 50),
				RemoveDeadLinkTemplate: lo.ToPtr(linkTempl),
				RemoveDeadWorkers: lo.ToPtr(2),
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "name 1 alive", Infohash: hashAlive}}},
				{Items: []acestream.Item{{Name: "name 3 alive", Infohash: hashAlive}}},
			},
			logLines: []string{
				timeRx + ` INFO Keep: name "name 1 alive", link "` + fmt.Sprintf(linkRxFmt, hashAlive) + `"`,
				timeRx + ` WARN Reject: name "name 2 dead", link "` + fmt.Sprintf(linkRxFmt, hashDead) + `", reason ` +
					`"Response status 500 Internal Server Error"`,
				timeRx + ` INFO Rejected: sources "2", by "response", playlist "file.m3u8"`,
			},
		},
	}

	infohashCheckErrorMap := make(map[string]error)
	for name, test := range tests {
		actual := removeDead(log, test.input, test.playlist, "127.0.0.1:6878", infohashCheckErrorMap)
		slices.SortStableFunc(actual, func(a, b acestream.SearchResult) int {
			return strings.Compare(a.Name, b.Name)
		})
		assert.Exactly(t, test.expected, actual, fmt.Sprintf("Bad returned value in test '%v'", name))
		msg := fmt.Sprintf("Bad log output in test '%v'", name)
		for _, line := range test.logLines {
			assert.Regexp(t, regexp2.MustCompile(line, regexp2.RE2), consoleBuff.String(), msg)
		}
		consoleBuff.Reset()
	}

	assert.Contains(t, infohashCheckErrorMap, hashAlive)
	assert.Contains(t, infohashCheckErrorMap, hashDead)
}

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
		"filter and blacklist is set": {
			input: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
				{Items: []acestream.Item{{Name: "xxx skip1 xxx"}, {Name: "xxx skip2 xxx"}, {Name: "xxx keep xxx"}}},
			},
			playlist: config.Playlist{
				OutputPath:          "file.m3u8",
				NameRegexpFilter:    []*regexp.Regexp{regexp.MustCompile(`xxx .* xxx`)},
				NameRegexpBlacklist: []*regexp.Regexp{regexp.MustCompile(`.*skip1.*`), regexp.MustCompile(`.*skip2.*`)},
			},
			expected: []acestream.SearchResult{
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
				{Items: []acestream.Item{{Name: "xxx keep xxx"}}},
			},
			logOutput: timeRx + ` INFO Rejected: sources "4", by "name", playlist "file.m3u8"`,
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

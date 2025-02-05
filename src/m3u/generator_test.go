package m3u

import (
	"bytes"
	"regexp"
	"testing"

	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/config"
	"m3u_gen_acestream/util/logger"

	"github.com/stretchr/testify/assert"
)

func TestFilterByName(t *testing.T) {
	var consoleBuff bytes.Buffer
	log := logger.New(logger.DebugLevel, &consoleBuff)

	tests := map[string]struct {
		input     []acestream.SearchResult
		playlist  config.Playlist
		expected  []acestream.SearchResult
		logOutput string
	}{
		"regular expressions are nil": {
			input: []acestream.SearchResult{
				{
					Items: []acestream.Item{
						{Name: "name 1"},
					},
				},
			},
			playlist: config.Playlist{
				OutputPath: "file.m3u8",
				NameRegexpFilter: []*regexp.Regexp{
					nil,
				},
				NameRegexpBlacklist: []*regexp.Regexp{
					nil,
				},
			},
			expected: []acestream.SearchResult{
				{
					Items: []acestream.Item{
						{Name: "name 1"},
					},
				},
			},
			logOutput: `Rejected: sources "0", by "name", playlist "file.m3u8"`,
		},
	}

	for _, test := range tests {
		actual := filterByName(log, test.input, test.playlist)
		assert.Exactly(t, test.expected, actual, "Unexpected returned value")
		assert.Contains(t, consoleBuff.String(), test.logOutput, "Unexpected log output")
		consoleBuff.Reset()
	}
}

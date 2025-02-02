package m3u

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"

	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/config"
	"m3u_gen_acestream/util/logger"
)

// Entry represents M3U file entry to execute template on.
type Entry struct {
	Categories string
	Name       string
	EngineAddr string
	Infohash   string
	// TODO: TVGName: Replace " " with "_" in Name? (Add tvg-name="{{.TVGName}}" to entry template)
}

// Generate writes M3U file based on filtered `searchResults` using settings in config `cfg`.
func Generate(log *logger.Logger, searchResults []acestream.SearchResult, cfg *config.Config) error {
	log.Info("Generating M3U files")

	for _, playlist := range cfg.Playlists {
		// Filter []SearchResult
		log.Infof("Filtering results for playlist %v", playlist.OutputPath)
		searchResults = lo.Filter(searchResults, func(sr acestream.SearchResult, _ int) bool {
			var keep bool
			keep = playlist.NameRegexpFilter.MatchString(sr.Name)
			return keep
		})

		// Transform []SearchResult to []Entry.
		entries := lo.FlatMap(searchResults, func(searchResult acestream.SearchResult, _ int) []Entry {
			return lo.Map(searchResult.Items, func(item acestream.Item, _ int) Entry {
				return Entry{
					Categories: strings.Join(item.Categories, ";"),
					Name:       item.Name,
					EngineAddr: cfg.EngineAddr,
					Infohash:   item.Infohash,
				}
			})
		})

		// Write playlists
		log.Infof("Writing playlist %v", playlist.OutputPath)
		if err := os.MkdirAll(filepath.Dir(playlist.OutputPath), os.ModePerm); err != nil {
			return errors.Wrapf(err, "Make directory structure for playlist %v", playlist.OutputPath)
		}
		var buff bytes.Buffer
		buff.WriteString(string(playlist.HeaderTemplate))
		for _, entry := range entries {
			if err := playlist.EntryTemplate.Execute(&buff, entry); err != nil {
				return errors.Wrapf(err, "Execute template for entry %+v", entry)
			}
		}
		if err := os.WriteFile(playlist.OutputPath, buff.Bytes(), 0644); err != nil {
			return errors.Wrapf(err, "Write playlist file %v", playlist.OutputPath)
		}
	}

	return nil
}

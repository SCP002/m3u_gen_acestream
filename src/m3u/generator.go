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
	Name       string
	Infohash   string
	Categories string
	EngineAddr string
	TVGName string
	IconURL string
}

// Generate writes M3U file based on filtered `searchResults` using settings in config `cfg`.
func Generate(log *logger.Logger, searchResults []acestream.SearchResult, cfg *config.Config) error {
	log.Info("Generating M3U files")

	for _, playlist := range cfg.Playlists {
		log.Infof("Filtering results for playlist %v", playlist.OutputPath)

		// // Filter by status
		// searchResults = lo.FilterMap(searchResults, func(sr acestream.SearchResult, _ int) (acestream.SearchResult, bool) {
		// 	sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
		// 		return lo.Contains(playlist.StatusFilter, item.Status)
		// 	})
		// 	return sr, true
		// })

		// // Filter by name
		// searchResults = lo.Filter(searchResults, func(sr acestream.SearchResult, _ int) bool {
		// 	return playlist.NameRegexpFilter.MatchString(sr.Name)
		// })

		// Transform []SearchResult to []Entry.
		entries := lo.FlatMap(searchResults, func(searchResult acestream.SearchResult, _ int) []Entry {
			return lo.Map(searchResult.Items, func(item acestream.Item, _ int) Entry {
				iconURLs := lo.Map(searchResult.Icons, func(icon acestream.Icon, _ int) string {
					return icon.URL
				})
				return Entry{
					Name:       item.Name,
					Infohash:   item.Infohash,
					Categories: strings.Join(item.Categories, ";"),
					EngineAddr: cfg.EngineAddr,
					TVGName:    strings.ReplaceAll(item.Name, " ", "_"),
					IconURL:    lo.FirstOr(iconURLs, ""),
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

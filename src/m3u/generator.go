package m3u

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	TVGName    string
	IconURL    string
}

// Generate writes M3U file based on filtered `searchResults` using settings in config `cfg`.
func Generate(log *logger.Logger, searchResults []acestream.SearchResult, cfg *config.Config) error {
	log.Info("Generating M3U files")

	for _, playlist := range cfg.Playlists {
		log.Infof("Filtering channels for playlist %v", playlist.OutputPath)

		// Filter by status
		prevSources := acestream.GetSourcesAmount(searchResults)
		searchResults = lo.Map(searchResults, func(searchResult acestream.SearchResult, _ int) acestream.SearchResult {
			searchResult.Items = lo.Filter(searchResult.Items, func(item acestream.Item, _ int) bool {
				return lo.Contains(playlist.StatusFilter, item.Status)
			})
			return searchResult
		})
		currSources := acestream.GetSourcesAmount(searchResults)
		log.Infof("Rejected %v sources by status for playlist %v", prevSources-currSources, playlist.OutputPath)

		// Filter by availability
		prevSources = currSources
		searchResults = lo.Map(searchResults, func(searchResult acestream.SearchResult, _ int) acestream.SearchResult {
			searchResult.Items = lo.Filter(searchResult.Items, func(item acestream.Item, _ int) bool {
				return item.Availability >= playlist.AvailabilityThreshold
			})
			return searchResult
		})
		currSources = acestream.GetSourcesAmount(searchResults)
		log.Infof("Rejected %v sources by availability for playlist %v", prevSources-currSources, playlist.OutputPath)

		// Filter by availability update time
		prevSources = currSources
		searchResults = lo.Map(searchResults, func(searchResult acestream.SearchResult, _ int) acestream.SearchResult {
			searchResult.Items = lo.Filter(searchResult.Items, func(item acestream.Item, _ int) bool {
				now := time.Now().Unix()
				return (now - item.AvailabilityUpdatedAt) <= int64(playlist.AvailabilityUpdatedThreshold.Seconds())
			})
			return searchResult
		})
		currSources = acestream.GetSourcesAmount(searchResults)
		log.Infof("Rejected %v sources by availability update time for playlist %v",
			prevSources-currSources, playlist.OutputPath)

		// Filter by name
		prevSources = currSources
		searchResults = lo.Filter(searchResults, func(searchResult acestream.SearchResult, _ int) bool {
			return playlist.NameRegexpFilter.MatchString(searchResult.Name)
		})
		currSources = acestream.GetSourcesAmount(searchResults)
		log.Infof("Rejected %v sources by name for playlist %v", prevSources-currSources, playlist.OutputPath)

		// Transform []SearchResult to []Entry.
		entries := lo.FlatMap(searchResults, func(searchResult acestream.SearchResult, _ int) []Entry {
			iconURLs := lo.Map(searchResult.Icons, func(icon acestream.Icon, _ int) string {
				return icon.URL
			})
			return lo.Map(searchResult.Items, func(item acestream.Item, _ int) Entry {
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

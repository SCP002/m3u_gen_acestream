package m3u

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
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
		searchResults := filter(log, searchResults, playlist)

		// Transform []SearchResult to []Entry.
		entries := lo.FlatMap(searchResults, func(sr acestream.SearchResult, _ int) []Entry {
			iconURLs := lo.Map(sr.Icons, func(icon acestream.Icon, _ int) string {
				return icon.URL
			})
			return lo.Map(sr.Items, func(item acestream.Item, _ int) Entry {
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

		// Write playlists.
		log.InfoFi("Writing output", "playlist", playlist.OutputPath)
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

// filter returns filtered `searchResults` by criterias in `playlist`.
func filter(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	searchResults = filterByStatus(log, searchResults, playlist)
	searchResults = filterByAvailability(log, searchResults, playlist)
	searchResults = filterByAvailabilityUpdateTime(log, searchResults, playlist)
	searchResults = filterByCategories(log, searchResults, playlist)
	searchResults = filterByLanguages(log, searchResults, playlist)
	searchResults = filterByCountries(log, searchResults, playlist)
	searchResults = filterByName(log, searchResults, playlist)
	return searchResults
}

// filterByStatus returns filtered `searchResults` by status list in `playlist`.
func filterByStatus(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
			return lo.Contains(playlist.StatusFilter, item.Status)
		})
		return sr
	})
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "status", "playlist", playlist.OutputPath)
	return searchResults
}

// filterByAvailability returns filtered `searchResults` by availability in `playlist`.
func filterByAvailability(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
			return item.Availability >= playlist.AvailabilityThreshold
		})
		return sr
	})
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "availability", "playlist", playlist.OutputPath)
	return searchResults
}

// filterByAvailabilityUpdateTime returns filtered `searchResults` by availability update time in `playlist`.
func filterByAvailabilityUpdateTime(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
			availabilityUpdatedAgo := time.Now().Unix() - item.AvailabilityUpdatedAt
			return availabilityUpdatedAgo <= int64(playlist.AvailabilityUpdatedThreshold.Seconds())
		})
		return sr
	})
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "availability update time",
		"playlist", playlist.OutputPath)
	return searchResults
}

// filterByCategories returns filtered `searchResults` by categories list in `playlist`.
func filterByCategories(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	if len(playlist.CategoriesFilter) > 0 {
		searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
			sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
				return lo.Some(item.Categories, playlist.CategoriesFilter)
			})
			return sr
		})
	}
	if len(playlist.CategoriesBlacklist) > 0 {
		searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
			sr.Items = lo.Reject(sr.Items, func(item acestream.Item, _ int) bool {
				return lo.Some(item.Categories, playlist.CategoriesBlacklist)
			})
			return sr
		})
	}
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "categories", "playlist", playlist.OutputPath)
	return searchResults
}

// filterByLanguages returns filtered `searchResults` by languages list in `playlist`.
func filterByLanguages(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	if len(playlist.LanguagesFilter) > 0 {
		searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
			sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
				return lo.Some(item.Languages, playlist.LanguagesFilter)
			})
			return sr
		})
	}
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "languages", "playlist", playlist.OutputPath)
	return searchResults
}

// filterByCountries returns filtered `searchResults` by countries list in `playlist`.
func filterByCountries(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	if len(playlist.CountriesFilter) > 0 {
		searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
			sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
				return lo.Some(item.Countries, playlist.CountriesFilter)
			})
			return sr
		})
	}
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "countries", "playlist", playlist.OutputPath)
	return searchResults
}

// filterByName returns filtered `searchResults` by name regular expressions in `playlist`.
func filterByName(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	prevSources := acestream.GetSourcesAmount(searchResults)
	if len(playlist.NameRegexpFilter) > 0 {
		searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
			sr.Items = lo.Filter(sr.Items, func(item acestream.Item, _ int) bool {
				return lo.SomeBy(playlist.NameRegexpFilter, func(rx *regexp.Regexp) bool {
					if rx == nil {
						return true
					}
					return rx.MatchString(item.Name)
				})
			})
			return sr
		})
	}
	if len(playlist.NameRegexpBlacklist) > 0 {
		searchResults = lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
			sr.Items = lo.Reject(sr.Items, func(item acestream.Item, _ int) bool {
				return lo.SomeBy(playlist.NameRegexpBlacklist, func(rx *regexp.Regexp) bool {
					if rx == nil {
						return false
					}
					return rx.MatchString(item.Name)
				})
			})
			return sr
		})
	}
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "name", "playlist", playlist.OutputPath)
	return searchResults
}

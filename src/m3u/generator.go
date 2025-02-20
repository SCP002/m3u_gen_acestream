package m3u

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"

	"m3u_gen_acestream/acestream"
	"m3u_gen_acestream/config"
	"m3u_gen_acestream/util/logger"
	"m3u_gen_acestream/util/maps"
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
		searchResults := remap(log, searchResults, playlist)
		searchResults = filter(log, searchResults, playlist)

		// Transform []SearchResult to []Entry.
		entries := lo.FlatMap(searchResults, func(sr acestream.SearchResult, _ int) []Entry {
			iconURLs := lo.Map(sr.Icons, func(icon acestream.Icon, _ int) string {
				return icon.URL
			})
			return lo.Map(sr.Items, func(item acestream.Item, _ int) Entry {
				categories := lo.Compact(lo.Uniq(lo.Map(item.Categories, func(category string, _ int) string {
					return strings.ToLower(category)
				})))
				slices.Sort(categories)
				return Entry{
					Name:       item.Name,
					Infohash:   item.Infohash,
					Categories: strings.Join(categories, ";"),
					EngineAddr: cfg.EngineAddr,
					TVGName:    strings.ReplaceAll(item.Name, " ", "_"),
					IconURL:    lo.FirstOr(iconURLs, ""),
				}
			})
		})

		// Sort entries by categories.
		slices.SortFunc(entries, func(a, b Entry) int {
			return strings.Compare(a.Categories, b.Categories)
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

// remap returns `searchResults` with categories changed by criterias in `playlist`.
func remap(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	searchResults = remapCategoryToCategory(log, searchResults, playlist)
	return searchResults
}

// remapCategoryToCategory returns `searchResults` with categories changed to respective map values in `playlist`.
func remapCategoryToCategory(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	var changed int
	if len(playlist.CategoryRxToCategoryMap) > 0 {
		searchResults = mapAcestreamCategories(searchResults, func(category string, _ int) string {
			maps.ForEveryMatchingRx(playlist.CategoryRxToCategoryMap, category, func(newCategory string) {
				category = newCategory
				changed++
			})
			return category
		})
	}
	log.InfoFi("Changed", "categories", changed, "by", "category to category map", "playlist", playlist.OutputPath)
	return searchResults
}

// mapAcestreamCategories runs `cb` function for every acestream item category in `searchResults`.
//
// `cb` function should return modified acestream category.
//
// `cb` function arguments are:
//   - `category` - acestream category.
//   - `idx` - category index.
func mapAcestreamCategories(searchResults []acestream.SearchResult,
	cb func(category string, idx int) string) []acestream.SearchResult {
	return mapAcestreamItems(searchResults, func(item acestream.Item, _ int) acestream.Item {
		item.Categories = lo.Map(item.Categories, cb)
		return item
	})
}

// mapAcestreamItems runs `cb` function for every acestream item in `searchResults`.
//
// `cb` function should return modified acestream item.
//
// `cb` function arguments are:
//   - `item` - acestream item.
//   - `idx` - item index.
func mapAcestreamItems(searchResults []acestream.SearchResult,
	cb func(item acestream.Item, idx int) acestream.Item) []acestream.SearchResult {
	return lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Map(sr.Items, cb)
		return sr
	})
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
	searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
		return lo.Contains(playlist.StatusFilter, item.Status)
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
	searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
		return item.Availability >= playlist.AvailabilityThreshold
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
	searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
		availabilityUpdatedAgo := time.Now().Unix() - item.AvailabilityUpdatedAt
		return availabilityUpdatedAgo <= int64(playlist.AvailabilityUpdatedThreshold.Seconds())
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
		searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			if playlist.CategoriesFilterStrict {
				return lo.Every(playlist.CategoriesFilter, item.Categories)
			} else {
				return lo.Some(item.Categories, playlist.CategoriesFilter)
			}
		})
	}
	if len(playlist.CategoriesBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.Some(item.Categories, playlist.CategoriesBlacklist)
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
		searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			if playlist.LanguagesFilterStrict {
				return lo.Every(playlist.LanguagesFilter, item.Languages)
			} else {
				return lo.Some(item.Languages, playlist.LanguagesFilter)
			}
		})
	}
	if len(playlist.LanguagesBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.Some(item.Languages, playlist.LanguagesBlacklist)
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
		searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			if playlist.CountriesFilterStrict {
				return lo.Every(playlist.CountriesFilter, item.Countries)
			} else {
				return lo.Some(item.Countries, playlist.CountriesFilter)
			}
		})
	}
	if len(playlist.CountriesBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.Some(item.Countries, playlist.CountriesBlacklist)
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
		searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.SomeBy(playlist.NameRegexpFilter, func(rx *regexp.Regexp) bool {
				if rx == nil {
					return true
				}
				return rx.MatchString(item.Name)
			})
		})
	}
	if len(playlist.NameRegexpBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.SomeBy(playlist.NameRegexpBlacklist, func(rx *regexp.Regexp) bool {
				if rx == nil {
					return false
				}
				return rx.MatchString(item.Name)
			})
		})
	}
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "name", "playlist", playlist.OutputPath)
	return searchResults
}

// filterAcestreamItems runs `cb` function for every acestream item in `searchResults`.
//
// `cb` function should return 'true' if item should stay in `searchResults`.
//
// `cb` function arguments are:
//   - `item` - acestream item.
//   - `idx` - current item index.
func filterAcestreamItems(searchResults []acestream.SearchResult,
	cb func(item acestream.Item, idx int) bool) []acestream.SearchResult {
	return lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Filter(sr.Items, cb)
		return sr
	})
}

// rejectAcestreamItems runs `cb` function for every acestream item in `searchResults`.
//
// `cb` function should return 'true' if item should be removed from `searchResults`.
//
// `cb` function arguments are:
//   - `item` - acestream item.
//   - `idx` - item index.
func rejectAcestreamItems(searchResults []acestream.SearchResult,
	cb func(item acestream.Item, idx int) bool) []acestream.SearchResult {
	return lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Reject(sr.Items, cb)
		return sr
	})
}

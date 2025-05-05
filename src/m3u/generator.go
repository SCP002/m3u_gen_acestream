package m3u

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dlclark/regexp2"
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
	Countries  string
	Languages  string
	EngineAddr string
	TVGName    string
	IconURL    string
}

// Generate writes M3U file based on filtered `searchResults` using settings in config `cfg`.
func Generate(log *logger.Logger, searchResults []acestream.SearchResult, cfg *config.Config) error {
	log.Info("Generating M3U files")

	infohashCheckErrorMap := make(map[string]error)

	for _, playlist := range cfg.Playlists {
		searchResults := remap(log, searchResults, playlist)
		searchResults = filter(log, searchResults, playlist)
		if *playlist.RemoveDeadSources {
			searchResults = removeDead(log, searchResults, playlist, cfg.EngineAddr, infohashCheckErrorMap)
		}

		// Transform []SearchResult to []Entry.
		entries := lo.FlatMap(searchResults, func(sr acestream.SearchResult, _ int) []Entry {
			iconURLs := lo.Map(sr.Icons, func(icon acestream.Icon, _ int) string {
				return icon.URL
			})
			return lo.Map(sr.Items, func(item acestream.Item, _ int) Entry {
				categories := lo.Compact(lo.Uniq(lo.Map(item.Categories, func(category string, _ int) string {
					return strings.ToLower(category)
				})))
				countries := lo.Compact(lo.Uniq(lo.Map(item.Countries, func(country string, _ int) string {
					return strings.ToLower(country)
				})))
				languages := lo.Compact(lo.Uniq(lo.Map(item.Languages, func(language string, _ int) string {
					return strings.ToLower(language)
				})))
				slices.Sort(categories)
				slices.Sort(countries)
				slices.Sort(languages)

				return Entry{
					Name:       item.Name,
					Infohash:   item.Infohash,
					Categories: strings.Join(categories, ";"),
					Countries:  strings.Join(countries, ";"),
					Languages:  strings.Join(languages, ";"),
					EngineAddr: cfg.EngineAddr,
					TVGName:    strings.ReplaceAll(item.Name, " ", "_"),
					IconURL:    lo.FirstOr(iconURLs, ""),
				}
			})
		})

		// Sort entries by names and categories.
		slices.SortStableFunc(entries, func(a, b Entry) int {
			return strings.Compare(a.Name, b.Name)
		})
		slices.SortStableFunc(entries, func(a, b Entry) int {
			return strings.Compare(a.Categories, b.Categories)
		})

		// Write playlists.
		log.InfoFi("Writing output", "playlist", playlist.OutputPath)
		if err := os.MkdirAll(filepath.Dir(playlist.OutputPath), os.ModePerm); err != nil {
			return errors.Wrapf(err, "Make directory structure for playlist %v", playlist.OutputPath)
		}
		var buff bytes.Buffer
		buff.WriteString(playlist.HeaderTemplate)
		templ := template.Must(template.New("").Parse(playlist.EntryTemplate))
		for _, entry := range entries {
			if err := templ.Execute(&buff, entry); err != nil {
				return errors.Wrapf(err, "Execute template for entry %+v", entry)
			}
		}
		if err := os.WriteFile(playlist.OutputPath, buff.Bytes(), 0644); err != nil {
			return errors.Wrapf(err, "Write playlist file %v", playlist.OutputPath)
		}
		log.InfoFi("Written", "sources", len(entries), "playlist", playlist.OutputPath)
	}

	return nil
}

// remap returns `searchResults` with categories changed by criterias in `playlist`.
func remap(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	searchResults = remapCategoryToCategory(log, searchResults, playlist)
	searchResults = remapNameToCategories(log, searchResults, playlist)
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
				log.DebugFi("Changed", "category", category, "to", newCategory, "playlist", playlist.OutputPath)
				category = newCategory
				changed++
			})
			return category
		})
	}
	log.InfoFi("Changed", "categories", changed, "by", "category to category map", "playlist", playlist.OutputPath)
	return searchResults
}

// remapNameToCategories returns `searchResults` with categories changed to respective map values in `playlist`.
func remapNameToCategories(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist) []acestream.SearchResult {
	var changed int
	if len(playlist.NameRxToCategoriesMap) > 0 {
		searchResults = mapAcestreamItems(searchResults, func(item acestream.Item, _ int) acestream.Item {
			maps.ForEveryMatchingRx(playlist.NameRxToCategoriesMap, item.Name, func(newCategories []string) {
				log.DebugFi("Changed", "categories", item.Categories, "to", newCategories, "by name", item.Name,
					"playlist", playlist.OutputPath)
				item.Categories = newCategories
				changed += len(newCategories)
			})
			return item
		})
	}
	log.InfoFi("Changed", "categories", changed, "by", "name to categories map", "playlist", playlist.OutputPath)
	return searchResults
}

// mapAcestreamCategories runs `cb` function for every ace stream item category in `searchResults`.
//
// `cb` function should return modified ace stream category.
//
// `cb` function arguments are:
//   - `category` - ace stream category.
//   - `idx` - category index.
func mapAcestreamCategories(searchResults []acestream.SearchResult,
	cb func(category string, idx int) string) []acestream.SearchResult {
	return mapAcestreamItems(searchResults, func(item acestream.Item, _ int) acestream.Item {
		if len(item.Categories) == 0 {
			item.Categories = []string{""}
		}
		item.Categories = lo.Map(item.Categories, cb)
		return item
	})
}

// mapAcestreamItems runs `cb` function for every ace stream item in `searchResults`.
//
// `cb` function should return modified ace stream item.
//
// `cb` function arguments are:
//   - `item` - ace stream item.
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
		keep := lo.Contains(playlist.StatusFilter, item.Status)
		if !keep {
			log.DebugFi("Rejected", "name", item.Name, "status", item.Status, "playlist", playlist.OutputPath)
		}
		return keep
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
		keep := item.Availability >= playlist.AvailabilityThreshold
		if !keep {
			log.DebugFi("Rejected", "name", item.Name, "availability", item.Availability,
				"playlist", playlist.OutputPath)
		}
		return keep
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
		keep := availabilityUpdatedAgo <= int64(playlist.AvailabilityUpdatedThreshold.Seconds())
		if !keep {
			log.DebugFi("Rejected", "name", item.Name, "availability updated at", item.AvailabilityUpdatedAt,
				"playlist", playlist.OutputPath)
		}
		return keep
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
			if len(item.Categories) == 0 {
				item.Categories = []string{""}
			}
			var keep bool
			if playlist.CategoriesFilterStrict {
				keep = lo.Every(playlist.CategoriesFilter, item.Categories)
			} else {
				keep = lo.Some(item.Categories, playlist.CategoriesFilter)
			}
			if !keep {
				log.DebugFi("Rejected", "name", item.Name, "categories", item.Categories,
					"playlist", playlist.OutputPath)
			}
			return keep
		})
	}
	if len(playlist.CategoriesBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			if len(item.Categories) == 0 {
				item.Categories = []string{""}
			}
			reject := lo.Some(item.Categories, playlist.CategoriesBlacklist)
			if reject {
				log.DebugFi("Rejected", "name", item.Name, "categories", item.Categories,
					"playlist", playlist.OutputPath)
			}
			return reject
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
			if len(item.Languages) == 0 {
				item.Languages = []string{""}
			}
			var keep bool
			if playlist.LanguagesFilterStrict {
				keep = lo.Every(playlist.LanguagesFilter, item.Languages)
			} else {
				keep = lo.Some(item.Languages, playlist.LanguagesFilter)
			}
			if !keep {
				log.DebugFi("Rejected", "name", item.Name, "languages", item.Languages, "playlist", playlist.OutputPath)
			}
			return keep
		})
	}
	if len(playlist.LanguagesBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			if len(item.Languages) == 0 {
				item.Languages = []string{""}
			}
			reject := lo.Some(item.Languages, playlist.LanguagesBlacklist)
			if reject {
				log.DebugFi("Rejected", "name", item.Name, "languages", item.Languages, "playlist", playlist.OutputPath)
			}
			return reject
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
			if len(item.Countries) == 0 {
				item.Countries = []string{""}
			}
			var keep bool
			if playlist.CountriesFilterStrict {
				keep = lo.Every(playlist.CountriesFilter, item.Countries)
			} else {
				keep = lo.Some(item.Countries, playlist.CountriesFilter)
			}
			if !keep {
				log.DebugFi("Rejected", "name", item.Name, "countries", item.Countries, "playlist", playlist.OutputPath)
			}
			return keep
		})
	}
	if len(playlist.CountriesBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			if len(item.Countries) == 0 {
				item.Countries = []string{""}
			}
			reject := lo.Some(item.Countries, playlist.CountriesBlacklist)
			if reject {
				log.DebugFi("Rejected", "name", item.Name, "countries", item.Countries, "playlist", playlist.OutputPath)
			}
			return reject
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
	if len(playlist.NameRxFilter) > 0 {
		searchResults = filterAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.SomeBy(playlist.NameRxFilter, func(rxStr string) bool {
				rx := regexp2.MustCompile(rxStr, regexp2.RE2)
				keep, _ := rx.MatchString(item.Name)
				if !keep {
					log.DebugFi("Rejected", "name", item.Name, "playlist", playlist.OutputPath)
				}
				return keep
			})
		})
	}
	if len(playlist.NameRxBlacklist) > 0 {
		searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
			return lo.SomeBy(playlist.NameRxBlacklist, func(rxStr string) bool {
				rx := regexp2.MustCompile(rxStr, regexp2.RE2)
				reject, _ := rx.MatchString(item.Name)
				if reject {
					log.DebugFi("Rejected", "name", item.Name, "playlist", playlist.OutputPath)
				}
				return reject
			})
		})
	}
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "name", "playlist", playlist.OutputPath)
	return searchResults
}

// removeDead returns `searchResults` without unavailable sources using settings in `playlist` and Ace Stream Engine
// address `engineAddr`.
//
// `infohashCheckErrorMap` is used to cache check results and prevent repeating checks over multiple calls to this
// function.
func removeDead(log *logger.Logger,
	searchResults []acestream.SearchResult,
	playlist config.Playlist,
	engineAddr string,
	infohashCheckErrorMap map[string]error) []acestream.SearchResult {
	log.InfoFi("Removing dead sources", "playlist", playlist.OutputPath)
	prevSources := acestream.GetSourcesAmount(searchResults)
	checker := acestream.NewChecker()
	searchResults = rejectAcestreamItems(searchResults, func(item acestream.Item, _ int) bool {
		cachedResultError, found := infohashCheckErrorMap[item.Infohash]
		link := fmt.Sprintf("http://%v/ace/getstream?infohash=%v", engineAddr, item.Infohash)
		if found {
			if cachedResultError == nil {
				log.InfoFi("Keep", "name", item.Name, "link", link)
				return false
			} else {
				log.WarnFi("Reject", "name", item.Name, "link", link, "reason", cachedResultError)
				return true
			}
		}
		err := checker.IsAvailable(link, *playlist.CheckRespTimeout, *playlist.UseMpegTsAnalyzer)
		infohashCheckErrorMap[item.Infohash] = err
		if err != nil {
			log.WarnFi("Reject", "name", item.Name, "link", link, "reason", err)
			return true
		}
		log.InfoFi("Keep", "name", item.Name, "link", link)
		return false
	})
	currSources := acestream.GetSourcesAmount(searchResults)
	log.InfoFi("Rejected", "sources", prevSources-currSources, "by", "response", "playlist", playlist.OutputPath)
	return searchResults
}

// filterAcestreamItems runs `cb` function for every ace stream item in `searchResults`.
//
// `cb` function should return 'true' if item should stay in `searchResults`.
//
// `cb` function arguments are:
//   - `item` - ace stream item.
//   - `idx` - current item index.
func filterAcestreamItems(searchResults []acestream.SearchResult,
	cb func(item acestream.Item, idx int) bool) []acestream.SearchResult {
	return lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Filter(sr.Items, cb)
		return sr
	})
}

// rejectAcestreamItems runs `cb` function for every ace stream item in `searchResults`.
//
// `cb` function should return 'true' if item should be removed from `searchResults`.
//
// `cb` function arguments are:
//   - `item` - ace stream item.
//   - `idx` - item index.
func rejectAcestreamItems(searchResults []acestream.SearchResult,
	cb func(item acestream.Item, idx int) bool) []acestream.SearchResult {
	return lo.Map(searchResults, func(sr acestream.SearchResult, _ int) acestream.SearchResult {
		sr.Items = lo.Reject(sr.Items, cb)
		return sr
	})
}

package acestream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"m3u_gen_acestream/util/logger"
)

// Engine respresents handler for AceStream engine to interract with it using REST API.
type Engine struct {
	log        *logger.Logger
	httpClient *http.Client
	addr       string
	pageSize   int
}

// SearchResult represents available channels response to search request to engine.
type SearchResult struct {
	Items []Item `json:"items"`
	Name  string `json:"name"`
	Icons []Icon `json:"icons"`
}

// UnmarshalJSON implements json.Unmarshaller interface and made to deal with problematic Name field which can be both
// number or string.
func (sr *SearchResult) UnmarshalJSON(bytes []byte) error {
	type Embed SearchResult
	tmp := struct {
		Embed
		Name any `json:"name"`
	}{Embed: Embed(*sr)}

	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}
	*sr = SearchResult(tmp.Embed)
	sr.Name = fmt.Sprintf("%v", tmp.Name)

	return nil
}

// Item represents item in SearchResult.
type Item struct {
	Status                int      `json:"status"`
	Languages             []string `json:"languages"`
	Name                  string   `json:"name"`
	Countries             []string `json:"countries"`
	Infohash              string   `json:"infohash"`
	ChannelID             int      `json:"channel_id"`
	AvailabilityUpdatedAt int64    `json:"availability_updated_at"`
	Availability          float64  `json:"availability"`
	Categories            []string `json:"categories"`
}

// Icon represents icon in SearchResult.
type Icon struct {
	URL  string `json:"url"`
	Type int    `json:"type"`
}

// versionResp represents response to version request to engine.
type versionResp struct {
	Result struct {
		Code     int    `json:"code"`
		Platform string `json:"platform"`
		Version  string `json:"version"`
	} `json:"result"`
	Error any `json:"error"`
}

// searchResp represents response to search request to engine.
type searchResp struct {
	Result struct {
		Total   int            `json:"total"`
		Results []SearchResult `json:"results"`
		Time    float64        `json:"time"`
	} `json:"result"`
}

// NewEngine returns new engine handler with it's address at `addr`, which should be in format of 'host:port'.
func NewEngine(log *logger.Logger, httpClient *http.Client, addr string) *Engine {
	return &Engine{log: log, httpClient: httpClient, addr: addr, pageSize: 200}
}

// WaitForConnection blocks current goroutine until engine responds with version info or until `ctx` deadline exceedes.
func (e Engine) WaitForConnection(ctx context.Context) {
	e.log.Info("Connecting to engine")

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		select {
		case <-ctx.Done():
			e.log.Error(errors.Wrap(ctx.Err(), "Connect to engine"))
			return
		default:
			break
		}
		url := url.URL{Scheme: "http", Host: e.addr, Path: "webui/api/service", RawQuery: "method=get_version"}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
		if err != nil {
			e.log.Error(errors.Wrap(err, "Connect to engine: Create get_version request"))
			e.log.Debug("Sleeping before reconnect")
			continue
		}
		resp, err := e.httpClient.Do(req)
		if errors.Is(err, context.DeadlineExceeded) {
			e.log.Error(errors.Wrap(err, "Connect to engine: Send get_version request"))
			return
		}
		if err != nil {
			e.log.Error(errors.Wrap(err, "Connect to engine: Send get_version request"))
			e.log.Debug("Sleeping before reconnect")
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			e.log.Error(errors.Wrap(err, "Connect to engine: Read get_version response body"))
			e.log.Debug("Sleeping before reconnect")
			continue
		}
		var version versionResp
		err = json.Unmarshal(body, &version)
		if err != nil {
			e.log.Error(errors.Wrap(err, "Connect to engine: Decode get_version response body as JSON"))
			e.log.Debug("Sleeping before reconnect")
			continue
		}
		if version.Result.Code == 0 || version.Error != nil {
			e.log.Errorf("Connect to engine: Bad engine response: %+v", version)
			e.log.Debug("Sleeping before reconnect")
			continue
		}
		e.log.Info("Engine is running")
		return
	}
}

// SearchAll returns all currently available acestream channels.
func (e Engine) SearchAll(ctx context.Context) ([]SearchResult, error) {
	e.log.Info("Searching for channels")
	results := []SearchResult{}
	for page := 0; ; page++ {
		currResults, err := e.searchAtPage(ctx, page)
		if err != nil {
			return results, errors.Wrapf(err, "Search at page %v", page)
		}
		results = append(results, currResults...)
		if len(currResults) < e.pageSize {
			return results, nil
		}
	}
}

// searchAtPage returns acestream channels at page `page` with maximum page size.
func (e Engine) searchAtPage(ctx context.Context, page int) ([]SearchResult, error) {
	query := fmt.Sprintf("page_size=%v&page=%v", e.pageSize, page)
	url := url.URL{Scheme: "http", Host: e.addr, Path: "search", RawQuery: query}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return []SearchResult{}, errors.Wrap(err, "Create search request")
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return []SearchResult{}, errors.Wrap(err, "Send search request")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []SearchResult{}, errors.Wrap(err, "Read search response body")
	}
	var out searchResp
	err = json.Unmarshal(body, &out)
	if err != nil {
		return []SearchResult{}, errors.Wrap(err, "Decode search response body as JSON")
	}
	e.log.InfoFi("Received", "channels", len(out.Result.Results), "sources", GetSourcesAmount(out.Result.Results),
		"page", page)
	return out.Result.Results, nil
}

// GetSourcesAmount returns total amount of Item's in `searchResults`.
func GetSourcesAmount(searchResults []SearchResult) int {
	return lo.SumBy(searchResults, func(sr SearchResult) int {
		return len(sr.Items)
	})
}

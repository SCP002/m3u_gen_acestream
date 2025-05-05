package acestream

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/ziutek/dvb/ts"

	"m3u_gen_acestream/util/network"
)

// Checker respresents availability checker.
type Checker struct {
	httpClient *http.Client
}

// NewChecker return new availability checker.
func NewChecker() *Checker {
	return &Checker{httpClient: network.NewHTTPClient(0)}
}

// IsAvailable returns nil error if `link` responds with content or non-nil error otherwise.
//
// If engine does not respond with content in `timeout`, it will return error.
//
// If `analyzeMpegTs` is true, try to parse response as a TS packet and return error if it fails.
func (c Checker) IsAvailable(link string, timeout time.Duration, analyzeMpegTs bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return errors.Wrap(err, "Create request")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Execute request")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return errors.Newf("Responded with status %v", resp.Status)
	}
	buff := make([]byte, ts.PktLen*10)
	read, err := resp.Body.Read(buff)
	if err != nil {
		return errors.Wrap(err, "Read response body")
	}
	if read == 0 {
		return errors.New("Read 0 bytes from body")
	}
	if !analyzeMpegTs {
		return nil
	}
	streamReader := ts.NewPktStreamReader(bytes.NewReader(buff))
	pkt := ts.AsPkt(make([]byte, ts.PktLen))
	err = streamReader.ReadPkt(pkt)
	if err != nil {
		return errors.Wrap(err, "Read packet")
	}
	return nil
}

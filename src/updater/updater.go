package updater

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/cockroachdb/errors"
	"github.com/fynelabs/selfupdate"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"m3u_gen_acestream/util/logger"
)

// Updater represents update handler for this program.
type Updater struct {
	log        *logger.Logger
	httpClient *http.Client
}

// New returns new updater.
func New(log *logger.Logger, httpClient *http.Client) *Updater {
	return &Updater{log: log, httpClient: httpClient}
}

// Release represents github release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents github release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Update checks if latest release is equal to `currentVersion`. If so, it will exit the program with code 0.
// Otherwise it will update current executable to the latest version.
func (u Updater) Update(currentVersion string) error {
	release, err := u.getLatestRelease()
	if err != nil {
		return errors.Wrap(err, "Get latest release info")
	}

	if release.TagName == currentVersion {
		u.log.Info("No update available")
		os.Exit(0)
	}

	u.log.InfoFi("Update is available", "new version", release.TagName)

	downloadUrl, err := getDownloadUrl(release.Assets)
	if err != nil {
		return errors.Wrap(err, "Get download URL")
	}

	u.log.InfoFi("Found binary", "URL", downloadUrl)

	err = u.doUpdate(downloadUrl)
	if err != nil {
		return err
	}

	return nil
}

// doUpdate updates current binary to the one provided at `url`.
func (u Updater) doUpdate(url string) error {
	u.log.Info("Downloading update")
	resp, err := u.httpClient.Get(url)
	if err != nil {
		return errors.Wrap(err, "Send get request")
	}
	defer resp.Body.Close()

	u.log.Info("Installing update. This program will terminate to do it.")
	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		return errors.Wrap(err, "Update executable")
	}

	return nil
}

// getLatestRelease returns latest release info.
func (u Updater) getLatestRelease() (Release, error) {
	resp, err := u.httpClient.Get("https://api.github.com/repos/SCP002/m3u_gen_acestream/releases/latest")
	if err != nil {
		return Release{}, errors.Wrap(err, "Send get request")
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Release{}, errors.Wrap(err, "Read response body")
	}
	var release Release
	err = json.Unmarshal(respBody, &release)
	if err != nil {
		return Release{}, errors.Wrap(err, "Decode response body as JSON")
	}
	return release, nil
}

// getDownloadUrl returns proper URL to download a binary in `assets`.
func getDownloadUrl(assets []Asset) (string, error) {
	requiredName := fmt.Sprintf("m3u_gen_acestream_%v_%v", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		requiredName += ".exe"
	}
	asset, found := lo.Find(assets, func(asset Asset) bool {
		return asset.Name == requiredName
	})
	if found {
		return asset.BrowserDownloadURL, nil
	}
	return "", errors.Newf("No URL found for binary matching the name: %v", requiredName)
}

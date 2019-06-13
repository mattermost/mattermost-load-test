package ltops

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func GetFileOrURL(reference string) ([]byte, error) {
	// Support a local file path
	if _, err := os.Stat(reference); err == nil {
		return getFile(reference)
	}

	// Support an explicit URL
	if strings.HasPrefix(reference, "http") {
		return getURL(reference)
	}

	return nil, fmt.Errorf("failed to resolve %s to a file or URL", reference)
}

func GetMattermostFileOrURL(reference string) ([]byte, error) {
	// Support a local file path
	if _, err := os.Stat(reference); err == nil {
		return getFile(reference)
	}

	// Support an explicit URL
	if strings.HasPrefix(reference, "http") {
		return getURL(reference)
	}

	// Support the latest release from master
	if reference == "master" {
		tryURL := "https://releases.mattermost.com/mattermost-platform/master/mattermost-enterprise-linux-amd64.tar.gz"
		logrus.Infof("resolved %s to %s", reference, tryURL)
		return getURL(tryURL)
	}

	// Support a PR#
	if matched, _ := regexp.MatchString("^[0-9]+$", reference); matched {
		tryURL := fmt.Sprintf("https://releases.mattermost.com/mattermost-platform-pr/%s/mattermost-enterprise-linux-amd64.tar.gz", reference)
		logrus.Infof("resolved %s to %s", reference, tryURL)
		return getURL(tryURL)
	}

	// Support a named branch or release
	tryURL := fmt.Sprintf("https://releases.mattermost.com/%s/mattermost-%s-linux-amd64.tar.gz", reference, reference)
	logrus.Infof("resolved %s to %s", reference, tryURL)

	return getURL(tryURL)
}

func GetLoadtestFileOrURL(reference string) ([]byte, error) {
	// Support a local file path
	if _, err := os.Stat(reference); err == nil {
		return getFile(reference)
	}

	// Support an explicit URL
	if strings.HasPrefix(reference, "http") {
		return getURL(reference)
	}

	// Support the latest release from master
	if reference == "master" {
		tryURL := "https://releases.mattermost.com/mattermost-load-test/mattermost-load-test.tar.gz"
		logrus.Infof("resolved %s to %s", reference, tryURL)
		return getURL(tryURL)
	}
	// Support a named branch.
	tryURL := fmt.Sprintf("https://releases.mattermost.com/loadtest-pr/%s/mattermost-load-test.tar.gz", reference)
	logrus.Info(tryURL)
	if data, err := getURL(tryURL); err == nil {
		logrus.Infof("resolved %s to %s", reference, tryURL)
		return data, nil
	}

	return nil, fmt.Errorf("failed to resolve %s to a loadtest URL", reference)
}

func getFile(file string) ([]byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, errors.Wrap(err, "unable to open file "+file)
	}
	defer f.Close()

	buffer := bytes.NewBuffer(nil)
	io.Copy(buffer, f)

	return buffer.Bytes(), nil
}

func getURL(url string) ([]byte, error) {
	logrus.Debugf("loading %s", url)

	response, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Can't get file at URL: "+url)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("fetching %s failed with status code %d", url, response.StatusCode)
	}

	buffer := bytes.NewBuffer(nil)
	io.Copy(buffer, response.Body)

	return buffer.Bytes(), nil
}

package ltops

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func GetFileOrURL(fileOrUrl string) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if strings.HasPrefix(fileOrUrl, "http") {
		response, err := http.Get(fileOrUrl)
		if err != nil {
			return nil, errors.Wrap(err, "Can't get file at URL: "+fileOrUrl)
		}
		defer response.Body.Close()

		io.Copy(buffer, response.Body)

		return buffer.Bytes(), nil
	} else {
		f, err := os.Open(fileOrUrl)
		if err != nil {
			return nil, errors.Wrap(err, "unable to open file "+fileOrUrl)
		}
		defer f.Close()

		io.Copy(buffer, f)

		return buffer.Bytes(), nil
	}
}

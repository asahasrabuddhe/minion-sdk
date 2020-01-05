package minonsdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gosuri/uilive"
	"go.ajitem.com/bindiff"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

const UserAgent = "pkgUpdater/1.0.0"

var UpdateNotAvailable = errors.New("no update available")

type Options struct {
	AppId          string
	Version        string
	Channel        string
	UpdateCheckURL string
}

func Check(opts Options) ([]byte, error) {
	if opts.AppId == "" {
		return nil, errors.New("pkg updater sdk: app id cannot be empty")
	}

	if opts.Version == "" {
		return nil, errors.New("pkg updater sdk: version cannot be empty")
	}

	if opts.Channel == "" {
		opts.Channel = "stable"
	}

	var client http.Client
	client.Transport = NewUserAgentTransport(UserAgent, client.Transport)

	payload, err := json.Marshal(map[string]interface{}{
		"app_id":       opts.AppId,
		"version":      opts.Version,
		"channel":      opts.Channel,
		"os":           runtime.GOOS,
		"architecture": runtime.GOARCH,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, opts.UpdateCheckURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusNoContent:
		// no update available
		return nil, UpdateNotAvailable
	case http.StatusOK:
		// got update, download
		writer := uilive.New()
		writer.Start()

		total, err := strconv.Atoi(response.Header.Get("Content-Length"))
		if err != nil {
			return nil, err
		}

		downloaded := 0

		reader := &progressReader{
			Reader: response.Body,
			Reporter: func(r int) {
				downloaded += r
				_, _ = fmt.Fprintf(writer, "Downloading.. (%d/%d) bytes\n", downloaded, total)
			},
		}

		body, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		writer.Stop()

		return body, nil
	default:
		// error
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(string(body))
	}
}

func Apply(b []byte) error {
	oldPath, err := os.Executable()
	if err != nil {
		return err
	}

	newPath := fmt.Sprintf("%s/new_%s", filepath.Dir(oldPath), filepath.Base(oldPath))
	intermediatePath := fmt.Sprintf("%s/old_%s", filepath.Dir(oldPath), filepath.Base(oldPath))

	oldFile, err := os.Open(oldPath)
	if err != nil {
		return err
	}
	defer oldFile.Close()

	newFile, err := os.OpenFile(newPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer newFile.Close()

	err = bindiff.Patch(oldFile, newFile, bytes.NewReader(b))
	if err != nil {
		return err
	}

	err = os.Rename(oldPath, intermediatePath)
	if err != nil {
		return err
	}

	err = os.Rename(newPath, oldPath)
	if err != nil {
		// handle unsuccessful move
		return err
	}

	err = os.Remove(intermediatePath)
	if err != nil {
		return err
	}

	return nil
}

func Reload() error {
	binSelf, err := os.Executable()
	if err != nil {
		return err
	}

	err = syscall.Exec(binSelf, append([]string{binSelf}, os.Args[1:]...), os.Environ())
	if err != nil {
		return fmt.Errorf("pkg updater sdk: cannot restart: %v", err)
	}

	return nil
}
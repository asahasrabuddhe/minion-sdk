package sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/gosuri/uilive"
	"go.ajitem.com/bindiff"
)

const UserAgent = "minion-updater/1.0.0"

var UpdateNotAvailable = errors.New("no update available")

type Options struct {
	AppId          string
	Version        string
	Channel        string
	UpdateCheckURL string
	NewVersion     string
}

type UpdateResponse struct {
	Success bool   `json:"success"`
	Version string `json:"version"`
}

type Minion struct {
	client http.Client
}

func New() *Minion {
	m := &Minion{}

	m.client.Transport = NewUserAgentTransport(UserAgent, m.client.Transport)

	return m
}

func (m *Minion) Check(opts Options) (*UpdateResponse, error) {
	if opts.AppId == "" {
		return nil, errors.New("pkg updater sdk: app id cannot be empty")
	}

	if opts.Version == "" {
		return nil, errors.New("pkg updater sdk: version cannot be empty")
	}

	if opts.Channel == "" {
		opts.Channel = "stable"
	}

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

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/check", opts.UpdateCheckURL), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	response, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusNoContent:
		// no update available
		return nil, UpdateNotAvailable
	case http.StatusOK:
		// update available
		var res UpdateResponse
		err := json.Unmarshal(body, &res)
		if err != nil {
			return nil, err
		}

		opts.NewVersion = res.Version

		return &res, nil
	default:
		// error
		return nil, errors.New(string(body))
	}
}

func (m *Minion) Download(opts Options) ([]byte, error) {
	// got update, download

	payload, err := json.Marshal(map[string]interface{}{
		"app_id":       opts.AppId,
		"version":      opts.Version,
		"new_version":  opts.NewVersion,
		"channel":      opts.Channel,
		"os":           runtime.GOOS,
		"architecture": runtime.GOARCH,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/download", opts.UpdateCheckURL), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Close = true

	response, err := m.client.Do(req)
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

		reader := &ProgressReader{
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

func (m *Minion) Apply(b []byte) error {
	oldPath, err := os.Executable()
	if err != nil {
		return err
	}

	newPath := path.Join(filepath.Dir(oldPath), fmt.Sprintf("new_%s", filepath.Base(oldPath)))
	intermediatePath := path.Join(filepath.Dir(oldPath), fmt.Sprintf("old_%s", filepath.Base(oldPath)))

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

	err = RenameFile(oldPath, intermediatePath)
	if err != nil {
		return err
	}

	err = RenameFile(newPath, oldPath)
	if err != nil {
		// handle unsuccessful move
		return err
	}

	err = RemoveFile(intermediatePath)
	if err != nil {
		return err
	}

	return nil
}

func (m *Minion) Reload() error {
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

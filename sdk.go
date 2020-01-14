package sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gosuri/uilive"
	"go.ajitem.com/bindiff"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
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

func (m *Minion) CurrentPath() (string, error) {
	return os.Executable()
}

func(m *Minion) OldPath() (string, error) {
	path, err := m.CurrentPath()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%c.%s.old", filepath.Dir(path), os.PathSeparator, filepath.Base(path)), nil
}

func(m *Minion) NewPath() (string, error) {
	path, err := m.CurrentPath()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s%c.%s.new", filepath.Dir(path), os.PathSeparator, filepath.Base(path)), nil
}

func (m *Minion) Apply(b []byte) error {
	targetPath, err := m.CurrentPath()
	if err != nil {
		return err
	}

	targetFile, err := os.Open(targetPath)
	if err != nil {
		return err
	}

	newPath, err := m.NewPath()
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	err = bindiff.Patch(targetFile, &buf, bytes.NewReader(b))
	if err != nil {
		return err
	}

	err = targetFile.Close()
	if err != nil {
		return err
	}

	newFile, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = io.Copy(newFile, &buf)
	if err != nil {
		return err
	}

	err = newFile.Close()
	if err != nil {
		return err
	}

	oldPath, err := m.OldPath()
	if err != nil {
		return err
	}

	_ = os.Remove(oldPath)

	err = os.Rename(targetPath, oldPath)
	if err != nil {
		return err
	}

	err = os.Rename(newPath, targetPath)
	if err != nil {
		return err
	}

	err = os.Remove(oldPath)
	if err != nil {
		return hideFile(oldPath)
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

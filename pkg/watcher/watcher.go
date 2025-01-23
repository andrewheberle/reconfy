package watcher

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/a8m/envsubst"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	input   string
	output  string
	watcher *fsnotify.Watcher
	client  *http.Client
	webhook string
	done    chan bool

	// File permission bits for output file
	FileMode os.FileMode
}

func NewWatcher(input, output, webhook string) (*Watcher, error) {
	if input == "" || webhook == "" {
		return nil, fmt.Errorf("input and webhook must be provided")
	}

	// validate url
	if _, err := url.Parse(webhook); err != nil {
		return nil, err
	}

	// ensure input and output are not the same
	if input == output {
		return nil, fmt.Errorf("input and output path cannot be the same")
	}

	// create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		input:    input,
		output:   output,
		watcher:  watcher,
		client:   &http.Client{Timeout: time.Second * 5},
		webhook:  webhook,
		done:     make(chan bool),
		FileMode: 0644,
	}, nil
}

func (w *Watcher) Close() error {
	// signal done to watcher loop
	defer func() { close(w.done) }()

	// close watcher
	return w.watcher.Close()
}

func (w *Watcher) Watch() error {
	slog.Info("starting watch", "input", w.input, "webhook", w.webhook)

	// start watcher loop
	go w.watchLoop()

	// add path to watcher
	slog.Debug("adding path to watcher", "path", w.input)
	if err := w.watcher.Add(filepath.Dir(w.input)); err != nil {
		return fmt.Errorf("could not start watching: %w", err)
	}

	// generate output first off
	slog.Debug("doing initial envsubst if required")
	if err := w.envsubst(); err != nil {
		return fmt.Errorf("could not perform initial envsubst: %w", err)
	}

	slog.Debug("waiting here until done")
	<-w.done

	slog.Debug("watch done")
	return nil
}

func (w *Watcher) watchLoop() {
	var (
		mu sync.Mutex

		wait = 100 * time.Millisecond

		timers = make(map[string]*time.Timer)
	)
	slog.Debug("starting watch loop")
	defer func() {
		slog.Debug("watch loop finishing")
	}()

	for {
		select {
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}

			slog.Error("error from watcher", "error", err)

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			slog.Debug("event from watcher", "name", event.Name, "op", event.Op)
			if !event.Has(fsnotify.Write) && event.Has(fsnotify.Create) {
				continue
			}

			slog.Info("change detected", "input", w.input)

			// get timer
			mu.Lock()
			t, ok := timers[event.Name]
			mu.Unlock()

			// no timer found
			if !ok {
				t = time.AfterFunc(math.MaxInt64, func() {
					// clean up timer
					defer func() {
						mu.Lock()
						delete(timers, event.Name)
						mu.Unlock()
					}()

					if err := w.envsubst(); err != nil {
						slog.Error("problem during envsubst", "error", err, "input", w.input, "output", w.output)
						return
					}

					// do webhook request
					res, err := w.client.Get(w.webhook)
					if err != nil {
						slog.Error("error calling webhook", "error", err, "input", w.input, "webhook", w.webhook)
						return
					}

					// close body immediately
					res.Body.Close()

					// check response
					if res.StatusCode != http.StatusOK {
						slog.Error("got a non 200 response", "response", res.StatusCode)
						return
					}
				})
				t.Stop()

				mu.Lock()
				timers[event.Name] = t
				mu.Unlock()
			}

			// reset timer
			t.Reset(wait)
		}
	}
}

func (w *Watcher) envsubst() error {
	// without an output then return
	if w.output == "" {
		return nil
	}

	// do environment variable substitutions
	data, err := envsubst.ReadFile(w.input)
	if err != nil {
		return err
	}

	// write new config file
	if err := writefile(w.output, data, w.FileMode); err != nil {
		return err
	}

	return nil
}

func writefile(filename string, data []byte, mode os.FileMode) error {
	// Create a temporary file in the same directory
	dir := filepath.Dir(filename)
	tempFile, err := os.CreateTemp(dir, "temp*")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	// Write data to the temporary file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return err
	}

	// Close the temporary file
	if err := tempFile.Close(); err != nil {
		return err
	}

	// Set permissions
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		return err
	}

	// Rename the temporary file to the target filename
	return os.Rename(tempFile.Name(), filename)
}

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
	input         string
	output        string
	client        *http.Client
	webhookUrl    *url.URL
	webhookMethod string
	done          chan bool

	// File permission bits for output file
	FileMode os.FileMode
}

func NewWatcher(input string, opts ...WatcherOption) (*Watcher, error) {
	if input == "" {
		return nil, fmt.Errorf("input must not be empty")
	}

	w := new(Watcher)

	// set defaults
	w.FileMode = 0644
	w.client = &http.Client{Timeout: time.Second * 5}
	w.done = make(chan bool)

	// apply options
	for _, o := range opts {
		o(w)
	}

	// ensure output or webhook url (or both) are set
	if w.output == "" && w.webhookUrl == nil {
		return nil, fmt.Errorf("a valid webhook url or an output path (or both) must be provided")
	}

	// clean up input path
	w.input = filepath.Clean(input)

	// ensure input and output are not the same
	if w.input == w.output {
		return nil, fmt.Errorf("input and output path cannot be the same")
	}

	return w, nil
}

func (w *Watcher) Close() error {
	// signal done to watcher loop
	close(w.done)

	return nil
}

func (w *Watcher) Watch() error {
	slog.Info("starting watch", "input", w.input, "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod)

	// Create a new watcher.
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create watcher: %w", err)
	}
	defer watch.Close()

	// generate output first off
	slog.Debug("doing initial envsubst if required")
	if err := w.envsubst(); err != nil {
		return fmt.Errorf("could not perform initial envsubst: %w", err)
	}

	// start watcher loop
	go w.watchLoop(watch)

	slog.Debug("adding path to watcher", "path", w.input)
	// check input file exists
	stat, err := os.Lstat(w.input)
	if err != nil {
		return fmt.Errorf("could not stat input file %s: %w", w.input, err)
	}

	// make sure it's not a directory
	if stat.IsDir() {
		return fmt.Errorf("%s is a directory", w.input)
	}

	// add path to watcher
	if err := watch.Add(filepath.Dir(w.input)); err != nil {
		return fmt.Errorf("could not add path to watcher: %w", err)
	}

	slog.Debug("waiting here until done")
	<-w.done

	slog.Debug("watch done")
	return nil
}

func (w *Watcher) watchLoop(watch *fsnotify.Watcher) {
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
		slog.Debug("waiting for events...")

		select {
		// read from Errors
		case err, ok := <-watch.Errors:
			if !ok {
				slog.Debug("channel was closed")
				return
			}

			slog.Error("error from watcher", "error", err)

			// read from Events
		case event, ok := <-watch.Events:
			if !ok {
				slog.Debug("channel was closed")
				return
			}

			// check if op was one we are interested in
			if !event.Has(fsnotify.Write) && event.Has(fsnotify.Create) {
				slog.Debug("event was not of a type we are watching for", "op", event.Op)
				continue
			}

			// check if change was for our watched file
			if event.Name != w.input {
				slog.Debug("event was not for our watched file", "name", event.Name)
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

					// finish here if no url is set
					if w.webhookUrl == nil {
						return
					}

					// do webhook request
					slog.Info("sending request to webhook", "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod)
					res, err := w.client.Do(&http.Request{
						Method: w.webhookMethod,
						URL:    w.webhookUrl,
					})
					if err != nil {
						slog.Error("error calling webhook", "error", err, "input", w.input, "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod)
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

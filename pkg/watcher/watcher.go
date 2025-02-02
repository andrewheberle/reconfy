package watcher

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/a8m/envsubst"
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	inputs        []string
	output        string
	client        *http.Client
	webhookUrl    *url.URL
	webhookMethod string
	done          chan bool
	fileonly      bool

	// File permission bits for output file
	FileMode os.FileMode
}

const (
	DefaultFileMode      os.FileMode = 0644
	DefaultWebhookMethod             = http.MethodPost
)

func NewWatcher(input interface{}, opts ...WatcherOption) (*Watcher, error) {
	if input == nil {
		return nil, fmt.Errorf("input must not be nil")
	}

	inputs := make([]string, 0)
	switch v := input.(type) {
		case []string:
		inputs = v
		case string:
		inputs = []string{v}
		default:
		return nil, fmt.Errorf("input must be a string or []string")
	}

	w := new(Watcher)

	// set defaults
	w.webhookMethod = DefaultWebhookMethod
	w.FileMode = DefaultFileMode
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

	// clean up input paths
	w.inputs = make([]string, 0)
	for v := range inputs {
		w.inputs = append(w.inputs, filepath.Clean(v))
	}

	// ensure input and output are not the same
	if w.inputs[0] == w.output {
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
	slog.Info("starting watch", "inputs", w.inputs, "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod, "watch-fileonly", w.fileonly)

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

	for input := range w.inputs {
		slog.Debug("adding path to watcher", "path", input)
		// check input file exists
		stat, err := os.Lstat(input)
		if err != nil {
			return fmt.Errorf("could not stat input file %s: %w", input, err)
		}
		
		// make sure it's not a directory
		if stat.IsDir() {
			return fmt.Errorf("%s is a directory", input)
		}
		
		// add path to watcher
		if w.fileonly {
			if err := watch.Add(input); err != nil {
				return fmt.Errorf("could not add path to watcher: %w", err)
			}
		} else {
			if err := watch.Add(filepath.Dir(input)); err != nil {
				return fmt.Errorf("could not add path to watcher: %w", err)
			}
		}
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
			if slices.Contains(w.inputs, event.Name) {
				slog.Debug("event was not for one of our watched files", "name", event.Name)
				continue
			}

			slog.Debug("change detected (may be dupes)", "inputs", w.inputs, "op", event.Op)

			// get timer
			mu.Lock()
			t, ok := timers[event.Name]
			mu.Unlock()

			// no timer found
			if !ok {
				t = time.AfterFunc(math.MaxInt64, func() {
					// we are doing some work
					slog.Info("change detected and actioned", "inputs", w.inputs, "op", event.Op)

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
					if err := w.webhook(); err != nil {
						slog.Error("error during webhook call", "error", err, "input", w.input, "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod)
						return
					}

					slog.Info("webhook request completed successfully", "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod)
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
	data, err := envsubst.ReadFile(w.inputs[0])
	if err != nil {
		return err
	}

	// write new config file
	if err := writefile(w.output, data, w.FileMode); err != nil {
		return err
	}

	return nil
}

func (w *Watcher) webhook() error {
	// finish here if no url is set
	if w.webhookUrl == nil {
		return nil
	}

	// do webhook request
	slog.Info("sending request to webhook", "webhook-url", w.webhookUrl, "webhook-method", w.webhookMethod)
	res, err := w.client.Do(&http.Request{
		Method: w.webhookMethod,
		URL:    w.webhookUrl,
	})
	if err != nil {
		return err
	}

	// close body immediately
	res.Body.Close()

	// check response
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("got a non 200 response from webhook: %d", res.StatusCode)
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

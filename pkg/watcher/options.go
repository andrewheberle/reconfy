package watcher

import (
	"os"
	"net/url"
)

type WatcherOption func(*Watcher)

func WithFileMode(filemode os.FileMode) WatcherOption {
	return func(w *Watcher) {
		w.FileMode = filemode
	}
}

func WithWebhookUrl(webhookUrl string) WatcherOption {
	// parse url
	u, err := url.Parse(webhookUrl)
	if err != nil {
		// set as nil if parse fails
		u = nil
	}
	
	return func(w *Watcher) {
		w.webhookUrl = u
	}
}

func WithWebhookMethod(method string) WatcherOption {
	return func(w *Watcher) {
		w.webhookMethod = method
	}
}

func WithWebhookOutput(output string) WatcherOption {
	return func(w *Watcher) {
		w.output = output
	}
}

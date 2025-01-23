package watcher

import "os"

type WatcherOption func(*Watcher)

func WithFileMode(filemode os.FileMode) WatcherOption {
	return func(w *Watcher) {
		w.FileMode = filemode
	}
}

func WithWebhookMethod(method string) WatcherOption {
	return func(w *Watcher) {
		w.webhookMethod = method
	}
}

package watcher

import "fs"

type WatcherOption func(*Watcher)

func WithFileMode(filemode fs.FileMode) WatcherOption {
	return func(w *Watcher) {
		w.FileMode = filemode
	}
}

func WithWebhookMethod(method string) WatcherOption {
	return func(w *Watcher) {
		w.webhookMethod = method
	}
}

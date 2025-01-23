package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/andrewheberle/reconfy/pkg/watcher"
	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {
	// command line args
	pflag.String("input", "", "Input file path to watch for changes")
	pflag.String("output", "", "Output path for environment variable substitutions")
	pflag.String("webhook-url", "http://localhost:8080", "Webhook URL")
	pflag.String("webhook-method", http.MethodPost, "Webhook method")
	pflag.Bool("debug", false, "Enable debug logging")
	pflag.Parse()

	// pull from env and bind flags to viper
	viper.SetEnvPrefix("reconfy")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(pflag.CommandLine)

	// set up logger
	logLevel := new(slog.LevelVar)
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(h))
	if viper.GetBool("debug") {
		logLevel.Set(slog.LevelDebug)
	}

	// set up watcher
	w, err := watcher.NewWatcher(
		viper.GetString("input"),
		viper.GetString("output"),
		viper.GetString("webhook-url"),
		viper.GetString("webhook-method"),
	)
	if err != nil {
		slog.Error("could not create watcher",
			"error", err,
			"input", viper.GetString("input"),
			"output", viper.GetString("output"),
			"webhook-url", viper.GetString("webhook-url"),
			"webhook-method", viper.GetString("webhook-method"),
		)
		os.Exit(1)
	}
	defer w.Close()

	var g run.Group
	{
		g.Add(func() error {
			return w.Watch()
		}, func(err error) {
		})
	}

	if err := g.Run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

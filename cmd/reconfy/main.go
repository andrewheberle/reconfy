package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thanos-io/thanos/pkg/reloader"
)

func main() {
	// command line args
	pflag.String("input", "", "Input file path to watch for changes")
	pflag.String("output", "", "Output path for environment variable substitutions")
	pflag.String("webhook", "http://localhost:8080", "Webhook URL")
	pflag.StringSlice("watchdirs", []string{}, "Additional directories to watch for changes")
	pflag.String("metrics.listen", "", "Listen address for metrics")
	pflag.String("metrics.path", "/metrics", "Path for metrics")
	pflag.Bool("ignoremissing", false, "Ignore missing environment variables when performing substitutions")
	pflag.String("config", "", "Configuration file to load (supports multiple reloaders)")
	pflag.Parse()

	// pull from env and bind flags to viper
	viper.SetEnvPrefix("reconfy")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(pflag.CommandLine)

	// set up logger
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)

	// parse config/flags and get list of reloaders
	reloaders, err := LoadConfig(viper.GetViper(), viper.GetString("config"))
	if err != nil {
		logger.Log("msg", "problem loading config", "error", err)
		os.Exit(1)
	}

	// set up metrics
	var srv *http.Server
	globalRegistry := prometheus.NewRegistry()
	if listen := viper.GetString("metrics.listen"); listen != "" {
		r := http.NewServeMux()
		r.Handle(viper.GetString("metrics.path"), promhttp.HandlerFor(globalRegistry, promhttp.HandlerOpts{Registry: globalRegistry}))
		srv = &http.Server{
			Addr:         listen,
			Handler:      r,
			ReadTimeout:  time.Second * 2,
			WriteTimeout: time.Second * 2,
		}
	}

	var g run.Group
	{
		// loop through reloaders
		for _, rl := range reloaders {
			// set up specific logger for reloader
			thisLogger := logger
			if rl.Name != "" {
				thisLogger = log.With(logger, "name", rl.Name)
			}

			// parse provided url
			u, err := url.Parse(rl.Webhook)
			if err != nil {
				thisLogger.Log("msg", "error with webhook url", "error", err)
				os.Exit(1)
			}

			// set reloader options
			rOptions := &reloader.Options{
				ReloadURL:                     u,
				CfgFile:                       rl.Input,
				CfgOutputFile:                 rl.Output,
				WatchedDirs:                   rl.Watchdirs,
				WatchInterval:                 3 * time.Minute,
				RetryInterval:                 5 * time.Second,
				DelayInterval:                 1 * time.Second,
				TolerateEnvVarExpansionErrors: viper.GetBool("ignoremissing"),
			}

			// set up reloader
			name := rl.Name
			if name == "" {
				name = rl.Input
			}
			labels := prometheus.Labels{"reloader": name}
			reg := prometheus.WrapRegistererWith(labels, globalRegistry)
			r := reloader.New(thisLogger, reg, rOptions)

			// add reloader to run group
			ctx, cancel := context.WithCancel(context.Background())
			g.Add(func() error {
				return r.Watch(ctx)
			}, func(err error) {
				cancel()
			})
		}

		// add metrics server
		if srv != nil {
			g.Add(func() error {
				logger.Log("msg", "starting metrics HTTP server", "listen", viper.GetString("metrics.listen"), "path", viper.GetString("metrics.path"))
				return srv.ListenAndServe()
			}, func(err error) {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					srv.Shutdown(ctx)
					cancel()
				}()
			})
		}
	}

	if err := g.Run(); err != nil {
		logger.Log("msg", "application error", "error", err)
		os.Exit(1)
	}
}

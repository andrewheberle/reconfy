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
	pflag.StringSlice("watch-dirs", []string{}, "Additional directories to watch for changes")
	pflag.String("metrics-listen", "", "Listen address for metrics")
	pflag.String("metrics-path", "/metrics", "Path for metrics")
	pflag.Bool("ignore-missing", false, "Ignore missing environment variables when performing substitutions")
	pflag.Parse()

	// pull from env and bind flags to viper
	viper.SetEnvPrefix("reconfy")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(pflag.CommandLine)

	// set up logger
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)

	// check input path
	if viper.GetString("input") == "" {
		logger.Log("msg", "input cannot be empty")
		os.Exit(1)
	}

	// parse provided url
	u, err := url.Parse(viper.GetString("webhook"))
	if err != nil {
		logger.Log("msg", "error with webhook url", "error", err)
		os.Exit(1)
	}

	// set up metrics
	var srv *http.Server
	reg := prometheus.NewRegistry()
	if listen := viper.GetString("metrics-listen"); listen != "" {
		r := http.NewServeMux()
		r.Handle(viper.GetString("metrics-path"), promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
		srv = &http.Server{
			Addr:         listen,
			Handler:      r,
			ReadTimeout:  time.Second * 2,
			WriteTimeout: time.Second * 2,
		}
	}

	// set up reloader
	r := reloader.New(logger, reg, &reloader.Options{
		ReloadURL:                     u,
		CfgFile:                       viper.GetString("input"),
		CfgOutputFile:                 viper.GetString("output"),
		WatchedDirs:                   viper.GetStringSlice("watch-dirs"),
		WatchInterval:                 3 * time.Minute,
		RetryInterval:                 5 * time.Second,
		DelayInterval:                 1 * time.Second,
		TolerateEnvVarExpansionErrors: viper.GetBool("ignore-missing"),
	})
	ctx, cancel := context.WithCancel(context.Background())

	var g run.Group
	{
		// add reloader
		g.Add(func() error {
			return r.Watch(ctx)
		}, func(err error) {
			cancel()
		})

		// add metrics server
		if srv != nil {
			g.Add(func() error {
				return srv.ListenAndServe()
			}, func(err error) {
				go func() {
					ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
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

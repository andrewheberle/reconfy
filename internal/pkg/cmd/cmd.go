package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bep/simplecobra"
	"github.com/go-kit/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/thanos-io/thanos/pkg/reloader"
)

type rootCommand struct {
	name string

	logger log.Logger

	viper               *viper.Viper
	viperEnvPrefix      string
	viperStringReplacer *strings.Replacer

	commands []simplecobra.Commander
}

func (c *rootCommand) Name() string {
	return c.name
}

func (c *rootCommand) Init(cd *simplecobra.Commandeer) error {
	cmd := cd.CobraCommand
	cmd.Short = "A simple reconfigurator"

	// command line args
	cmd.Flags().String("input", "", "Input file path to watch for changes")
	cmd.Flags().String("output", "", "Output path for environment variable substitutions")
	cmd.Flags().String("webhook", "http://localhost:8080", "Webhook URL")
	cmd.Flags().StringSlice("watchdirs", []string{}, "Additional directories to watch for changes")
	cmd.Flags().String("metrics.listen", "", "Listen address for metrics")
	cmd.Flags().String("metrics.path", "/metrics", "Path for metrics")
	cmd.Flags().Bool("ignoremissing", false, "Ignore missing environment variables when performing substitutions")
	cmd.Flags().String("config", "", "Configuration file to load (supports multiple reloaders)")

	// bind flags to viper
	c.viper.BindPFlags(cmd.Flags())

	// use env vars
	if c.viperEnvPrefix != "" {
		c.viper.SetEnvPrefix(c.viperEnvPrefix)
	}
	if c.viperStringReplacer != nil {
		c.viper.SetEnvKeyReplacer(c.viperStringReplacer)
	}
	c.viper.AutomaticEnv()

	// set any flags found in environment/config via viper
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if c.viper.IsSet(f.Name) && c.viper.GetString(f.Name) != "" {
			cmd.Flags().Set(f.Name, c.viper.GetString(f.Name))
		}
	})

	return nil
}

func (c *rootCommand) PreRun(this, runner *simplecobra.Commandeer) error {
	return nil
}

func (c *rootCommand) Run(ctx context.Context, cd *simplecobra.Commandeer, args []string) error {
	// parse config/flags and get list of reloaders
	reloaders, err := c.loadConfig(c.viper.GetString("config"))
	if err != nil {
		return fmt.Errorf("problem loading config: %w", err)
	}

	// validate config
	if err := validate(reloaders); err != nil {
		return fmt.Errorf("invalid config: %w", err)
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
			thisLogger := c.logger
			if rl.Name != "" {
				thisLogger = log.With(c.logger, "name", rl.Name)
			}

			// parse provided url
			u, err := url.Parse(rl.Webhook)
			if err != nil {
				return fmt.Errorf("error with webhook url for %s: %w", rl.Name, err)
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
				c.logger.Log("msg", "starting metrics HTTP server", "listen", viper.GetString("metrics.listen"), "path", viper.GetString("metrics.path"))
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

	return g.Run()
}

func (c *rootCommand) Commands() []simplecobra.Commander {
	return c.commands
}

func Execute(args []string) error {
	// set up logger
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestamp, "caller", log.DefaultCaller)

	// set up rootCmd
	rootCmd := &rootCommand{
		name:                "reconfy",
		logger:              logger,
		viper:               viper.New(),
		viperEnvPrefix:      "reconfy",
		viperStringReplacer: strings.NewReplacer("-", "_", ".", "_"),
	}
	x, err := simplecobra.New(rootCmd)
	if err != nil {
		return err
	}

	if _, err := x.Execute(context.Background(), args); err != nil {
		return err
	}

	return nil
}

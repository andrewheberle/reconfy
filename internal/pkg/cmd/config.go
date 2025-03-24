package cmd

import (
	"fmt"
	"path/filepath"
	"slices"
)

type ReloaderConfig struct {
	Name      string
	Input     string
	Output    string
	Webhook   string
	Watchdirs []string
}

type BaseConfig struct {
	Metrics       MetricsConfig
	IgnoreMissing bool
}

type SingleReloaderConfig struct {
	ReloaderConfig `mapstructure:",squash"`
	BaseConfig     `mapstructure:",squash"`
}

type MultipleReloaderConfig struct {
	BaseConfig `mapstructure:",squash"`
	Reloaders  []ReloaderConfig
}

type MetricsConfig struct {
	Path   string
	Listen string
}

func (c *rootCommand) loadConfig(config string) ([]ReloaderConfig, error) {
	if config == "" {
		// with no config just use flags
		return []ReloaderConfig{
			{
				Input:     c.viper.GetString("input"),
				Output:    c.viper.GetString("output"),
				Webhook:   c.viper.GetString("webhook"),
				Watchdirs: c.viper.GetStringSlice("watchdirs"),
			},
		}, nil
	}

	// load config
	c.viper.SetConfigFile(config)
	if err := c.viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// try to unmarshal as a multi reloader config
	var multi MultipleReloaderConfig
	if err := c.viper.Unmarshal(&multi); err == nil {
		if err := validate(multi.Reloaders); err != nil {
			c.logger.Log(err)
		}
		return multi.Reloaders, nil
	}

	// try as a single reloader config
	var single SingleReloaderConfig
	if err := c.viper.Unmarshal(&single); err != nil {
		return nil, err
	}

	return []ReloaderConfig{
		{
			Input:     single.Input,
			Name:      single.Name,
			Output:    single.Output,
			Webhook:   single.Webhook,
			Watchdirs: single.Watchdirs,
		},
	}, nil
}

// does basic validation of reloaders
func validate(reloaders []ReloaderConfig) error {
	names := make([]string, 0)
	for _, r := range reloaders {
		// check input isn't an output
		if r.Input == r.Output {
			return fmt.Errorf("circular dependency found as %s is an input and output file", r.Input)
		}

		// check watch dirs are not the same as an output
		d := filepath.Dir(r.Output)
		for _, w := range r.Watchdirs {
			if d == filepath.Clean(w) {
				return fmt.Errorf("circular dependency found as watched directory %s is used as an output location for %s", w, r.Output)
			}
		}

		// check name is not already used
		if slices.Contains(names, r.Name) {
			return fmt.Errorf("reloader name %s already in use", r.Name)
		}
		names = append(names, r.Name)
	}

	return nil
}

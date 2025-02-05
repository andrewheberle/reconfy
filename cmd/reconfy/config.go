package main

import "github.com/spf13/viper"

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

func LoadConfig(v *viper.Viper, config string) ([]ReloaderConfig, error) {
	if config == "" {
		// with no config just use flags
		return []ReloaderConfig{
			ReloaderConfig{
				Input:     v.GetString("input"),
				Output:    v.GetString("output"),
				Webhook:   v.GetString("webhook"),
				Watchdirs: v.GetStringSlice("watchdirs"),
			},
		}, nil
	}

	// load config
	v.SetConfigFile(config)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// try to unmarshal as a multi reloader config
	var multi MultipleReloaderConfig
	if err := v.Unmarshal(&multi); err == nil {
		return multi.Reloaders, nil
	}

	// try as a single reloader config
	var single SingleReloaderConfig
	if err := v.Unmarshal(&single); err != nil {
		return nil, err
	}

	return []ReloaderConfig{
		ReloaderConfig{
			Input:     single.Input,
			Name:      single.Name,
			Output:    single.Output,
			Webhook:   single.Webhook,
			Watchdirs: single.Watchdirs,
		},
	}, nil
}

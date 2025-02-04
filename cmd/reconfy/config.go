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

func LoadConfig(config string) ([]ReloaderConfig, error) {
	if config == "" {
		// with no config just use flags
		return []ReloaderConfig{
			ReloaderConfig{
				Input:     viper.GetString("input"),
				Output:    viper.GetString("output"),
				Webhook:   viper.GetString("webhook"),
				Watchdirs: viper.GetStringSlice("watchdirs"),
			},
		}, nil
	}

	// load config
	viper.SetConfigFile(config)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// try to unmarshal as a multi reloader config multi
	var multi MultipleReloaderConfig
	if err := viper.Unmarshal(&multi); err != nil {
		return nil, err
	}

	return multi.Reloaders, nil
}

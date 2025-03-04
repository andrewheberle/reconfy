package cmd

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
			ReloaderConfig{
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
		return multi.Reloaders, nil
	}

	// try as a single reloader config
	var single SingleReloaderConfig
	if err := c.viper.Unmarshal(&single); err != nil {
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

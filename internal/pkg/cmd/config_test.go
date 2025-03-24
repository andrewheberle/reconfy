package cmd

import "testing"

func Test_validate(t *testing.T) {
	tests := []struct {
		name      string
		reloaders []ReloaderConfig
		wantErr   bool
	}{
		// single configs
		{"valid single config", []ReloaderConfig{
			{
				Name:   "single",
				Input:  "/input/config.in",
				Output: "/output/config.conf",
			},
		}, false},
		{"valid single config (with watchdir)", []ReloaderConfig{
			{
				Name:      "single",
				Input:     "/input/config.in",
				Output:    "/output/config.conf",
				Watchdirs: []string{"/another"},
			},
		}, false},
		{"invalid single config", []ReloaderConfig{
			{
				Name:   "single",
				Input:  "/input/config.in",
				Output: "/input/config.in",
			},
		}, true},
		{"invalid single config (with watchdir)", []ReloaderConfig{
			{
				Name:      "single",
				Input:     "/input/config.in",
				Output:    "/output/config.conf",
				Watchdirs: []string{"/output"},
			},
		}, true},

		// multiple configs
		{"valid multiple config", []ReloaderConfig{
			{
				Name:   "multiple1",
				Input:  "/input/config1.in",
				Output: "/output/config1.conf",
			},
			{
				Name:   "multiple2",
				Input:  "/input/config2.in",
				Output: "/output/config2.conf",
			},
		}, false},
		{"valid multiple config (with watchdir)", []ReloaderConfig{
			{
				Name:      "multiple1",
				Input:     "/input/config1.in",
				Output:    "/output/config1.conf",
				Watchdirs: []string{"/another"},
			},
			{
				Name:      "multiple2",
				Input:     "/input/config2.in",
				Output:    "/output/config2.conf",
				Watchdirs: []string{"/another"},
			},
		}, false},
		{"invalid multiple config", []ReloaderConfig{
			{
				Name:   "multiple1",
				Input:  "/input/config1.in",
				Output: "/output/config1.conf",
			},
			{
				Name:   "multiple2",
				Input:  "/input/config2.in",
				Output: "/input/config2.in",
			},
		}, true},
		{"invalid multiple config (duplicate name)", []ReloaderConfig{
			{
				Name:      "multiple",
				Input:     "/input/config1.in",
				Output:    "/output/config1.conf",
				Watchdirs: []string{"/another"},
			},
			{
				Name:      "multiple",
				Input:     "/input/config2.in",
				Output:    "/output/config2.conf",
				Watchdirs: []string{"/another"},
			},
		}, true},
		{"invalid multiple config (with watchdir)", []ReloaderConfig{
			{
				Name:      "multiple1",
				Input:     "/input/config1.in",
				Output:    "/output/config1.conf",
				Watchdirs: []string{"/output"},
			},
			{
				Name:      "multiple2",
				Input:     "/input/config2.in",
				Output:    "/output/config2.conf",
				Watchdirs: []string{"/another"},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate(tt.reloaders); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

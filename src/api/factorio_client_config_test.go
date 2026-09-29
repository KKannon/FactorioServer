package api

import "testing"

func TestValidateFactorioClientConfigAcceptsSafeSettings(t *testing.T) {
	config := defaultFactorioClientConfig()
	config.PublicGameHost = "factorio.example.com"
	config.GamePort = 34197
	if err := validateFactorioClientConfig(config); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
}

func TestValidateFactorioClientConfigRejectsUnsafeSettings(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*factorioClientConfig)
	}{
		{"host with path", func(config *factorioClientConfig) { config.PublicGameHost = "example.com/path" }},
		{"invalid port", func(config *factorioClientConfig) { config.GamePort = 70000 }},
		{"non HTTPS link", func(config *factorioClientConfig) { config.OfficialDownloadURL = "http://example.com/game" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := defaultFactorioClientConfig()
			test.mutate(&config)
			if err := validateFactorioClientConfig(config); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

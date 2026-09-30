package api

import "testing"

func TestValidateFactorioClientConfigAcceptsSafeSettings(t *testing.T) {
	config := defaultFactorioClientConfig()
	config.PublicGameHost = "factorio.example.com"
	config.GamePort = 34197
	config.IncludeGamePort = false
	config.GamePort = 0
	config.AlternativeDownloadMagnet = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Factorio"
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
		{"missing enabled port", func(config *factorioClientConfig) { config.GamePort = 0; config.IncludeGamePort = true }},
		{"non HTTPS link", func(config *factorioClientConfig) { config.OfficialDownloadURL = "http://example.com/game" }},
		{"non magnet alternative link", func(config *factorioClientConfig) {
			config.AlternativeDownloadMagnet = "https://example.com/factorio.torrent"
		}},
		{"magnet without exact topic", func(config *factorioClientConfig) { config.AlternativeDownloadMagnet = "magnet:?dn=Factorio" }},
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

func TestEffectiveFactorioClientConfigHidesMagnetForAuthenticatedServers(t *testing.T) {
	config := defaultFactorioClientConfig()
	config.AlternativeDownloadMagnet = "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"
	if visible := effectiveFactorioClientConfig(config, true); visible.AlternativeDownloadMagnet == "" {
		t.Fatal("magnet must be exposed when unauthenticated players are allowed")
	}
	if hidden := effectiveFactorioClientConfig(config, false); hidden.AlternativeDownloadMagnet != "" {
		t.Fatal("magnet must be hidden when Factorio.com verification is required")
	}
}

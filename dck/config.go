package megatwist

import (
	"encoding/json"
	"os"
)

// Config contains the optional desktop configuration. Android uses these
// defaults because an APK has no repository-relative config.json.
type Config struct {
	Fullscreen     bool    `json:"fullscreen"`
	VSync          bool    `json:"vsync"`
	MusicVolume    float64 `json:"musicVolume"`
	SpriteCount    int     `json:"spriteCount"`
	DistortionRate float64 `json:"distortionRate"`
	EnableCRT      bool    `json:"enableCRT"`
	EnableGlow     bool    `json:"enableGlow"`
}

func defaultConfig() Config {
	return Config{
		VSync:          true,
		MusicVolume:    0.7,
		SpriteCount:    10,
		DistortionRate: 1,
		EnableCRT:      true,
		EnableGlow:     true,
	}
}

func loadConfig() *Config {
	data, err := os.ReadFile("config.json")
	if err != nil {
		cfg := defaultConfig()
		return &cfg
	}
	cfg := decodeConfig(data)
	return &cfg
}

func decodeConfig(data []byte) Config {
	defaults := defaultConfig()
	cfg := defaults
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaults
	}
	if cfg.SpriteCount <= 0 {
		cfg.SpriteCount = defaults.SpriteCount
	}
	if cfg.DistortionRate <= 0 {
		cfg.DistortionRate = defaults.DistortionRate
	}
	if cfg.MusicVolume < 0 || cfg.MusicVolume > 1 {
		cfg.MusicVolume = defaults.MusicVolume
	}
	return cfg
}

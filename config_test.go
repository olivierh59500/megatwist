package megatwist

import "testing"

func TestDecodeConfigKeepsDefaultsForMissingFields(t *testing.T) {
	cfg := decodeConfig([]byte(`{"enableCRT":false,"spriteCount":3}`))
	if cfg.EnableCRT {
		t.Fatal("EnableCRT = true, want false")
	}
	if cfg.SpriteCount != 3 {
		t.Fatalf("SpriteCount = %d, want 3", cfg.SpriteCount)
	}
	if cfg.MusicVolume != 0.7 || !cfg.VSync || !cfg.EnableGlow {
		t.Fatalf("missing fields did not retain defaults: %+v", cfg)
	}
}

func TestDecodeConfigRejectsInvalidJSONWithoutPartialMutation(t *testing.T) {
	cfg := decodeConfig([]byte(`{"spriteCount":99,`))
	want := defaultConfig()
	if cfg != want {
		t.Fatalf("decodeConfig(invalid) = %+v, want %+v", cfg, want)
	}
}

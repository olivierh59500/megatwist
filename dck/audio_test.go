package megatwist

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/olivierh59500/democonstructionkit/sound"
)

func TestMusicStreamReadProducesStereoWithoutAllocating(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := player.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	pcm := make([]byte, 4096*4)
	read := func() {
		n, err := player.Read(pcm)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if n != len(pcm) {
			t.Fatalf("Read bytes = %d, want %d", n, len(pcm))
		}
	}

	read()
	nonSilent := false
	for offset := 0; offset < len(pcm); offset += 4 {
		left := int16(binary.LittleEndian.Uint16(pcm[offset : offset+2]))
		right := int16(binary.LittleEndian.Uint16(pcm[offset+2 : offset+4]))
		if left != right {
			t.Fatalf("frame %d differs between channels: %d != %d", offset/4, left, right)
		}
		if left != 0 {
			nonSilent = true
		}
	}
	if !nonSilent {
		t.Fatal("generated PCM block is silent")
	}

	if allocations := testing.AllocsPerRun(100, read); allocations != 0 {
		t.Fatalf("Read allocations = %.2f, want 0", allocations)
	}
}

func TestMusicStreamReadAfterCloseReturnsClosedPipe(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}

	pcm := make([]byte, 64)
	for i := range pcm {
		pcm[i] = 0xff
	}
	n, err := player.Read(pcm)
	if n != 0 {
		t.Fatalf("closed Read bytes = %d, want 0", n)
	}
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("Read error = %v, want io.ErrClosedPipe", err)
	}
	for i, value := range pcm {
		if value != 0xff {
			t.Fatalf("closed stream changed PCM byte %d to %d", i, value)
		}
	}
}

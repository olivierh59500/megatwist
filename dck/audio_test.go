package megatwist

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestYMPlayerReadProducesStereoWithoutAllocating(t *testing.T) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
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

func TestYMPlayerReadAfterCloseReturnsSilence(t *testing.T) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
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
	if n != len(pcm) {
		t.Fatalf("Read bytes = %d, want %d", n, len(pcm))
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Read error = %v, want io.EOF", err)
	}
	for i, value := range pcm {
		if value != 0 {
			t.Fatalf("PCM byte %d = %d, want silence", i, value)
		}
	}
}

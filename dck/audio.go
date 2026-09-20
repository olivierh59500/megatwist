package megatwist

import originalassets "megatwist"

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const sampleRate = 48000

var musicData = originalassets.

	// YMPlayer adapts the mono YM synthesizer to Ebitengine's little-endian,
	// 16-bit stereo PCM stream. Its hot Read path performs no allocation.
	DCKAssetMusicData()

type YMPlayer struct {
	player *stsound.StSound
	buffer []int16
	mutex  sync.Mutex
	loop   bool
}

func NewYMPlayer(data []byte, rate int, loop bool) (*YMPlayer, error) {
	player := stsound.CreateWithRate(rate)
	if err := player.LoadMemory(data); err != nil {
		player.Destroy()
		return nil, fmt.Errorf("load YM data: %w", err)
	}
	player.SetLoopMode(loop)

	return &YMPlayer{
		player: player,
		buffer: make([]int16, 4096),
		loop:   loop,
	}, nil
}

func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	samplesNeeded := len(p) / 4
	byteCount := samplesNeeded * 4
	if y.player == nil {
		clear(p[:byteCount])
		return byteCount, io.EOF
	}

	processed := 0
	for processed < samplesNeeded {
		chunkSize := min(samplesNeeded-processed, len(y.buffer))
		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) && !y.loop {
			clear(p[processed*4 : byteCount])
			err = io.EOF
			break
		}

		for i, mono := range y.buffer[:chunkSize] {
			// The YM output is intentionally attenuated to 50% before the
			// configurable Ebitengine player volume is applied.
			sample := mono / 2
			offset := (processed + i) * 4
			p[offset] = byte(sample)
			p[offset+1] = byte(sample >> 8)
			p[offset+2] = byte(sample)
			p[offset+3] = byte(sample >> 8)
		}
		processed += chunkSize
	}
	return byteCount, err
}

func (y *YMPlayer) Close() error {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player != nil {
		y.player.Destroy()
		y.player = nil
	}
	return nil
}

func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	ym, err := NewYMPlayer(musicData, sampleRate, true)
	if err != nil {
		log.Printf("could not create YM player: %v", err)
		return
	}
	g.ymPlayer = ym

	player, err := g.audioContext.NewPlayer(ym)
	if err != nil {
		log.Printf("could not create audio player: %v", err)
		if closeErr := ym.Close(); closeErr != nil {
			log.Printf("could not close YM player: %v", closeErr)
		}
		g.ymPlayer = nil
		return
	}

	g.audioPlayer = player
	g.audioPlayer.SetVolume(g.config.MusicVolume)
	g.audioPlayer.Play()
}

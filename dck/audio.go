package megatwist

import (
	"log"
	originalassets "megatwist"

	"github.com/olivierh59500/democonstructionkit/sound"

	audio "github.com/olivierh59500/democonstructionkit/sound/output"
)

const sampleRate = 48000

var musicData = originalassets.DCKAssetMusicData()

func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	music, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.5})
	if err != nil {
		log.Printf("could not open music: %v", err)
		return
	}
	g.musicStream = music

	player, err := g.audioContext.NewPlayer(music)
	if err != nil {
		log.Printf("could not create audio player: %v", err)
		if closeErr := music.Close(); closeErr != nil {
			log.Printf("could not close music: %v", closeErr)
		}
		g.musicStream = nil
		return
	}

	g.audioPlayer = player
	g.audioPlayer.SetVolume(g.config.MusicVolume)
	g.audioPlayer.Play()
}

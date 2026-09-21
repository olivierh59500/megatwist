// Command video exports the complete game canvas and its own audio.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
	demo "megatwist/dck"
)

func main() {
	config := video.Config{Output: "megatwist.mp4", PosterAt: 60 * time.Second, Title: "Megatwist", Width: 832, Height: 552, FPS: 60, TPS: 60, SampleRate: 48000, Duration: 3 * time.Minute}
	config.Flags(flag.CommandLine)
	flag.Parse()
	if err := video.Run(config, func() (ebiten.Game, error) {
		return demo.NewGame(), nil
	}); err != nil {
		log.Fatal(err)
	}
}

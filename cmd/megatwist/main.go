package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"megatwist"
)

func main() {
	ebiten.SetWindowSize(megatwist.ContentWidth, megatwist.ContentHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle(megatwist.WindowTitle)
	ebiten.SetScreenClearedEveryFrame(false)
	game := megatwist.NewGame()
	ebiten.SetFullscreen(game.InitialFullscreen())
	if err := ebiten.RunGame(newDrawOnUpdateGame(game)); err != nil {
		log.Fatal(err)
	}
}

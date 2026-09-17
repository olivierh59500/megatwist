package main

import "github.com/hajimehoshi/ebiten/v2"

// drawOnUpdateGame avoids rebuilding an identical frame on displays whose
// refresh rate is higher than Ebitengine's 60 Hz update rate.
type drawOnUpdateGame struct {
	game  ebiten.Game
	dirty bool
}

func newDrawOnUpdateGame(game ebiten.Game) *drawOnUpdateGame {
	return &drawOnUpdateGame{game: game, dirty: true}
}

func (g *drawOnUpdateGame) Update() error {
	if err := g.game.Update(); err != nil {
		return err
	}
	g.dirty = true
	return nil
}

func (g *drawOnUpdateGame) Draw(screen *ebiten.Image) {
	if !g.dirty {
		return
	}
	g.game.Draw(screen)
	g.dirty = false
}

func (g *drawOnUpdateGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.game.Layout(outsideWidth, outsideHeight)
}

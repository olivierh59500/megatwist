package megatwist

import (
	"github.com/hajimehoshi/ebiten/v2"
	kit "github.com/olivierh59500/democonstructionkit"
	"image/color"
)

func (g *Game) animIntro() error {
	if err := g.introScroll.Update(kit.Frame{}); err != nil {
		return err
	}
	if g.introScroll.Finished() {
		g.state = stateSplash
		g.iteration = 0
		g.splash.Reset()
		return nil
	}
	g.surfMain.Fill(color.Black)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 170)
	g.surfMain.DrawImage(g.introScroll.Image(), op)
	return nil
}

func (g *Game) animSplash() {
	if g.splash.Step() {
		g.state = stateDemo
		g.iteration = 0
		g.splash.Reset()
	}
	g.surfMain.Fill(color.Black)
}

func (g *Game) animDemo() error {
	if err := g.mainScroll.Update(kit.Frame{Tick: uint64(g.iteration)}); err != nil {
		return err
	}
	if err := g.background.Update(kit.Frame{Tick: uint64(g.iteration), Time: float64(g.iteration) / 60, Delta: 1.0 / 60}); err != nil {
		return err
	}
	g.surfMain.Clear()
	g.background.Draw(g.surfMain)
	g.mainScroll.Draw(g.surfMain)
	g.iteration++
	return g.spriteGroup.Update(kit.Frame{})
}

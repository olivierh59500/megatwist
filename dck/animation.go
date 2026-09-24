package megatwist

import (
	"github.com/hajimehoshi/ebiten/v2"
	kit "github.com/olivierh59500/democonstructionkit"
	"image/color"
	"math"
)

func (g *Game) updateSprites() {
	const (
		centerX = float64(screenWidth) / 2
		centerY = float64(screenHeight) / 2
		half    = float64(spriteSize) / 2
	)
	for i := range g.sprites {
		phase := g.ctrSprite + float64(i)*0.155
		x := centerX + 100*math.Sin(phase*1.35+1.25) + 100*math.Sin(phase*1.86+0.54)
		y := centerY + 60*math.Cos(phase*1.72+0.23) + 60*math.Cos(phase*1.63+0.98)
		x += 20 * math.Sin(float64(i)*0.289+1.15)
		y += 20 * math.Cos(float64(i)*0.456+0.85)
		g.sprites[i].x = min(max(x, half), float64(screenWidth)-half)
		g.sprites[i].y = min(max(y, half), float64(screenHeight)-half)
	}
}

func (g *Game) animIntro() error {
	if err := g.introScroll.Update(kit.Frame{}); err != nil {
		return err
	}
	if g.introScroll.Finished() {
		g.lastState = g.state
		g.state = stateSplash
		g.iteration = 0
		g.transitionProgress = 0
		return nil
	}
	g.surfMain.Fill(color.Black)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 170)
	g.surfMain.DrawImage(g.introScroll.Image(), op)
	return nil
}

func (g *Game) animSplash() {
	if g.iteration < 90 {
		g.iteration++
		g.transitionProgress = float64(g.iteration) / 90
	} else {
		g.lastState = g.state
		g.state = stateDemo
		g.iteration = 0
		g.transitionProgress = 0
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
	g.ctrSprite += .02
	g.updateSprites()
	return nil
}

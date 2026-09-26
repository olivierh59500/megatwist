package megatwist

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (g *Game) Draw(screen *ebiten.Image) {
	g.frame.Clear()
	if g.config.EnableCRT && g.crt != nil && g.state == stateIntro {
		g.scaledMain.Clear()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		g.scaledMain.DrawImage(g.surfMain, op)

		g.crt.DrawAt(g.frame, g.scaledMain, 0, 0)
	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		g.frame.DrawImage(g.surfMain, op)
	}

	if g.state == stateDemo {
		g.spriteGlow.DrawGroup(g.frame, g.spriteGroup)
	}
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		ebitenutil.DebugPrint(g.frame, g.DebugSummary())
	}

	screen.Fill(color.Black)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((screen.Bounds().Dx()-ContentWidth)/2), 0)
	screen.DrawImage(g.frame, op)
}

package megatwist

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (g *Game) drawTransition(target *ebiten.Image) {
	progress := g.transitionProgress
	if progress <= 0 || progress >= 1 {
		return
	}

	g.overlay.Fill(color.RGBA{0, 0, 0, uint8(255 * (1 - progress))})
	if g.lastState == stateSplash && g.state == stateDemo {
		op := &ebiten.DrawImageOptions{}
		scale := 1 + (1-progress)*0.2
		op.GeoM.Translate(-float64(ContentWidth)/2, -float64(ContentHeight)/2)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(float64(ContentWidth)/2, float64(ContentHeight)/2)
		target.DrawImage(g.overlay, op)
		return
	}
	target.DrawImage(g.overlay, nil)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.frame.Clear()
	if g.config.EnableCRT && g.crtShader != nil && g.state == stateIntro {
		g.scaledMain.Clear()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		g.scaledMain.DrawImage(g.surfMain, op)

		shaderOptions := &ebiten.DrawRectShaderOptions{}
		shaderOptions.Images[0] = g.scaledMain
		shaderOptions.Blend = ebiten.BlendCopy
		g.frame.DrawRectShader(ContentWidth, ContentHeight, g.crtShader, shaderOptions)
	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		g.frame.DrawImage(g.surfMain, op)
	}

	if g.state == stateDemo {
		g.spriteGlow.DrawGroup(g.frame, g.spriteGroup)
	}
	g.drawTransition(g.frame)

	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		ebitenutil.DebugPrint(g.frame, g.DebugSummary())
	}

	screen.Fill(color.Black)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((screen.Bounds().Dx()-ContentWidth)/2), 0)
	screen.DrawImage(g.frame, op)
}

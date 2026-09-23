package megatwist

import (
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func (g *Game) displayText(letterOffset int) {
	if letterOffset == g.displayedLetter {
		return
	}
	g.displayedLetter = letterOffset
	g.surfScroll.Clear()
	if g.scrollRenderer == nil {
		glyphs := make([]scrolling.Glyph, len(g.text))
		for i, r := range g.text {
			glyphImage, letter, ok := g.fontAtlas.ExactGlyph(r)
			if !ok {
				glyphImage, letter, _ = g.fontAtlas.ExactGlyph(' ')
			}
			glyphs[i] = scrolling.Glyph{Image: glyphImage, Advance: float64(int(letter.Advance))}
		}
		var err error
		g.scrollRenderer, err = scrolling.New(scrolling.Config{Glyphs: glyphs})
		if err != nil {
			panic(err)
		}
	}
	state := g.scrollRenderer.Window(letterOffset, float64(g.surfScroll.Bounds().Dx()))
	g.scrollRenderer.DrawAt(g.surfScroll, state)
}

func appendScanline(
	vertices []ebiten.Vertex,
	indices []uint16,
	destinationY, sourceX, sourceY int,
) ([]ebiten.Vertex, []uint16) {
	base := uint16(len(vertices))
	dstTop := float32(destinationY)
	dstBottom := dstTop + 1
	srcLeft := float32(sourceX)
	srcRight := srcLeft + screenWidth
	srcTop := float32(sourceY)
	srcBottom := srcTop + 1

	vertices = append(vertices,
		ebiten.Vertex{DstX: 0, DstY: dstTop, SrcX: srcLeft, SrcY: srcTop, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: screenWidth, DstY: dstTop, SrcX: srcRight, SrcY: srcTop, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: 0, DstY: dstBottom, SrcX: srcLeft, SrcY: srcBottom, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		ebiten.Vertex{DstX: screenWidth, DstY: dstBottom, SrcX: srcRight, SrcY: srcBottom, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	)
	indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
	return vertices, indices
}

func positiveMod(value, modulus int) int {
	result := value % modulus
	if result < 0 {
		return result + modulus
	}
	return result
}

func (g *Game) renderDistortion(bounceBack, bounceFront int) {
	if g.backgroundBatch == nil {
		g.backgroundBatch = composite.NewQuadBatch(screenHeight)
		g.frontBatch = composite.NewQuadBatch(screenHeight)
		g.backgroundBatch.AlternateDiagonal = true
		g.frontBatch.AlternateDiagonal = true
	}
	g.surfMain.Clear()
	g.backProgram.Fill(g.backRows[:], g.backWavePos)
	g.backgroundBatch.Begin(g.surfMain, g.surfBack)
	for line := 0; line < screenHeight; line++ {
		wave := g.backRows[line]
		x := positiveMod(80+wave/2, g.backImg.Bounds().Dx())
		y := (line + bounceBack) % backHeight
		g.backgroundBatch.Rect(image.Rect(x, y, x+screenWidth, y+1), 0, float32(line), screenWidth, 1)
	}
	g.backgroundBatch.Flush()
	g.frontBatch.Begin(g.surfMain, g.surfScroll)
	maxX := g.surfScroll.Bounds().Dx() - screenWidth
	for line := 0; line < screenHeight; line++ {
		wave := g.frontRows[line]
		x := wave - g.letterDecal
		if x >= 0 && x < maxX {
			y := (line + bounceFront) % fontHeight
			g.frontBatch.Rect(image.Rect(x, y, x+screenWidth, y+1), 0, float32(line), screenWidth, 1)
		}
	}
	g.frontBatch.Flush()
}

func (g *Game) drawGlowSprite(target *ebiten.Image, sprite *Sprite) {
	if g.config.EnableGlow {
		for layer := 3; layer > 0; layer-- {
			op := &ebiten.DrawImageOptions{}
			scale := zoom + float64(layer)*0.1
			op.GeoM.Translate(-float64(spriteSize)/2, -float64(spriteSize)/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(sprite.x*zoom, sprite.y*zoom)
			op.ColorScale.ScaleAlpha(float32(0.3 / float64(layer)))
			op.Filter = ebiten.FilterLinear
			composite.Instance{Image: g.logoImg, Options: *op}.Draw(target)
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(spriteSize)/2, -float64(spriteSize)/2)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(sprite.x*zoom, sprite.y*zoom)
	composite.Instance{Image: g.logoImg, Options: *op}.Draw(target)
}

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
		for i := range g.sprites {
			g.drawGlowSprite(g.frame, &g.sprites[i])
		}
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

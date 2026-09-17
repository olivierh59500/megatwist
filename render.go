package megatwist

import (
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
	x := 0
	for index := 0; x < g.surfScroll.Bounds().Dx(); index++ {
		letter, ok := g.letterData[getLetter(g.text, index+letterOffset)]
		if !ok {
			letter = g.letterData[' ']
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), 0)
		g.surfScroll.DrawImage(letter.glyph, op)
		x += letter.width
	}
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
	g.backgroundVertices = g.backgroundVertices[:0]
	g.backgroundIndices = g.backgroundIndices[:0]
	g.scrollVertices = g.scrollVertices[:0]
	g.scrollIndices = g.scrollIndices[:0]

	backgroundPeriod := g.backImg.Bounds().Dx()
	maxScrollX := g.surfScroll.Bounds().Dx() - screenWidth
	for line := 0; line < screenHeight; line++ {
		backWave := getWave(g.backWavePos+line, g.backIntroWave, g.backMainWave)
		backX := positiveMod(80+backWave/2, backgroundPeriod)
		backY := (line + bounceBack) % backHeight
		g.backgroundVertices, g.backgroundIndices = appendScanline(
			g.backgroundVertices,
			g.backgroundIndices,
			line,
			backX,
			backY,
		)

		frontWave := getWave(g.frontWavePos+line, g.frontIntroWave, g.frontMainWave)
		scrollX := frontWave - g.letterDecal
		if scrollX >= 0 && scrollX < maxScrollX {
			scrollY := (line + bounceFront) % fontHeight
			g.scrollVertices, g.scrollIndices = appendScanline(
				g.scrollVertices,
				g.scrollIndices,
				line,
				scrollX,
				scrollY,
			)
		}
	}

	g.surfMain.Clear()
	g.surfMain.DrawTriangles(g.backgroundVertices, g.backgroundIndices, g.surfBack, nil)
	g.surfMain.DrawTriangles(g.scrollVertices, g.scrollIndices, g.surfScroll, nil)
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
			target.DrawImage(g.logoImg, op)
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(spriteSize)/2, -float64(spriteSize)/2)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(sprite.x*zoom, sprite.y*zoom)
	target.DrawImage(g.logoImg, op)
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

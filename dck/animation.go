package megatwist

import (
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/presets"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

func createCurves(distortionRate float64) [][]int {
	curves, err := presets.RibbonCurves(distortionRate)
	if err != nil {
		panic(err)
	}
	return curves
}

func precalcWave(curves [][]int, waveTypes []int) []int {
	values, err := composite.JoinDeltaCurves(curves, waveTypes)
	if err != nil {
		panic(err)
	}
	return values
}

func (g *Game) precalcPosition() {
	g.position = make([]int, 0, len(g.text))
	position := 0
	for _, char := range g.text {
		if _, letter, ok := g.fontAtlas.ExactGlyph(char); ok {
			position += int(letter.Advance)
			g.position = append(g.position, position)
		}
	}
}

func getSum(values []int, index, decal int) int {
	return composite.CumulativeAt(values, index, decal)
}

func (g *Game) getPosition(index int) int {
	if index > 0 && index <= len(g.position) {
		return getSum(g.position, index-1, 0)
	}
	return 0
}

func getLetter(text []rune, position int) rune {
	if len(text) == 0 {
		return ' '
	}
	return text[position%len(text)]
}

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

func (g *Game) animIntro() {
	if g.introX < 0 {
		if g.introTile >= 0 {
			if _, letter, ok := g.fontAtlas.ExactGlyph(getLetter(g.introText, g.introTile)); ok {
				g.introX += int(letter.Advance)
			}
		}
		g.introLetter++
		if g.introLetter >= len(g.introText) {
			g.lastState = g.state
			g.state = stateSplash
			g.iteration = 0
			g.transitionProgress = 0
			return
		}
		g.introTile = g.introLetter
	}
	g.introX -= g.introSpeed

	// Shift into the spare surface, then swap the two pointers. This replaces
	// both a transient SubImage and an unnecessary full-surface copy.
	g.surfScroll2.Clear()
	shift := &ebiten.DrawImageOptions{}
	shift.GeoM.Translate(float64(-g.introSpeed), 0)
	g.surfScroll2.DrawImage(g.surfScroll1, shift)
	g.surfScroll1, g.surfScroll2 = g.surfScroll2, g.surfScroll1

	if glyphImage, _, ok := g.fontAtlas.ExactGlyph(getLetter(g.introText, g.introTile)); ok {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth+g.introX), 0)
		g.surfScroll1.DrawImage(glyphImage, op)
	}

	g.surfMain.Fill(color.Black)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 170)
	g.surfMain.DrawImage(g.surfScroll1, op)
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

func (g *Game) animDemo() {
	g.calculateAndRenderDemo()
	g.iteration++
	g.backWavePos = g.iteration * 5
	g.frontWavePos = g.iteration * 10
	g.ctrSprite += 0.02
	g.updateSprites()
}

func (g *Game) calculateAndRenderDemo() {
	bounceBack := int(30 * math.Abs(math.Sin(float64(g.iteration)*0.1)))
	bounceFront := int(18 * math.Abs(math.Sin(float64(g.iteration)*0.1)))

	g.frontProgram.Fill(g.frontRows[:], g.frontWavePos)
	decalX := math.MaxInt
	for line := 0; line < screenHeight; line++ {
		decalX = min(decalX, g.frontRows[line])
	}
	decalX = max(decalX, 0)

	direction := 0
	if decalX > g.letterDecal {
		direction = 1
	} else if decalX < g.letterDecal {
		direction = -1
	}
	if direction != 0 {
		offset := 0
		for decalX < g.getPosition(g.letterNum+offset) || g.getPosition(g.letterNum+offset+1) <= decalX {
			offset += direction
			if g.letterNum+offset < 0 || g.letterNum+offset >= len(g.position) {
				break
			}
		}
		g.letterNum += offset
	}
	g.letterNum = min(max(g.letterNum, 0), len(g.position)-1)
	g.letterDecal = g.getPosition(g.letterNum)

	g.displayText(g.letterNum)
	g.renderDistortion(bounceBack, bounceFront)
}

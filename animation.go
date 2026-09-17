package megatwist

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

func createCurves(distortionRate float64) [][]int {
	curves := make([][]int, bgSin3+1)
	for curveType := cdZero; curveType <= bgSin3; curveType++ {
		var step, progress float64
		switch curveType {
		case cdZero:
			step = 2.25
		case cdSlowSin:
			step, progress = 0.20, 140
		case cdMedSin:
			step, progress = 0.25, 175
		case cdFastSin:
			step, progress = 0.30, 210
		case cdSlowDist:
			step, progress = 0.12, 175
		case cdMedDist:
			step, progress = 0.16, 210
		case cdFastDist:
			step, progress = 0.20, 245
		case cdSplitted:
			step = 0.18
		case bgSin1:
			step = 0.50
		case bgSin2:
			step = 0.80
		case bgSin3:
			step = 0.50
		}
		step *= distortionRate

		maxAngle := 360.0
		if curveType == cdSplitted {
			maxAngle = 720
		}
		values := make([]float64, 0, int(maxAngle/step)+1)
		for angle := 0.0; angle < maxAngle-step; angle += step {
			radians := angle * math.Pi / 180
			var value float64
			switch curveType {
			case cdZero:
			case cdSlowSin:
				value = 100 * math.Sin(radians)
			case cdMedSin:
				value = 110 * math.Sin(radians)
			case cdFastSin:
				value = 120 * math.Sin(radians)
			case cdSlowDist:
				value = 100*math.Sin(radians) + 25*math.Sin(radians*10)
			case cdMedDist:
				value = 110*math.Sin(radians) + 27.5*math.Sin(radians*9)
			case cdFastDist:
				value = 120*math.Sin(radians) + 30*math.Sin(radians*8)
			case cdSplitted:
				direction := 1.0
				if len(values)%2 == 1 {
					direction = -1
				}
				amplitude := 12.0
				if angle < 160 {
					amplitude *= angle / 160
				} else if angle > 560 {
					amplitude *= (720 - angle) / 160
				}
				value = 90*math.Sin(radians) + direction*amplitude*math.Sin(radians*3)
			case bgSin1, bgSin2:
				value = -60 * math.Sin(radians)
			case bgSin3:
				value = -60*math.Sin(radians) - 15*math.Sin(radians*4)
			}
			values = append(values, value)
		}

		curve := make([]int, len(values))
		decal := 0.0
		previous := 0
		for i, value := range values {
			item := -int(math.Floor(value - decal))
			curve[i] = item - previous
			previous = item
			decal += progress / float64(len(values))
		}
		curves[curveType] = curve
	}
	return curves
}

func precalcWave(curves [][]int, waveTypes []int) []int {
	capacity := 0
	for _, waveType := range waveTypes {
		capacity += len(curves[waveType])
	}
	waveValues := make([]int, 0, capacity)
	value := 0
	for _, waveType := range waveTypes {
		for _, delta := range curves[waveType] {
			value += delta
			waveValues = append(waveValues, value)
		}
	}
	return waveValues
}

func (g *Game) precalcPosition() {
	g.position = make([]int, 0, len(g.text))
	position := 0
	for _, char := range g.text {
		if letter, ok := g.letterData[char]; ok {
			position += letter.width
			g.position = append(g.position, position)
		}
	}
}

func getSum(values []int, index, decal int) int {
	if len(values) == 0 {
		return decal
	}
	cycles, offset := index/len(values), index%len(values)
	return decal + cycles*values[len(values)-1] + values[offset]
}

func getWave(index int, introWave, mainWave []int) int {
	if index < len(introWave) {
		return getSum(introWave, index, 0)
	}
	return getSum(mainWave, index-len(introWave), introWave[len(introWave)-1])
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
			if letter, ok := g.letterData[getLetter(g.introText, g.introTile)]; ok {
				g.introX += letter.width
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

	if letter, ok := g.letterData[getLetter(g.introText, g.introTile)]; ok {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth+g.introX), 0)
		g.surfScroll1.DrawImage(letter.glyph, op)
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

	decalX := math.MaxInt
	for line := 0; line < screenHeight; line++ {
		decalX = min(decalX, getWave(g.frontWavePos+line, g.frontIntroWave, g.frontMainWave))
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

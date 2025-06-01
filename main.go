// main.go
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth  = 416
	screenHeight = 276
	zoom         = 2
	backHeight   = 64
	fontHeight   = 36
	spriteSize   = 32
)

// Wave types
const (
	cdZero = iota
	cdSlowSin
	cdMedSin
	cdFastSin
	cdSlowDist
	cdMedDist
	cdFastDist
	cdSplitted
	bgSin1
	bgSin2
	bgSin3
)

// Embed assets
//go:embed assets/*
var assets embed.FS

// Config represents user configuration
type Config struct {
	Fullscreen     bool    `json:"fullscreen"`
	VSync          bool    `json:"vsync"`
	MusicVolume    float64 `json:"musicVolume"`
	SpriteCount    int     `json:"spriteCount"`
	DistortionRate float64 `json:"distortionRate"`
	EnableCRT      bool    `json:"enableCRT"`
	EnableGlow     bool    `json:"enableGlow"`
}

// Letter represents a character in the font
type Letter struct {
	char  rune
	x, y  int
	width int
}

// Sprite represents a logo sprite
type Sprite struct {
	x, y  float64
	index int
}

// Delete PrecomputedFrame as it's no longer needed

// Game represents the game state
type Game struct {
	// Images
	backImg  *ebiten.Image
	fontImg  *ebiten.Image
	logoImg  *ebiten.Image

	// Surfaces
	surfMain    *ebiten.Image
	surfScroll  *ebiten.Image
	surfBack    *ebiten.Image
	surfScroll1 *ebiten.Image
	surfScroll2 *ebiten.Image

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player

	// State
	state        string // "intro", "splash", "demo"
	iteration    int
	backWavePos  int
	frontWavePos int
	letterNum    int
	letterDecal  int

	// Intro
	introX      int
	introLetter int
	introTile   int
	introSpeed  int

	// Sprites
	sprites   []*Sprite
	ctrSprite float64

	// Wave tables
	backIntroWaveTable  []int
	backMainWaveTable   []int
	frontIntroWaveTable []int
	frontMainWaveTable  []int

	// Precalc
	curves          [][]int
	backIntroWave   []int
	backMainWave    []int
	frontIntroWave  []int
	frontMainWave   []int
	position        []int

	// Font data
	letterData map[rune]*Letter

	// Text
	text      string
	introText string

	// Config
	config *Config

	// Shaders
	crtShader *ebiten.Shader

	// Transition
	transitionProgress float64
	lastState          string
}

// CRT shader source
const crtShaderSrc = `
package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var uv vec2
	uv = texCoord

	// Barrel distortion
	var dc vec2
	dc = uv - 0.5
	dc = dc * (1.0 + dot(dc, dc) * 0.15)
	uv = dc + 0.5

	// Check bounds
	if uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0 {
		return vec4(0.0, 0.0, 0.0, 1.0)
	}

	// Sample texture
	var col vec4
	col = imageSrc0At(uv)

	// Scanlines
	var scanline float
	scanline = sin(uv.y * 800.0) * 0.04
	col.rgb = col.rgb - scanline

	// RGB shift
	var rShift float
	var bShift float
	rShift = imageSrc0At(uv + vec2(0.002, 0.0)).r
	bShift = imageSrc0At(uv - vec2(0.002, 0.0)).b
	col.r = rShift
	col.b = bShift

	// Vignette
	var vignette float
	vignette = 1.0 - dot(dc, dc) * 0.5
	col.rgb = col.rgb * vignette

	return col * color
}
`

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		state:       "intro",
		introX:      -1,
		introLetter: -1,
		introTile:   -1,
		introSpeed:  4,
		letterData:  make(map[rune]*Letter),
		lastState:   "",
	}

	// Load config
	g.config = g.loadConfig()

	// Initialize sprites based on config
	g.sprites = make([]*Sprite, g.config.SpriteCount)
	for i := 0; i < g.config.SpriteCount; i++ {
		g.sprites[i] = &Sprite{
			x:     float64(screenWidth) / 2,
			y:     float64(screenHeight) / 2,
			index: i,
		}
	}

	// Initialize wave tables
	g.backIntroWaveTable = []int{cdZero, cdZero, cdZero, cdZero, cdZero}
	g.backMainWaveTable = []int{
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		cdSplitted,
	}
	g.frontIntroWaveTable = []int{
		cdZero, cdZero, cdZero, cdZero, cdZero,
		cdZero, cdZero, cdZero, cdZero, cdZero,
		cdFastSin, cdMedSin, cdSlowSin, cdSplitted,
	}
	g.frontMainWaveTable = []int{
		cdSlowSin, cdSlowSin, cdSlowDist, cdSlowSin,
		cdSlowSin, cdMedSin, cdFastSin, cdMedSin,
		cdSlowSin, cdMedDist, cdMedSin, cdSlowSin,
		cdSplitted,
	}

	// Initialize text
	spc := "     "
	g.text = spc + spc + spc +
		"BILIZIR PRESENTS HIS SECOND DEMO-SCREEN IN GOLANG USING EBITEN." + spc +
		"THE CREDITS FOR THIS SCREEN : " +
		"ORIGINAL SCREEN AND IDEA BY DYNO, " +
		"CODED IN GOLANG BY BILIZIR FROM DMA, " +
		"ORIGINAL FONT BY OXAR, " +
		"BACKGROUND BY AGENT-T CREAM, " +
		"MUSIC BY MAD MAX FROM THE EXCEPTIONS." + spc +
		"AND NOW, SOME GREETING :  " +
		"MEGA-GREETINGS TO ALL MEMBERS OF DMA (PDM, COCO, JINX, TWISTER, DWORKIN) AND ALL MEMBERS OF THE UNION ! " +
		"LAST BUT NOT LEAST, I'D LIKE TO SEND A SPECIAL DEDICATION TO ALL DEMOSCENE LOVERS " + spc +
		"IT'S NOW TIME TO WRAP !" + spc

	g.introText = spc +
		"ONCE UPON A TIME, THERE WAS A SCREEN CALLED <THE PARALLAX DISTORTER> BY ULM.      " +
		"35 YEARS LATER, JUST FOR FUN, BILIZIR RECODED A VERSION IN GOLANG (ADAPTED FROM DYNO'S VERSION) !                    ";

	return g
}

// loadConfig loads or creates default configuration
func (g *Game) loadConfig() *Config {
	data, err := os.ReadFile("config.json")
	if err != nil {
		// Return default config
		return &Config{
			Fullscreen:     false,
			VSync:          true,
			MusicVolume:    0.7,
			SpriteCount:    10,
			DistortionRate: 1.0,
			EnableCRT:      true,
			EnableGlow:     true,
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return g.loadConfig() // Return default on parse error
	}

	// Validate config
	if cfg.SpriteCount <= 0 {
		cfg.SpriteCount = 10
	}
	if cfg.DistortionRate <= 0 {
		cfg.DistortionRate = 1.0
	}
	if cfg.MusicVolume < 0 || cfg.MusicVolume > 1 {
		cfg.MusicVolume = 0.7
	}

	return &cfg
}

// Init initializes the game
func (g *Game) Init() error {
	// Set performance options
	ebiten.SetMaxTPS(60)
	ebiten.SetVsyncEnabled(g.config.VSync)

	// Load images from embedded assets
	var err error

	// Load back image
	backData, _ := assets.ReadFile("assets/back.png")
	g.backImg, _, err = ebitenutil.NewImageFromReader(strings.NewReader(string(backData)))
	if err != nil {
		// Create placeholder if not found
		g.backImg = ebiten.NewImage(8, backHeight)
		g.backImg.Fill(color.RGBA{64, 32, 128, 255})
	}

	// Load font image
	fontData, _ := assets.ReadFile("assets/font.png")
	g.fontImg, _, err = ebitenutil.NewImageFromReader(strings.NewReader(string(fontData)))
	if err != nil {
		return err
	}

	// Load logo image
	logoData, _ := assets.ReadFile("assets/logo.png")
	g.logoImg, _, err = ebitenutil.NewImageFromReader(strings.NewReader(string(logoData)))
	if err != nil {
		// Create placeholder logo
		g.logoImg = ebiten.NewImage(spriteSize, spriteSize)
		g.logoImg.Fill(color.RGBA{255, 255, 0, 255})
	}

	// Create surfaces
	g.surfMain = ebiten.NewImage(screenWidth, screenHeight)
	g.surfScroll = ebiten.NewImage(int(math.Round(screenWidth*1.6)), fontHeight)
	g.surfBack = ebiten.NewImage(screenWidth+256, backHeight) // More width for distortion
	g.surfScroll1 = ebiten.NewImage(screenWidth+48, fontHeight)
	g.surfScroll2 = ebiten.NewImage(screenWidth+48, fontHeight)

	// Initialize font data
	g.initFontData()

	// Initialize curves
	g.curves = make([][]int, 11)
	g.createCurves()

	// Precalculate
	g.precalcPosition()
	g.precalcWave(g.frontIntroWaveTable, &g.frontIntroWave)
	g.precalcWave(g.frontMainWaveTable, &g.frontMainWave)
	g.precalcWave(g.backIntroWaveTable, &g.backIntroWave)
	g.precalcWave(g.backMainWaveTable, &g.backMainWave)

	// Prepare background surface
	g.surfBack.Clear()
	// Fill the entire background surface with tiled background image
	for i := 0; i < g.surfBack.Bounds().Dx(); i += g.backImg.Bounds().Dx() {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i), 0)
		g.surfBack.DrawImage(g.backImg, op)
	}

	// Initialize audio
	g.audioContext = audio.NewContext(44100)

	// Load and play music
	if err := g.loadMusic(); err != nil {
		log.Printf("Warning: Could not load music: %v", err)
	}

	// Compile CRT shader if enabled
	if g.config.EnableCRT {
		g.crtShader, err = ebiten.NewShader([]byte(crtShaderSrc))
		if err != nil {
			log.Printf("Warning: Could not compile CRT shader: %v", err)
			g.config.EnableCRT = false
		}
	}

	return nil
}

// loadMusic loads and plays the music file
func (g *Game) loadMusic() error {
	musicData, err := assets.ReadFile("assets/music.mp3")
	if err != nil {
		return err
	}

	d, err := mp3.DecodeWithSampleRate(g.audioContext.SampleRate(), strings.NewReader(string(musicData)))
	if err != nil {
		return err
	}

	loop := audio.NewInfiniteLoop(d, d.Length())
	g.audioPlayer, err = g.audioContext.NewPlayer(loop)
	if err != nil {
		return err
	}

	g.audioPlayer.SetVolume(g.config.MusicVolume)
	g.audioPlayer.Play()
	return nil
}

// initFontData initializes the font character data
func (g *Game) initFontData() {
	data := []struct {
		char  rune
		x, y  int
		width int
	}{
		{' ', 0, 0, 32},
		{'!', 48, 0, 16},
		{'"', 96, 0, 32},
		{'\'', 336, 0, 16},
		{'(', 384, 0, 32},
		{')', 432, 0, 32},
		{'+', 48, 36, 48},
		{',', 96, 36, 16},
		{'-', 144, 36, 32},
		{'.', 192, 36, 16},
		{'0', 288, 36, 48},
		{'1', 336, 36, 48},
		{'2', 384, 36, 48},
		{'3', 432, 36, 48},
		{'4', 0, 72, 48},
		{'5', 48, 72, 48},
		{'6', 96, 72, 48},
		{'7', 144, 72, 48},
		{'8', 192, 72, 48},
		{'9', 240, 72, 48},
		{':', 288, 72, 16},
		{';', 336, 72, 16},
		{'<', 384, 72, 32},
		{'=', 432, 72, 32},
		{'>', 0, 108, 32},
		{'?', 48, 108, 48},
		{'A', 144, 108, 48},
		{'B', 192, 108, 48},
		{'C', 240, 108, 48},
		{'D', 288, 108, 48},
		{'E', 336, 108, 48},
		{'F', 384, 108, 48},
		{'G', 432, 108, 48},
		{'H', 0, 144, 48},
		{'I', 48, 144, 16},
		{'J', 96, 144, 48},
		{'K', 144, 144, 48},
		{'L', 192, 144, 48},
		{'M', 240, 144, 48},
		{'N', 288, 144, 48},
		{'O', 336, 144, 48},
		{'P', 384, 144, 48},
		{'Q', 432, 144, 48},
		{'R', 0, 180, 48},
		{'S', 48, 180, 48},
		{'T', 96, 180, 48},
		{'U', 144, 180, 48},
		{'V', 192, 180, 48},
		{'W', 240, 180, 48},
		{'X', 288, 180, 48},
		{'Y', 336, 180, 48},
		{'Z', 384, 180, 48},
	}

	for _, d := range data {
		g.letterData[d.char] = &Letter{
			char:  d.char,
			x:     d.x,
			y:     d.y,
			width: d.width,
		}
	}
}

// createCurves generates the wave curves
func (g *Game) createCurves() {
	for funcType := 0; funcType <= 10; funcType++ {
		var step, progress float64

		switch funcType {
		case cdZero:
			step, progress = 2.25, 0
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
			step, progress = 0.18, 0
		case bgSin1:
			step, progress = 0.50, 0
		case bgSin2:
			step, progress = 0.80, 0
		case bgSin3:
			step, progress = 0.50, 0
		}

		// Apply distortion rate from config
		step *= g.config.DistortionRate

		local := []float64{}
		decal := 0.0
		previous := 0
		maxAngle := 360.0
		if funcType == cdSplitted {
			maxAngle = 720.0
		}

		for i := 0.0; i < maxAngle-step; i += step {
			val := 0.0
			rad := i * math.Pi / 180

			switch funcType {
			case cdZero:
				val = 0
			case cdSlowSin:
				val = 100 * math.Sin(rad)
			case cdMedSin:
				val = 110 * math.Sin(rad)
			case cdFastSin:
				val = 120 * math.Sin(rad)
			case cdSlowDist:
				val = 100*math.Sin(rad) + 25.0*math.Sin(rad*10)
			case cdMedDist:
				val = 110*math.Sin(rad) + 27.5*math.Sin(rad*9)
			case cdFastDist:
				val = 120*math.Sin(rad) + 30.0*math.Sin(rad*8)
			case cdSplitted:
				dir := 1.0
				if len(local)%2 == 1 {
					dir = -1.0
				}
				amp := 12.0
				if i < 160 {
					amp *= i / 160
				} else if (720 - 160) < i {
					amp *= (720 - i) / 160
				}
				val = 90*math.Sin(rad) + dir*amp*math.Sin(rad*3)
			case bgSin1:
				val = -60 * math.Sin(rad)
			case bgSin2:
				val = -60 * math.Sin(rad)
			case bgSin3:
				val = -60*math.Sin(rad) - 15*math.Sin(rad*4)
			}
			local = append(local, val)
		}

		g.curves[funcType] = make([]int, len(local))
		for i := 0; i < len(local); i++ {
			nitem := -int(math.Floor(local[i] - decal))
			g.curves[funcType][i] = nitem - previous
			previous = nitem
			decal += progress / float64(len(local))
		}
	}
}

// precalcPosition precalculates text positions
func (g *Game) precalcPosition() {
	count := 0
	g.position = []int{}

	for _, r := range g.text {
		if letter, ok := g.letterData[r]; ok {
			count += letter.width
			g.position = append(g.position, count)
		}
	}
}

// precalcWave precalculates wave data
func (g *Game) precalcWave(srcWaveTable []int, dstWaveTable *[]int) {
	count := 0
	*dstWaveTable = []int{}

	for _, waveType := range srcWaveTable {
		wave := g.curves[waveType]
		for _, val := range wave {
			count += val
			*dstWaveTable = append(*dstWaveTable, count)
		}
	}
}

// getSum calculates sum with wrapping
func (g *Game) getSum(arr []int, index, decal int) int {
	n := len(arr)
	if n == 0 {
		return decal
	}

	maxVal := arr[n-1]
	f := index / n
	m := index % n
	return decal + f*maxVal + arr[m]
}

// getWave gets wave value at position
func (g *Game) getWave(i int, introWave, mainWave []int) int {
	if i < len(introWave) {
		return g.getSum(introWave, i, 0)
	}
	return g.getSum(mainWave, i-len(introWave), introWave[len(introWave)-1])
}

// getPosition gets text position
func (g *Game) getPosition(i int) int {
	if i > 0 && i <= len(g.position) {
		return g.getSum(g.position, i-1, 0)
	}
	return 0
}

// getLetter gets letter at position
func (g *Game) getLetter(str string, pos int) rune {
	runes := []rune(str)
	if len(runes) == 0 {
		return ' '
	}
	return runes[pos%len(runes)]
}

// displayText renders text to scroll surface
func (g *Game) displayText(letterOffset int) {
	// Clear to transparent, not black - we want to see the background through
	g.surfScroll.Clear()

	xPos := 0
	i := 0
	for xPos < g.surfScroll.Bounds().Dx() {
		char := g.getLetter(g.text, i+letterOffset)
		if letter, ok := g.letterData[char]; ok {
			srcRect := image.Rect(letter.x, letter.y, letter.x+letter.width, letter.y+fontHeight)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(xPos), 0)
			g.surfScroll.DrawImage(g.fontImg.SubImage(srcRect).(*ebiten.Image), op)
			xPos += letter.width
		}
		i++
	}
}

// updateSprites updates sprite positions
func (g *Game) updateSprites() {
	for i := 0; i < len(g.sprites); i++ {
		c := g.ctrSprite + float64(i)*0.155

		centerX := float64(screenWidth) / 2
		centerY := float64(screenHeight) / 2

		posX := centerX + 100*math.Sin(c*1.35+1.25) + 100*math.Sin(c*1.86+0.54)
		posY := centerY + 60*math.Cos(c*1.72+0.23) + 60*math.Cos(c*1.63+0.98)

		posX += 20 * math.Sin(float64(i)*0.289+1.15)
		posY += 20 * math.Cos(float64(i)*0.456+0.85)

		halfSize := float64(spriteSize) / 2
		if posX < halfSize {
			posX = halfSize
		} else if posX > screenWidth-halfSize {
			posX = screenWidth - halfSize
		}

		if posY < halfSize {
			posY = halfSize
		} else if posY > screenHeight-halfSize {
			posY = screenHeight - halfSize
		}

		g.sprites[i].x = posX
		g.sprites[i].y = posY
	}
}

// drawGlowSprite draws a sprite with glow effect
func (g *Game) drawGlowSprite(screen *ebiten.Image, sprite *Sprite) {
	if g.config.EnableGlow {
		// Draw glow layers
		for i := 3; i > 0; i-- {
			op := &ebiten.DrawImageOptions{}
			scale := zoom + float64(i)*0.1
			op.GeoM.Translate(-float64(spriteSize)/2, -float64(spriteSize)/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(sprite.x*zoom, sprite.y*zoom)
			op.ColorM.Scale(1, 1, 1, 0.3/float64(i))
			op.Filter = ebiten.FilterLinear
			screen.DrawImage(g.logoImg, op)
		}
	}

	// Draw main sprite
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(spriteSize)/2, -float64(spriteSize)/2)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(sprite.x*zoom, sprite.y*zoom)
	screen.DrawImage(g.logoImg, op)
}

// animIntro handles intro animation
func (g *Game) animIntro() {
	if g.introX < 0 {
		if g.introTile > -1 {
			char := g.getLetter(g.introText, g.introTile)
			if letter, ok := g.letterData[char]; ok {
				g.introX += letter.width
			}
		}
		g.introLetter++
		if g.introLetter >= len([]rune(g.introText)) {
			g.lastState = g.state
			g.state = "splash"
			g.iteration = 0
			g.transitionProgress = 0
			return
		}
		g.introTile = g.introLetter
	}
	g.introX -= g.introSpeed

	// Scroll temp canvas
	g.surfScroll2.Clear()
	srcRect := image.Rect(g.introSpeed, 0, screenWidth+48, fontHeight)
	op := &ebiten.DrawImageOptions{}
	g.surfScroll2.DrawImage(g.surfScroll1.SubImage(srcRect).(*ebiten.Image), op)

	g.surfScroll1.Clear()
	g.surfScroll1.DrawImage(g.surfScroll2, op)

	// Draw letter
	char := g.getLetter(g.introText, g.introTile)
	if letter, ok := g.letterData[char]; ok {
		srcRect := image.Rect(letter.x, letter.y, letter.x+letter.width, letter.y+36)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(screenWidth+g.introX), 0)
		g.surfScroll1.DrawImage(g.fontImg.SubImage(srcRect).(*ebiten.Image), op)
	}

	// Draw to main surface
	g.surfMain.Fill(color.Black)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 170)
	g.surfMain.DrawImage(g.surfScroll1, op)
}

// animSplash handles splash screen
func (g *Game) animSplash() {
	if g.iteration < 90 {
		g.iteration++
		g.transitionProgress = float64(g.iteration) / 90.0
	} else {
		g.lastState = g.state
		g.state = "demo"
		g.iteration = 0
		g.transitionProgress = 0
	}

	g.surfMain.Fill(color.Black)
}

// animDemo handles main demo animation
func (g *Game) animDemo() {
	// Direct calculation without cache for smoother animation
	g.calculateAndRenderDemo()

	// Update counters with fixed increment for smooth animation
	g.iteration++
	g.backWavePos = g.iteration * 5
	g.frontWavePos = g.iteration * 10
	g.ctrSprite += 0.02
}

// calculateAndRenderDemo calculates and renders a demo frame
func (g *Game) calculateAndRenderDemo() {
	// Bounce values - smoother calculation
	bounceBack := int(math.Floor(30.0 * math.Abs(math.Sin(float64(g.iteration)*0.1))))
	bounceFront := int(math.Floor(18.0 * math.Abs(math.Sin(float64(g.iteration)*0.1))))

	// Calculate decal_x
	decalX := 999999999
	for ligne := 0; ligne < screenHeight; ligne++ {
		c := g.getWave(g.frontWavePos+ligne, g.frontIntroWave, g.frontMainWave)
		if c < decalX {
			decalX = c
		}
	}

	if decalX < 0 {
		decalX = 0
	}

	// Calculate first letter
	i := 0
	dir := 0
	if decalX > g.letterDecal {
		dir = 1
	} else if decalX < g.letterDecal {
		dir = -1
	}

	for decalX < g.getPosition(g.letterNum+i) || g.getPosition(g.letterNum+i+1) <= decalX {
		i += dir
		if g.letterNum+i < 0 || g.letterNum+i >= len(g.position) {
			break
		}
	}
	g.letterNum += i
	if g.letterNum < 0 {
		g.letterNum = 0
	} else if g.letterNum >= len(g.position) {
		g.letterNum = len(g.position) - 1
	}
	g.letterDecal = g.getPosition(g.letterNum)

	// Display text
	g.displayText(g.letterNum)

	// Render to main surface
	g.surfMain.Clear()

	// Draw line by line for proper layering
	for ligne := 0; ligne < screenHeight; ligne++ {
		// Background
		backWave := g.getWave(g.backWavePos+ligne, g.backIntroWave, g.backMainWave)
		backX := (80 + backWave/2) % g.backImg.Bounds().Dx()

		// Ensure we have enough width for the distortion
		srcWidth := screenWidth
		if backX+srcWidth > g.surfBack.Bounds().Dx() {
			// Wrap around if needed
			backX = backX % g.surfBack.Bounds().Dx()
		}

		// Draw background line using DrawImage
		srcRect := image.Rect(backX, (ligne+bounceBack)%backHeight, backX+srcWidth, ((ligne+bounceBack)%backHeight)+1)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(ligne))
		g.surfMain.DrawImage(g.surfBack.SubImage(srcRect).(*ebiten.Image), op)

		// Text scroll
		frontWave := g.getWave(g.frontWavePos+ligne, g.frontIntroWave, g.frontMainWave)
		scrollX := frontWave - g.letterDecal

		if scrollX >= 0 && scrollX < g.surfScroll.Bounds().Dx()-screenWidth {
			srcRect := image.Rect(scrollX, (ligne+bounceFront)%fontHeight, scrollX+screenWidth, ((ligne+bounceFront)%fontHeight)+1)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, float64(ligne))
			g.surfMain.DrawImage(g.surfScroll.SubImage(srcRect).(*ebiten.Image), op)
		}
	}
}

// Delete renderDemoFrame as it's no longer needed

// drawTransition draws transition effects between states
func (g *Game) drawTransition(screen *ebiten.Image, progress float64) {
	if progress > 0 && progress < 1 {
		overlay := ebiten.NewImage(screenWidth*zoom, screenHeight*zoom)

		// Fade effect
		alpha := uint8(255 * (1 - progress))
		overlay.Fill(color.RGBA{0, 0, 0, alpha})

		// Optional: Add more complex transition effects
		if g.lastState == "splash" && g.state == "demo" {
			// Zoom in effect
			op := &ebiten.DrawImageOptions{}
			scale := 1.0 + (1.0-progress)*0.2
			op.GeoM.Translate(-float64(screenWidth*zoom)/2, -float64(screenHeight*zoom)/2)
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(float64(screenWidth*zoom)/2, float64(screenHeight*zoom)/2)
			screen.DrawImage(overlay, op)
		} else {
			screen.DrawImage(overlay, nil)
		}
	}
}

// Update updates the game state
func (g *Game) Update() error {
	// Handle fullscreen toggle
	if ebiten.IsKeyPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Handle volume control
	if g.audioPlayer != nil {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			vol := g.audioPlayer.Volume() + 0.01
			if vol > 1.0 {
				vol = 1.0
			}
			g.audioPlayer.SetVolume(vol)
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			vol := g.audioPlayer.Volume() - 0.01
			if vol < 0 {
				vol = 0
			}
			g.audioPlayer.SetVolume(vol)
		}
	}

	// Update based on state
	switch g.state {
	case "intro":
		g.animIntro()
	case "splash":
		g.animSplash()
	case "demo":
		g.animDemo()
	}

	// Update transition
	if g.transitionProgress > 0 && g.transitionProgress < 1 {
		g.transitionProgress += 0.02
		if g.transitionProgress > 1 {
			g.transitionProgress = 1
		}
	}

	return nil
}

// Draw draws the game
func (g *Game) Draw(screen *ebiten.Image) {
	// Apply CRT shader only in intro state
	if g.config.EnableCRT && g.crtShader != nil && g.state == "intro" {
		// Create a temporary image at the target size for the shader
		tmpImg := ebiten.NewImage(screenWidth*zoom, screenHeight*zoom)

		// Draw main surface scaled to the temporary image
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		tmpImg.DrawImage(g.surfMain, op)

		// Apply CRT shader
		shaderOp := &ebiten.DrawRectShaderOptions{}
		shaderOp.Images[0] = tmpImg
		screen.DrawRectShader(screenWidth*zoom, screenHeight*zoom, g.crtShader, shaderOp)
	} else {
		// Draw main surface with zoom (no shader)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(zoom, zoom)
		screen.DrawImage(g.surfMain, op)
	}

	// Draw sprites (always without shader)
	if g.state == "demo" {
		g.updateSprites()
		for _, sprite := range g.sprites {
			g.drawGlowSprite(screen, sprite)
		}
	}

	// Draw transition
	g.drawTransition(screen, g.transitionProgress)

	// Draw debug info (optional)
	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f\nTPS: %0.2f\nSprites: %d\nState: %s\nCRT: %v",
			ebiten.CurrentFPS(),
			ebiten.CurrentTPS(),
			len(g.sprites),
			g.state,
			g.config.EnableCRT && g.state == "intro"))
	}
}

// Layout returns the screen dimensions
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth * zoom, screenHeight * zoom
}

func main() {
	// Set window properties
	ebiten.SetWindowSize(screenWidth*zoom, screenHeight*zoom)
	ebiten.SetWindowTitle("DMA IS BACK IN 2025 - GOLANG/EBITEN POWER :)")
	ebiten.SetWindowResizable(true)

	// Create and initialize game
	game := NewGame()
	if err := game.Init(); err != nil {
		log.Fatal(err)
	}

	// Run game
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// --- Additional files ---

// go.mod
/*
module megadist

go 1.24.3

require github.com/hajimehoshi/ebiten/v2 v2.8.8

require (
	github.com/ebitengine/gomobile v0.0.0-20240911145611-4856209ac325 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/oto/v3 v3.3.3 // indirect
	github.com/ebitengine/purego v0.8.0 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
)
*/

// config.json (example)
/*
{
    "fullscreen": false,
    "vsync": true,
    "musicVolume": 0.7,
    "spriteCount": 10,
    "distortionRate": 1.0,
    "enableCRT": true,
    "enableGlow": true
}
*/

// build.sh
/*
#!/bin/bash

# Build script for MegaDist

echo "Building MegaDist..."

# Build for current platform
go build -ldflags="-s -w" -o megadist

# Cross-compile for other platforms
echo "Cross-compiling..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o megadist-windows.exe
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o megadist-macos
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o megadist-linux

echo "Build complete!"
*/

// README.md
/*
# MegaDist - Parallax Distorter

A Go/Ebiten port of the classic Atari ST demo "ParaDis3" by Dyno.

## Features

- Faithful recreation of the original parallax distortion effects
- Sinusoidal text scrolling with multiple wave patterns
- Background parallax scrolling
- Animated logo sprites with complex trajectories
- Optional CRT shader effect
- Glow effects on sprites
- Configurable settings via config.json

## Requirements

- Go 1.21 or higher
- Ebiten v2.6.0 or higher

## Building

```bash
go mod init megadist
go mod tidy
go build -o megadist
```

## Running

```bash
./megadist
```

## Controls

- F11: Toggle fullscreen
- Up/Down arrows: Adjust volume
- Tab: Show debug information

## Configuration

Edit `config.json` to customize:
- Screen mode (fullscreen/windowed)
- VSync
- Music volume
- Number of sprites
- Distortion rate
- Visual effects (CRT, glow)

## Assets Required

Place in `assets/` directory:
- back.png: Background tile (8x64 pixels)
- font.png: Bitmap font (480x216 pixels)
- logo.png: Sprite image (32x32 pixels)
- music.mp3: Background music

*/

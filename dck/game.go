// Package megatwist implements the MegaTwist Atari ST demo remake.
package megatwist

import originalassets "megatwist"

import (
	"bytes"

	"fmt"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 416
	screenHeight = 276
	zoom         = 2
	backHeight   = 64
	fontHeight   = 36
	spriteSize   = 32
	scrollWidth  = (screenWidth*8 + 4) / 5

	// ContentWidth and ContentHeight are the native desktop canvas size.
	ContentWidth   = screenWidth * zoom
	ContentHeight  = screenHeight * zoom
	maxLayoutWidth = 1280
)

const WindowTitle = "DMA IS BACK IN 2025 - GOLANG/EBITEN POWER :)"

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

// Only runtime assets are embedded. The legacy 15 MB MP3 remains in the
// repository but is deliberately excluded from desktop binaries and APKs.
var assets = originalassets.DCKAssetAssets()

type gameState uint8

const (
	stateIntro gameState = iota
	stateSplash
	stateDemo
)

func (s gameState) String() string {
	switch s {
	case stateIntro:
		return "intro"
	case stateSplash:
		return "splash"
	case stateDemo:
		return "demo"
	default:
		return "unknown"
	}
}

type Letter struct {
	width int
	glyph *ebiten.Image
}

type Sprite struct {
	x float64
	y float64
}

// Game contains the shared desktop and Android game state.
type Game struct {
	scrollRenderer              *scrolling.Scrolling
	backgroundBatch, frontBatch *composite.QuadBatch
	backImg                     *ebiten.Image
	fontImg                     *ebiten.Image
	logoImg                     *ebiten.Image

	surfMain    *ebiten.Image
	surfScroll  *ebiten.Image
	surfBack    *ebiten.Image
	surfScroll1 *ebiten.Image
	surfScroll2 *ebiten.Image
	frame       *ebiten.Image
	scaledMain  *ebiten.Image
	overlay     *ebiten.Image

	audioContext *audio.Context
	audioPlayer  *audio.Player
	ymPlayer     *YMPlayer
	audioReady   bool

	state        gameState
	iteration    int
	backWavePos  int
	frontWavePos int
	letterNum    int
	letterDecal  int

	introX      int
	introLetter int
	introTile   int
	introSpeed  int

	sprites   []Sprite
	ctrSprite float64

	backIntroWave  []int
	backMainWave   []int
	frontIntroWave []int
	frontMainWave  []int
	position       []int

	backgroundVertices []ebiten.Vertex
	backgroundIndices  []uint16
	scrollVertices     []ebiten.Vertex
	scrollIndices      []uint16

	letterData      map[rune]Letter
	text            []rune
	introText       []rune
	displayedLetter int
	config          *Config
	crtShader       *ebiten.Shader

	transitionProgress float64
	lastState          gameState
}

const crtShaderSrc = `
package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var uv vec2
	uv = texCoord

	var dc vec2
	dc = uv - 0.5
	dc = dc * (1.0 + dot(dc, dc) * 0.15)
	uv = dc + 0.5

	if uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0 {
		return vec4(0.0, 0.0, 0.0, 1.0)
	}

	var col vec4
	col = imageSrc0At(uv)
	var scanline float
	scanline = sin(uv.y * 800.0) * 0.04
	col.rgb = col.rgb - scanline

	var rShift float
	var bShift float
	rShift = imageSrc0At(uv + vec2(0.002, 0.0)).r
	bShift = imageSrc0At(uv - vec2(0.002, 0.0)).b
	col.r = rShift
	col.b = bShift

	var vignette float
	vignette = 1.0 - dot(dc, dc) * 0.5
	col.rgb = col.rgb * vignette
	return col * color
}
`

// NewGame builds the platform-independent game. Audio is intentionally opened
// later, on the first Update, once Android has installed its native context.
func NewGame() *Game {
	g := &Game{
		state:           stateIntro,
		lastState:       stateIntro,
		introX:          -1,
		introLetter:     -1,
		introTile:       -1,
		introSpeed:      4,
		displayedLetter: -1,
		letterData:      make(map[rune]Letter, 48),
		config:          loadConfig(),
	}

	g.sprites = make([]Sprite, g.config.SpriteCount)
	for i := range g.sprites {
		g.sprites[i].x = float64(screenWidth) / 2
		g.sprites[i].y = float64(screenHeight) / 2
	}

	const spaces = "               "
	g.text = []rune(spaces +
		"BILIZIR PRESENTS HIS SECOND DEMO-SCREEN IN GOLANG USING EBITEN.     " +
		"THE CREDITS FOR THIS SCREEN : " +
		"ORIGINAL SCREEN AND IDEA BY DYNO, " +
		"CODED IN GOLANG BY BILIZIR FROM DMA, " +
		"ORIGINAL FONT BY OXAR, " +
		"BACKGROUND BY AGENT-T CREAM, " +
		"MUSIC BY MAD MAX FROM THE EXCEPTIONS.     " +
		"AND NOW, SOME GREETING :  " +
		"MEGA-GREETINGS TO ALL MEMBERS OF DMA (PDM, COCO, JINX, TWISTER, DWORKIN) AND ALL MEMBERS OF THE UNION ! " +
		"LAST BUT NOT LEAST, I'D LIKE TO SEND A SPECIAL DEDICATION TO ALL DEMOSCENE LOVERS     " +
		"IT'S NOW TIME TO WRAP !     ")
	g.introText = []rune("     " +
		"ONCE UPON A TIME, THERE WAS A SCREEN CALLED <THE PARALLAX DISTORTER> BY ULM.      " +
		"35 YEARS LATER, JUST FOR FUN, BILIZIR RECODED A VERSION IN GOLANG (ADAPTED FROM DYNO'S VERSION) !                    ")

	if err := g.initialize(); err != nil {
		panic(fmt.Sprintf("initialize MegaTwist: %v", err))
	}
	return g
}

func (g *Game) initialize() error {
	ebiten.SetTPS(60)
	ebiten.SetVsyncEnabled(g.config.VSync)

	var err error
	g.backImg, err = loadImage("assets/back.png")
	if err != nil {
		log.Printf("could not load background, using placeholder: %v", err)
		g.backImg = ebiten.NewImage(8, backHeight)
		g.backImg.Fill(color.RGBA{64, 32, 128, 255})
	}
	g.fontImg, err = loadImage("assets/font.png")
	if err != nil {
		return err
	}
	g.logoImg, err = loadImage("assets/logo.png")
	if err != nil {
		log.Printf("could not load logo, using placeholder: %v", err)
		g.logoImg = ebiten.NewImage(spriteSize, spriteSize)
		g.logoImg.Fill(color.RGBA{255, 255, 0, 255})
	}

	g.surfMain = ebiten.NewImage(screenWidth, screenHeight)
	g.surfScroll = ebiten.NewImage(scrollWidth, fontHeight)
	g.surfBack = ebiten.NewImage(screenWidth+256, backHeight)
	g.surfScroll1 = ebiten.NewImage(screenWidth+48, fontHeight)
	g.surfScroll2 = ebiten.NewImage(screenWidth+48, fontHeight)
	g.frame = ebiten.NewImage(ContentWidth, ContentHeight)
	g.scaledMain = ebiten.NewImage(ContentWidth, ContentHeight)
	g.overlay = ebiten.NewImage(ContentWidth, ContentHeight)

	g.initFontData()
	curves := createCurves(g.config.DistortionRate)
	g.frontIntroWave = precalcWave(curves, []int{
		cdZero, cdZero, cdZero, cdZero, cdZero,
		cdZero, cdZero, cdZero, cdZero, cdZero,
		cdFastSin, cdMedSin, cdSlowSin, cdSplitted,
	})
	g.frontMainWave = precalcWave(curves, []int{
		cdSlowSin, cdSlowSin, cdSlowDist, cdSlowSin,
		cdSlowSin, cdMedSin, cdFastSin, cdMedSin,
		cdSlowSin, cdMedDist, cdMedSin, cdSlowSin,
		cdSplitted,
	})
	g.backIntroWave = precalcWave(curves, []int{cdZero, cdZero, cdZero, cdZero, cdZero})
	g.backMainWave = precalcWave(curves, []int{
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		cdSplitted,
	})
	g.precalcPosition()

	for x := 0; x < g.surfBack.Bounds().Dx(); x += g.backImg.Bounds().Dx() {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), 0)
		g.surfBack.DrawImage(g.backImg, op)
	}

	g.backgroundVertices = make([]ebiten.Vertex, 0, screenHeight*4)
	g.backgroundIndices = make([]uint16, 0, screenHeight*6)
	g.scrollVertices = make([]ebiten.Vertex, 0, screenHeight*4)
	g.scrollIndices = make([]uint16, 0, screenHeight*6)

	if g.config.EnableCRT {
		g.crtShader, err = ebiten.NewShader([]byte(crtShaderSrc))
		if err != nil {
			log.Printf("could not compile CRT shader: %v", err)
			g.config.EnableCRT = false
		}
	}
	return nil
}

func loadImage(name string) (*ebiten.Image, error) {
	data, err := assets.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	return img, nil
}

func (g *Game) initFontData() {
	definitions := []struct {
		char  rune
		x, y  int
		width int
	}{
		{' ', 0, 0, 32}, {'!', 48, 0, 16}, {'"', 96, 0, 32}, {'\'', 336, 0, 16},
		{'(', 384, 0, 32}, {')', 432, 0, 32}, {'+', 48, 36, 48}, {',', 96, 36, 16},
		{'-', 144, 36, 32}, {'.', 192, 36, 16}, {'0', 288, 36, 48}, {'1', 336, 36, 48},
		{'2', 384, 36, 48}, {'3', 432, 36, 48}, {'4', 0, 72, 48}, {'5', 48, 72, 48},
		{'6', 96, 72, 48}, {'7', 144, 72, 48}, {'8', 192, 72, 48}, {'9', 240, 72, 48},
		{':', 288, 72, 16}, {';', 336, 72, 16}, {'<', 384, 72, 32}, {'=', 432, 72, 32},
		{'>', 0, 108, 32}, {'?', 48, 108, 48}, {'A', 144, 108, 48}, {'B', 192, 108, 48},
		{'C', 240, 108, 48}, {'D', 288, 108, 48}, {'E', 336, 108, 48}, {'F', 384, 108, 48},
		{'G', 432, 108, 48}, {'H', 0, 144, 48}, {'I', 48, 144, 16}, {'J', 96, 144, 48},
		{'K', 144, 144, 48}, {'L', 192, 144, 48}, {'M', 240, 144, 48}, {'N', 288, 144, 48},
		{'O', 336, 144, 48}, {'P', 384, 144, 48}, {'Q', 432, 144, 48}, {'R', 0, 180, 48},
		{'S', 48, 180, 48}, {'T', 96, 180, 48}, {'U', 144, 180, 48}, {'V', 192, 180, 48},
		{'W', 240, 180, 48}, {'X', 288, 180, 48}, {'Y', 336, 180, 48}, {'Z', 384, 180, 48},
	}

	for _, definition := range definitions {
		rect := image.Rect(
			definition.x,
			definition.y,
			definition.x+definition.width,
			definition.y+fontHeight,
		)
		g.letterData[definition.char] = Letter{
			width: definition.width,
			glyph: g.fontImg.SubImage(rect).(*ebiten.Image),
		}
	}
}

func (g *Game) Update() error {
	if !g.audioReady {
		g.audioReady = true
		g.initAudio()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if g.audioPlayer != nil {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			g.audioPlayer.SetVolume(min(g.audioPlayer.Volume()+0.01, 1))
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			g.audioPlayer.SetVolume(max(g.audioPlayer.Volume()-0.01, 0))
		}
	}

	switch g.state {
	case stateIntro:
		g.animIntro()
	case stateSplash:
		g.animSplash()
	case stateDemo:
		g.animDemo()
	}

	if g.transitionProgress > 0 && g.transitionProgress < 1 {
		g.transitionProgress = min(g.transitionProgress+0.02, 1)
	}
	return nil
}

func logicalWidth(outsideWidth, outsideHeight int) int {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		return ContentWidth
	}
	width := (outsideWidth*ContentHeight + outsideHeight - 1) / outsideHeight
	return min(max(width, ContentWidth), maxLayoutWidth)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return logicalWidth(outsideWidth, outsideHeight), ContentHeight
}

// InitialFullscreen reports the optional desktop startup preference.
func (g *Game) InitialFullscreen() bool {
	return g.config.Fullscreen
}

// DebugSummary avoids formatting state in the hot path unless the debug
// overlay is actually visible.
func (g *Game) DebugSummary() string {
	return fmt.Sprintf(
		"FPS: %0.2f\nTPS: %0.2f\nSprites: %d\nState: %s\nCRT: %v",
		ebiten.ActualFPS(),
		ebiten.ActualTPS(),
		len(g.sprites),
		g.state,
		g.config.EnableCRT && g.state == stateIntro,
	)
}

// Compile-time check that Game still satisfies Ebitengine's shared interface.
var _ ebiten.Game = (*Game)(nil)

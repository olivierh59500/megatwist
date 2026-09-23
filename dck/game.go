// Package megatwist implements the MegaTwist Atari ST demo remake.
package megatwist

import (
	"bytes"
	"fmt"
	"github.com/olivierh59500/democonstructionkit/presets"
	"image/color"
	"log"
	originalassets "megatwist"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"

	audio "github.com/olivierh59500/democonstructionkit/sound/output"
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
	musicStream  *sound.Stream
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

	frontProgram, backProgram *composite.DisplacementProgram
	frontRows, backRows       [screenHeight]int
	position                  []int

	backgroundVertices []ebiten.Vertex
	backgroundIndices  []uint16
	scrollVertices     []ebiten.Vertex
	scrollIndices      []uint16

	fontAtlas       *scrolling.Atlas
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
	frontIntroWave := precalcWave(curves, []int{
		cdZero, cdZero, cdZero, cdZero, cdZero,
		cdZero, cdZero, cdZero, cdZero, cdZero,
		cdFastSin, cdMedSin, cdSlowSin, cdSplitted,
	})
	frontMainWave := precalcWave(curves, []int{
		cdSlowSin, cdSlowSin, cdSlowDist, cdSlowSin,
		cdSlowSin, cdMedSin, cdFastSin, cdMedSin,
		cdSlowSin, cdMedDist, cdMedSin, cdSlowSin,
		cdSplitted,
	})
	backIntroWave := precalcWave(curves, []int{cdZero, cdZero, cdZero, cdZero, cdZero})
	backMainWave := precalcWave(curves, []int{
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3,
		cdSplitted,
	})

	g.frontProgram, err = composite.NewDisplacementProgram(frontIntroWave, frontMainWave)
	if err != nil {
		return err
	}
	g.backProgram, err = composite.NewDisplacementProgram(backIntroWave, backMainWave)
	if err != nil {
		return err
	}
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
	var err error
	g.fontAtlas, err = presets.FontAtlas("megatwist", g.fontImg)
	if err != nil {
		panic(err)
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

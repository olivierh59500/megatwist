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
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/democonstructionkit/timeline"

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

// Game contains the shared desktop and Android game state.
type Game struct {
	mainScroll *scrolling.Scrolling
	background *composite.ScanlineBackground
	backImg    *ebiten.Image
	fontImg    *ebiten.Image
	logoImg    *ebiten.Image

	surfMain   *ebiten.Image
	frame      *ebiten.Image
	scaledMain *ebiten.Image

	audioContext *audio.Context
	audioPlayer  *audio.Player
	musicStream  *sound.Stream
	audioReady   bool

	state     gameState
	iteration int
	splash    *timeline.HoldRamp

	introScroll *scrolling.Scrolling

	spriteGroup *sprites.Group
	spriteGlow  *sprites.GlowPainter

	fontAtlas *scrolling.Atlas
	text      string
	introText string
	config    *Config
	crtShader *ebiten.Shader
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
		state:  stateIntro,
		config: loadConfig(),
	}
	var err error
	g.splash, err = timeline.NewHoldRamp(presets.MegaTwistSplashRamp())
	if err != nil {
		panic(err)
	}

	const spaces = "               "
	g.text = spaces +
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
		"IT'S NOW TIME TO WRAP !     "
	g.introText = "     " +
		"ONCE UPON A TIME, THERE WAS A SCREEN CALLED <THE PARALLAX DISTORTER> BY ULM.      " +
		"35 YEARS LATER, JUST FOR FUN, BILIZIR RECODED A VERSION IN GOLANG (ADAPTED FROM DYNO'S VERSION) !                    "

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
	formation, err := motion.NewHarmonicFormation(presets.MegaTwistSpriteFormationConfig(screenWidth, screenHeight, spriteSize))
	if err != nil {
		return err
	}
	g.spriteGroup, err = sprites.NewGroup(sprites.GroupConfig{
		Frames: []*ebiten.Image{g.logoImg}, Count: g.config.SpriteCount,
		Harmonic: formation, HarmonicClockStep: [2]float64{.02},
	})
	if err != nil {
		return err
	}
	glowConfig := presets.MegaTwistGlowPainterConfig(spriteSize, zoom)
	if !g.config.EnableGlow {
		glowConfig.Layers = 0
	}
	g.spriteGlow, err = sprites.NewGlowPainter(glowConfig)
	if err != nil {
		return err
	}

	g.surfMain = ebiten.NewImage(screenWidth, screenHeight)
	g.frame = ebiten.NewImage(ContentWidth, ContentHeight)
	g.scaledMain = ebiten.NewImage(ContentWidth, ContentHeight)

	g.initFontData()
	feedConfig := presets.MegaTwistIntroFeed(g.fontAtlas, g.introText)
	g.introScroll, err = scrolling.New(scrolling.Config{Feed: &feedConfig})
	if err != nil {
		return err
	}
	front, back, err := presets.MegaTwistPrograms(g.config.DistortionRate)
	if err != nil {
		return err
	}
	g.background, err = composite.NewScanlineBackground(composite.ScanlineBackgroundConfig{
		Tile: g.backImg, Program: back,
		Width: screenWidth, Height: screenHeight,
		SurfaceWidth: screenWidth + 256, SurfaceHeight: backHeight,
		BaseX: 80, WaveDivisor: 2, WaveStep: 5,
		BounceAmplitude: 30, BounceRate: .1, AlternateDiagonal: true,
	})
	if err != nil {
		return err
	}
	scrollConfig := presets.MegaTwistScanlineScroll(g.fontAtlas, g.text, front)
	g.mainScroll, err = scrolling.New(scrolling.Config{Scanline: &scrollConfig})
	if err != nil {
		return err
	}

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
		if err := g.animIntro(); err != nil {
			return err
		}
	case stateSplash:
		g.animSplash()
	case stateDemo:
		if err := g.animDemo(); err != nil {
			return err
		}
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
		len(g.spriteGroup.Poses()),
		g.state,
		g.config.EnableCRT && g.state == stateIntro,
	)
}

// Compile-time check that Game still satisfies Ebitengine's shared interface.
var _ ebiten.Game = (*Game)(nil)

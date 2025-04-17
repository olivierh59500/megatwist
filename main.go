package main

import (
	"image"
	"image/png"
	"log"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

const (
	screenWidth  = 416
	screenHeight = 276
	zoom         = 2
	backHeight   = 64
	fontHeight   = 36
)

var (
	// Debug flag
	debug = false

	// Images
	backImg *ebiten.Image
	fontImg *ebiten.Image
	logoImg *ebiten.Image

	// Canvases
	mainCanvas   *ebiten.Image
	backCanvas   *ebiten.Image
	scrollCanvas *ebiten.Image

	// Animation state
	iteration    float64
	ctrSprite    float64 // Compteur pour les trajectoires des sprites
	backWavePos  float64
	frontWavePos float64
	letterNum    int
	letterDecal  float64
	startTime    time.Time
	useDeltaTime = 1

	// Sprites
	sprites []*Sprite

	// Wave types
	cdZero     = 0
	cdSlowSin  = 1
	cdMedSin   = 2
	cdFastSin  = 3
	cdSlowDist = 4
	cdMedDist  = 5
	cdFastDist = 6
	cdSplitted = 7
	bgSin1     = 8
	bgSin2     = 9
	bgSin3     = 10

	// Wave tables
	backIntroWaveTable  = []int{cdZero, cdZero, cdZero, cdZero, cdZero}
	backMainWaveTable   = []int{bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3, bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3, bgSin1, bgSin1, bgSin2, bgSin2, bgSin3, bgSin3, cdSplitted}
	frontIntroWaveTable = []int{cdZero, cdZero, cdZero, cdZero, cdZero, cdZero, cdZero, cdZero, cdZero, cdZero, cdFastSin, cdMedSin, cdSlowSin, cdSplitted}
	frontMainWaveTable  = []int{cdSlowSin, cdSlowSin, cdSlowDist, cdSlowSin, cdSlowSin, cdMedSin, cdFastSin, cdMedSin, cdSlowSin, cdMedDist, cdMedSin, cdSlowSin, cdSplitted}

	// Precomputed data
	curve          = make(map[int][]float64)
	backIntroWave  []float64
	backMainWave   []float64
	frontIntroWave []float64
	frontMainWave  []float64
	position       []float64

	// Font mapping
	letter = [][]interface{}{
		{" ", 0, 0, 32}, {"!", 48, 0, 16}, {"\"", 96, 0, 32}, {"'", 336, 0, 16},
		{"(", 384, 0, 32}, {")", 432, 0, 32}, {"+", 48, 36, 48}, {",", 96, 36, 16},
		{"-", 144, 36, 32}, {".", 192, 36, 16}, {"0", 288, 36, 48}, {"1", 336, 36, 48},
		{"2", 384, 36, 48}, {"3", 432, 36, 48}, {"4", 0, 72, 48}, {"5", 48, 72, 48},
		{"6", 96, 72, 48}, {"7", 144, 72, 48}, {"8", 192, 72, 48}, {"9", 240, 72, 48},
		{":", 288, 72, 16}, {";", 336, 72, 16}, {"<", 384, 72, 32}, {"=", 432, 72, 32},
		{">", 0, 108, 32}, {"?", 48, 108, 48}, {"A", 144, 108, 48}, {"B", 192, 108, 48},
		{"C", 240, 108, 48}, {"D", 288, 108, 48}, {"E", 336, 108, 48}, {"F", 384, 108, 48},
		{"G", 432, 108, 48}, {"H", 0, 144, 48}, {"I", 48, 144, 16}, {"J", 96, 144, 48},
		{"K", 144, 144, 48}, {"L", 192, 144, 48}, {"M", 240, 144, 48}, {"N", 288, 144, 48},
		{"O", 336, 144, 48}, {"P", 384, 144, 48}, {"Q", 432, 144, 48}, {"R", 0, 180, 48},
		{"S", 48, 180, 48}, {"T", 96, 180, 48}, {"U", 144, 180, 48}, {"V", 192, 180, 48},
		{"W", 240, 180, 48}, {"X", 288, 180, 48}, {"Y", 336, 180, 48}, {"Z", 384, 180, 48},
	}

	// Text
	spc  = "     "
	text = spc + spc + spc +
		"SALUT DIDIER ET PHILIPPE, EH OUAIS DMA IS BACK IN RETRO-DEMOSCENE." + spc +
		"BILIZIR PROUDLY PRESENTS HIS FIRST DEMO-SCREEN IN GOLANG AND EBITEN" + spc +
		"AND NOW, SOME GREETING : ALL MEMBERS OF DMA  ALL MEMBERS OF THE UNION  ALL DEMOSCENE LOVERS :)" + spc +
		"IT'S NOW TIME TO WRAP !" + spc
)

// Structure pour un sprite
type Sprite struct {
	X, Y       float64
	Trajectory func(float64) (float64, float64)
}

// Créer les 6 sprites avec la logique de sprites(), en évitant les sorties d’écran
func initSprites() {
	nbSprites := 10
	sprites = make([]*Sprite, nbSprites)
	for i := 0; i < nbSprites; i++ {
		spriteIndex := i // Capturer l'index pour la closure
		sprites[i] = &Sprite{
			Trajectory: func(t float64) (float64, float64) {
				// Simuler la logique de sprites()
				c := ctrSprite + float64(spriteIndex)*0.155 // Incrément par sprite
				// Centrer autour de (208, 138)
				centerX, centerY := float64(screenWidth)/2, float64(screenHeight)/2
				// Réduire les amplitudes pour rester dans l’écran
				posx := centerX + 100*math.Sin(c*1.35+1.25) + 100*math.Sin(c*1.86+0.54)
				posy := centerY + 60*math.Cos(c*1.72+0.23) + 60*math.Cos(c*1.63+0.98)
				// Ajustement par sprite
				posx += 20 * math.Sin(float64(spriteIndex)*0.289+1.15)
				posy += 20 * math.Cos(float64(spriteIndex)*0.456+0.85)
				// Limiter les positions pour éviter les sorties d’écran (sprite 32x32)
				if posx < 16 {
					posx = 16
				} else if posx > screenWidth-16 {
					posx = screenWidth - 16
				}
				if posy < 16 {
					posy = 16
				} else if posy > screenHeight-16 {
					posy = screenHeight - 16
				}
				return posx, posy
			},
		}
	}
}

type Game struct {
	audioContext *audio.Context
	player       *audio.Player
}

func init() {
	// Activer le debug via la variable d'environnement DEBUG=1
	if os.Getenv("DEBUG") == "1" {
		debug = true
	}
	// Initialiser les sprites
	initSprites()
}

func toRadians(angle float64) float64 {
	return angle * (math.Pi / 180)
}

func createCurve(funcID int, step, progress float64) {
	var local []float64
	decal, previous := 0.0, 0.0
	limit := 360.0
	if funcID == cdSplitted {
		limit = 720.0
	}
	c := 0
	for i := 0.0; i < limit-step; i += step {
		switch funcID {
		case cdZero:
			local = append(local, 0)
		case cdSlowSin:
			local = append(local, 140*math.Sin(toRadians(i)))
		case cdMedSin:
			local = append(local, 175*math.Sin(toRadians(i)))
		case cdFastSin:
			local = append(local, 210*math.Sin(toRadians(i)))
		case cdSlowDist:
			local = append(local, 100*math.Sin(toRadians(i))+25.0*math.Sin(toRadians(i*10)))
		case cdMedDist:
			local = append(local, 110*math.Sin(toRadians(i))+27.5*math.Sin(toRadians(i*9)))
		case cdFastDist:
			local = append(local, 120*math.Sin(toRadians(i))+30.0*math.Sin(toRadians(i*8)))
		case cdSplitted:
			dir := 1.0
			if c%2 != 0 {
				dir = -1
			}
			amp := 12.0
			if i < 160 {
				amp *= i / 160
			} else if (720 - 160) < i {
				amp *= (720 - i) / 160
			}
			local = append(local, 90*math.Sin(toRadians(i))+dir*amp*math.Sin(toRadians(i*3)))
		case bgSin1:
			local = append(local, -60*math.Sin(toRadians(i)))
		case bgSin2:
			local = append(local, -60*math.Sin(toRadians(i)))
		case bgSin3:
			local = append(local, -60*math.Sin(toRadians(i))-15*math.Sin(toRadians(i*4)))
		}
		c++
	}
	curve[funcID] = make([]float64, len(local))
	for i := 0; i < len(local); i++ {
		nitem := -math.Floor(local[i] - decal)
		curve[funcID][i] = nitem - previous
		previous = nitem
		decal += progress / float64(len(local))
	}
}

func getLetter(str string, pos int) int {
	for idx, l := range letter {
		if l[0].(string) == string(str[pos%len(str)]) {
			return idx
		}
	}
	return 0
}

func displayText(letterDecal int) {
	scrollCanvas.Clear()
	x := 0
	for i := 0; x < scrollCanvas.Bounds().Dx() && i < len(text); i++ {
		j := getLetter(text, i+letterDecal)
		l := letter[j]
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), 0)
		subImgRect := image.Rect(l[1].(int), l[2].(int), l[1].(int)+l[3].(int), l[2].(int)+fontHeight)
		if debug {
			log.Printf("Drawing letter %d, char: %v, rect: %v, x=%d", i, l[0], subImgRect, x)
		}
		scrollCanvas.DrawImage(fontImg.SubImage(subImgRect).(*ebiten.Image), op)
		x += l[3].(int)
	}
	if debug {
		log.Printf("displayText rendered, letterDecal: %d, scrollCanvas size: %v", letterDecal, scrollCanvas.Bounds().Size())
	}
}

func doPrecalcPosition() {
	count, x := 0, 0
	for i := 0; i < len(text); i++ {
		j := getLetter(text, i)
		count += letter[j][3].(int)
		position = append(position, float64(count))
		x++
	}
}

func doPrecalcWave(srcWaveTable []int, dstWaveTable *[]float64) {
	count, x := 0.0, 0
	for _, waveType := range srcWaveTable {
		wave := curve[waveType]
		for _, val := range wave {
			count += val
			*dstWaveTable = append(*dstWaveTable, count)
			x++
		}
	}
}

func getSum(array []float64, index int, decal float64) float64 {
	n := len(array)
	if n == 0 {
		return decal
	}
	max := array[n-1]
	f := index / n
	m := index % n
	return decal + float64(f)*max + array[m]
}

func getWave(i int, introWave, mainWave []float64) float64 {
	if i < len(introWave) {
		return getSum(introWave, i, 0)
	}
	return getSum(mainWave, i-len(introWave), introWave[len(introWave)-1])
}

func getPosition(i int) float64 {
	if i > 0 {
		return getSum(position, i-1, 0)
	}
	return 0
}

func (g *Game) Update() error {
	if useDeltaTime == 1 {
		deltaTime := time.Since(startTime).Seconds()
		iteration = math.Floor(deltaTime * 60)
	} else {
		iteration++
	}
	backWavePos = iteration * 5
	frontWavePos = iteration * 8

	// Incrémenter ctrSprite pour les trajectoires des sprites
	ctrSprite += 0.0253

	// Mettre à jour les positions des sprites
	for i, sprite := range sprites {
		sprite.X, sprite.Y = sprite.Trajectory(iteration)
		if debug {
			log.Printf("Sprite %d position: (%.2f, %.2f)", i, sprite.X, sprite.Y)
		}
	}

	// Stop music on Space key press
	if ebiten.IsKeyPressed(ebiten.KeySpace) && g.player != nil {
		g.player.Pause()
	}

	// Toggle debug mode with 'D' key
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		debug = !debug
		if debug {
			log.Println("Debug mode enabled")
		} else {
			log.Println("Debug mode disabled")
		}
	}

	if debug {
		log.Printf("Update: iteration=%.6f, backWavePos=%.6f, frontWavePos=%.6f, ctrSprite=%.6f", iteration, backWavePos, frontWavePos, ctrSprite)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Bounce values
	bounceBack := math.Floor(math.Abs(30.1 * math.Sin(toRadians(math.Mod(iteration, 42)*4.26))))
	bounceFront := math.Floor(math.Abs(18.1 * math.Sin(toRadians(math.Mod(iteration, 42)*4.26))))

	// Calculate decal_x
	decalX := 999999999.0
	for ligne := 0; ligne < screenHeight; ligne++ {
		c := getWave(int(frontWavePos)+ligne, frontIntroWave, frontMainWave)
		if c < decalX {
			if c > 0 {
				decalX = c
			} else {
				decalX = 0
			}
		}
	}
	if debug {
		log.Printf("decalX: %.6f", decalX)
	}

	// Calculate first letter
	dir := 0
	if decalX > letterDecal {
		dir = 1
	}
	if decalX < letterDecal {
		dir = -1
	}
	i := 0
	for decalX < getPosition(letterNum+i) || getPosition(letterNum+i+1) <= decalX {
		i += dir
	}
	letterNum += i
	letterDecal = getPosition(letterNum)
	if debug {
		log.Printf("letterNum: %d, letterDecal: %.6f", letterNum, letterDecal)
	}

	// Render text to scroll canvas
	displayText(letterNum)

	// Render to main canvas
	mainCanvas.Clear()
	for ligne := 0; ligne < screenHeight; ligne++ {
		// Background
		backXFloat := 80 + math.Floor(getWave(int(backWavePos+float64(ligne)), backIntroWave, backMainWave)/2)
		backX := int(backXFloat) % 8
		if backX < 0 {
			backX += 8
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(ligne))
		backRect := image.Rect(backX, int(math.Mod(float64(ligne)+bounceBack, backHeight)), backX+screenWidth, int(math.Mod(float64(ligne)+bounceBack, backHeight))+1)
		if debug {
			log.Printf("backCanvas SubImage rect: %v", backRect)
		}
		mainCanvas.DrawImage(backCanvas.SubImage(backRect).(*ebiten.Image), op)
		if debug {
			log.Printf("Drew backCanvas line %d, backX: %d", ligne, backX)
		}

		// Scroll text
		scrollX := getWave(int(frontWavePos+float64(ligne)), frontIntroWave, frontMainWave) - letterDecal
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(ligne))
		scrollRect := image.Rect(int(scrollX), int(math.Mod(float64(ligne)+bounceFront, fontHeight)), int(scrollX)+screenWidth, int(math.Mod(float64(ligne)+bounceFront, fontHeight))+1)
		if debug {
			log.Printf("scrollCanvas SubImage rect: %v, scrollX: %f", scrollRect, scrollX)
		}
		mainCanvas.DrawImage(scrollCanvas.SubImage(scrollRect).(*ebiten.Image), op)
		if debug {
			log.Printf("Drew scrollCanvas line %d, scrollX: %f", ligne, scrollX)
		}
	}

	// Dessiner les sprites
	for i, sprite := range sprites {
		op := &ebiten.DrawImageOptions{}
		// Centrer le logo (32x32)
		op.GeoM.Translate(sprite.X-16, sprite.Y-16)
		mainCanvas.DrawImage(logoImg, op)
		if debug {
			log.Printf("Drew sprite %d at (%.2f, %.2f)", i, sprite.X, sprite.Y)
		}
	}

	// Draw main canvas to screen with zoom
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(zoom), float64(zoom))
	screen.DrawImage(mainCanvas, op)
	if debug {
		log.Printf("Rendered mainCanvas to screen, size: %v", mainCanvas.Bounds().Size())
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth * zoom, screenHeight * zoom
}

func (g *Game) CloseAudio() {
	if g.player != nil {
		g.player.Close()
	}
}

func main() {
	// Initialize audio context
	audioContext := audio.NewContext(44100)

	// Load MP3 file
	f, err := os.Open("music.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode MP3
	mp3Stream, err := mp3.DecodeWithSampleRate(44100, f)
	if err != nil {
		log.Fatal(err)
	}

	// Create a looping player
	player, err := audioContext.NewPlayer(audio.NewInfiniteLoop(mp3Stream, mp3Stream.Length()))
	if err != nil {
		log.Fatal(err)
	}

	// Create game instance
	game := &Game{
		audioContext: audioContext,
		player:       player,
	}

	// Start playing the music
	player.Play()

	// Load images
	f, err = os.Open("back3.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	back, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	backImg = ebiten.NewImageFromImage(back)

	f, err = os.Open("font.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	font, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	fontImg = ebiten.NewImageFromImage(font)

	// Charger logo.png
	f, err = os.Open("logo.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	logo, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	logoImg = ebiten.NewImageFromImage(logo)
	if debug {
		log.Printf("logoImg loaded, size: %v", logoImg.Bounds().Size())
	}

	// Initialize canvases
	mainCanvas = ebiten.NewImage(screenWidth, screenHeight)
	if debug {
		log.Printf("mainCanvas initialized, size: %v", mainCanvas.Bounds().Size())
	}
	backCanvas = ebiten.NewImage(screenWidth+8, backHeight)
	if debug {
		log.Printf("backCanvas initialized, size: %v", backCanvas.Bounds().Size())
	}
	scrollCanvas = ebiten.NewImage(screenWidth*16/10, fontHeight)
	if debug {
		log.Printf("scrollCanvas initialized, size: %v", scrollCanvas.Bounds().Size())
	}

	// Initialize back canvas
	for i := 0; i < backCanvas.Bounds().Dx(); i += 8 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(i), 0)
		backCanvas.DrawImage(backImg, op)
	}
	if debug {
		log.Printf("backCanvas filled, size: %v", backCanvas.Bounds().Size())
	}

	// Precompute curves
	createCurve(cdZero, 2.25, 0)
	createCurve(cdSlowSin, 0.20, 140)
	createCurve(cdMedSin, 0.25, 175)
	createCurve(cdFastSin, 0.30, 210)
	createCurve(cdSlowDist, 0.12, 175)
	createCurve(cdMedDist, 0.16, 210)
	createCurve(cdFastDist, 0.20, 245)
	createCurve(cdSplitted, 0.18, 0)
	createCurve(bgSin1, 0.50, 0)
	createCurve(bgSin2, 0.80, 0)
	createCurve(bgSin3, 0.50, 0)

	// Precompute positions and waves
	doPrecalcPosition()
	doPrecalcWave(frontIntroWaveTable, &frontIntroWave)
	doPrecalcWave(frontMainWaveTable, &frontMainWave)
	doPrecalcWave(backIntroWaveTable, &backIntroWave)
	doPrecalcWave(backMainWaveTable, &backMainWave)

	// Initialize start time
	startTime = time.Now()

	// Set up Ebiten
	ebiten.SetWindowSize(screenWidth*zoom, screenHeight*zoom)
	ebiten.SetWindowTitle("DMA IS BACK IN 2025 - GOLANG/EBITEN POWER :)")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}

	// Ensure audio is stopped when the program exits
	defer game.CloseAudio()
}

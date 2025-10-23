package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	displaySize = 300 // Fixed display size in pixels
)

var (
	currentCellSize = 5
	currentGridSize = displaySize / currentCellSize
)

type Cell struct {
	val int
}

type ColorPalette struct {
	dead   color.Color
	young  [5]color.Color
	mature [15]color.Color
	old    [30]color.Color
	cycle  float64 // For palette animation
}

type Stats struct {
	generation   int
	population   int
	density      float64
	avgAge       float64
	entropy      float64
	ageHistogram [50]int
}

type Event struct {
	generation int
	eventType  string
	message    string
}

type SimulationState struct {
	growthRate     float64
	mutationChance float64
	paletteMode    int
	bloomEffect    bool
	events         []Event
	stats          Stats
	isPaused       bool
	isStarted      bool
	cellSize       int
	gridSize       int
	speed          int // ms between each generation
}

type mainThreadRunner interface {
	RunOnMain(func())
}

type mainThreadCaller interface {
	CallOnMainThread(func())
}

func runOnMain(d fyne.Driver, fn func()) {
	switch drv := d.(type) {
	case mainThreadRunner:
		drv.RunOnMain(fn)
	case mainThreadCaller:
		drv.CallOnMainThread(fn)
	default:
		fn()
	}
}

func randomColor(rng *rand.Rand, baseR, baseG, baseB uint8, variance uint8) color.Color {
	r := int(baseR) + rng.Intn(int(variance)*2) - int(variance)
	g := int(baseG) + rng.Intn(int(variance)*2) - int(variance)
	b := int(baseB) + rng.Intn(int(variance)*2) - int(variance)
	
	clamp := func(v int) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}
	
	return color.RGBA{clamp(r), clamp(g), clamp(b), 255}
}


func generateDynamicPalette(rng *rand.Rand, cycle float64, mode int) ColorPalette {
	var p ColorPalette
	p.cycle = cycle
	
	p.dead = color.RGBA{0, 0, 0, 255}
	
	// Different palette modes
	var youngBase, matureBase, oldBase struct{ r, g, b uint8 }
	
	switch mode {
	case 0: // Rainbow Mode
		youngBase = struct{ r, g, b uint8 }{
			uint8(127 + 127*math.Sin(cycle)),
			uint8(127 + 127*math.Sin(cycle+2*math.Pi/3)),
			uint8(127 + 127*math.Sin(cycle+4*math.Pi/3)),
		}
		matureBase = struct{ r, g, b uint8 }{
			uint8(200 + 55*math.Cos(cycle)),
			uint8(150 + 105*math.Sin(cycle)),
			uint8(100 + 155*math.Cos(cycle+math.Pi/2)),
		}
		oldBase = struct{ r, g, b uint8 }{
			uint8(255 - 55*math.Sin(cycle)),
			uint8(100 + 100*math.Cos(cycle)),
			uint8(150 + 105*math.Sin(cycle+math.Pi)),
		}
	case 1: // Ocean Mode
		youngBase = struct{ r, g, b uint8 }{0, uint8(150 + 50*math.Sin(cycle)), uint8(200 + 55*math.Cos(cycle))}
		matureBase = struct{ r, g, b uint8 }{0, uint8(180 + 75*math.Sin(cycle)), uint8(150 + 50*math.Cos(cycle))}
		oldBase = struct{ r, g, b uint8 }{uint8(50 + 50*math.Sin(cycle)), uint8(100 + 100*math.Cos(cycle)), 200}
	case 2: // Fire Mode
		youngBase = struct{ r, g, b uint8 }{uint8(200 + 55*math.Sin(cycle)), uint8(100 + 50*math.Cos(cycle)), 0}
		matureBase = struct{ r, g, b uint8 }{uint8(255 - 55*math.Cos(cycle)), uint8(150 + 50*math.Sin(cycle)), 0}
		oldBase = struct{ r, g, b uint8 }{255, uint8(50 + 100*math.Sin(cycle)), uint8(50 + 100*math.Cos(cycle))}
	default: // Original mode
		youngBase = struct{ r, g, b uint8 }{0, 200, 0}
		matureBase = struct{ r, g, b uint8 }{200, 200, 0}
		oldBase = struct{ r, g, b uint8 }{255, 0, 0}
	}
	
	for i := range p.young {
		intensity := float32(0.5 + float32(i)*0.1)
		r := uint8(float32(youngBase.r) * intensity)
		g := uint8(float32(youngBase.g) * intensity)
		b := uint8(float32(youngBase.b) * intensity)
		p.young[i] = randomColor(rng, r, g, b, 30)
	}
	
	for i := range p.mature {
		factor := float32(i) / float32(len(p.mature))
		r := uint8(float32(matureBase.r) * (0.7 + factor*0.3))
		g := uint8(float32(matureBase.g) * (1.0 - factor*0.5))
		b := uint8(float32(matureBase.b) * (0.5 + factor*0.5))
		p.mature[i] = randomColor(rng, r, g, b, 25)
	}
	
	for i := range p.old {
		factor := 1.0 - float32(i)/float32(len(p.old))*0.6
		r := uint8(float32(oldBase.r) * factor)
		g := uint8(float32(oldBase.g) * factor)
		b := uint8(float32(oldBase.b) * factor)
		p.old[i] = randomColor(rng, r, g, b, 20)
	}
	
	return p
}

func calculateStats(grid [][]Cell, generation int, gridSize int) Stats {
	var s Stats
	s.generation = generation
	totalCells := 0
	totalAge := 0
	
	// Initialize age histogram
	for i := range s.ageHistogram {
		s.ageHistogram[i] = 0
	}
	
	for y := range grid {
		for x := range grid[y] {
			val := grid[y][x].val
			if val > 0 {
				totalCells++
				totalAge += val
				idx := val - 1
				if idx >= len(s.ageHistogram) {
					idx = len(s.ageHistogram) - 1
				}
				s.ageHistogram[idx]++
			}
		}
	}
	
	s.population = totalCells
	s.density = float64(totalCells) / float64(gridSize*gridSize)
	
	if totalCells > 0 {
		s.avgAge = float64(totalAge) / float64(totalCells)
	}
	
	// Entropy calculation
	totalSize := float64(gridSize * gridSize)
	if s.population > 0 {
		p := float64(s.population) / totalSize
		if p > 0 && p < 1 {
			s.entropy = -p*math.Log2(p) - (1-p)*math.Log2(1-p)
		}
	}
	
	return s
}

func addEvent(state *SimulationState, eventType, message string) {
	event := Event{
		generation: state.stats.generation,
		eventType:  eventType,
		message:    message,
	}
	state.events = append(state.events, event)
	if len(state.events) > 10 {
		state.events = state.events[1:]
	}
}

func applyBloom(img *image.RGBA, intensity float64) {
	bounds := img.Bounds()
	tempImg := image.NewRGBA(bounds)
	
	// Copy the image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			tempImg.Set(x, y, img.At(x, y))
		}
	}
	
	// Apply simple blur for bloom effect
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			r, g, b, a := tempImg.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				// Add neighboring pixels with attenuation
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dx == 0 && dy == 0 {
							continue
						}
						nr, ng, nb, _ := img.At(x+dx, y+dy).RGBA()
						r += uint32(float64(nr) * intensity * 0.05)
						g += uint32(float64(ng) * intensity * 0.05)
						b += uint32(float64(nb) * intensity * 0.05)
					}
				}
				// Clamp
				if r > 65535 {
					r = 65535
				}
				if g > 65535 {
					g = 65535
				}
				if b > 65535 {
					b = 65535
				}
				img.Set(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
			}
		}
	}
}

func main() {
	a := app.New()
	w := a.NewWindow("Living Numbers Game - Experimental Laboratory")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	state := &SimulationState{
		growthRate:     0.05,
		mutationChance: 0.01,
		paletteMode:    0,
		bloomEffect:    true,
		events:         make([]Event, 0),
		isPaused:       false,
		isStarted:      false,
		cellSize:       5,
		gridSize:       displaySize / 5,
		speed:          50,
	}
	
	palette := generateDynamicPalette(rng, 0, state.paletteMode)

	grid := make([][]Cell, state.gridSize)
	for i := range grid {
		grid[i] = make([]Cell, state.gridSize)
	}

	// Empty grid at startup - cells appear on Start click
	// (no initialization here)

	img := image.NewRGBA(image.Rect(0, 0, displaySize, displaySize))
	drawGridDynamic(grid, img, palette, state.cellSize, state.gridSize)
	
	canvasImg := canvas.NewImageFromImage(img)
	canvasImg.FillMode = canvas.ImageFillOriginal
	canvasImg.SetMinSize(fyne.NewSize(float32(displaySize), float32(displaySize)))

	// Control interface
	statusLabel := widget.NewLabel("Empty grid - Press Start to begin")
	
	growthLabel := widget.NewLabel(fmt.Sprintf("Growth rate: %.2f", state.growthRate))
	growthSlider := widget.NewSlider(0.05, 0.5)
	growthSlider.Step = 0.01
	growthSlider.Value = state.growthRate
	growthSlider.OnChanged = func(v float64) {
		state.growthRate = v
		growthLabel.SetText(fmt.Sprintf("Growth rate: %.2f", v))
	}
	
	mutationLabel := widget.NewLabel(fmt.Sprintf("Mutation: %.3f", state.mutationChance))
	mutationSlider := widget.NewSlider(0, 0.1)
	mutationSlider.Step = 0.001
	mutationSlider.Value = state.mutationChance
	mutationSlider.OnChanged = func(v float64) {
		state.mutationChance = v
		mutationLabel.SetText(fmt.Sprintf("Mutation: %.3f", v))
	}
	
	maxPop := state.gridSize * state.gridSize
	pixelLabel := widget.NewLabel(fmt.Sprintf("Pixel size: %dpx (Max pop: %d)", state.cellSize, maxPop))
	pixelSlider := widget.NewSlider(2, 8)
	pixelSlider.Step = 1
	pixelSlider.Value = float64(state.cellSize)
	
	// Callback for pixel slider - recreates grid and image
	pixelSlider.OnChanged = func(v float64) {
		oldCellSize := state.cellSize
		state.cellSize = int(v)
		state.gridSize = displaySize / state.cellSize
		maxPop := state.gridSize * state.gridSize
		pixelLabel.SetText(fmt.Sprintf("Pixel size: %dpx (Max pop: %d)", state.cellSize, maxPop))
		
		// Recreate grid with new size
		grid = make([][]Cell, state.gridSize)
		for i := range grid {
			grid[i] = make([]Cell, state.gridSize)
		}
		
		// Recreate image
		img = image.NewRGBA(image.Rect(0, 0, displaySize, displaySize))
		drawGridDynamic(grid, img, palette, state.cellSize, state.gridSize)
		canvasImg.Image = img
		canvasImg.Refresh()
		
		// Log event if significant change
		if oldCellSize != state.cellSize {
			addEvent(state, "CONFIG", fmt.Sprintf("Grid resized: %dx%d cells (%d max)", state.gridSize, state.gridSize, maxPop))
		}
	}
	
	speedLabel := widget.NewLabel(fmt.Sprintf("Speed: %dms/gen", state.speed))
	speedSlider := widget.NewSlider(10, 200)
	speedSlider.Step = 10
	speedSlider.Value = float64(state.speed)
	speedSlider.OnChanged = func(v float64) {
		state.speed = int(v)
		speedLabel.SetText(fmt.Sprintf("Speed: %dms/gen", state.speed))
	}

	// Interactive color legend - BEFORE paletteSelect
	legendLabel := widget.NewLabel("🎨 Legend:")
	
	// Create smaller color squares
	deadRect := canvas.NewRectangle(palette.dead)
	deadRect.SetMinSize(fyne.NewSize(12, 12))
	youngRect := canvas.NewRectangle(palette.young[2])
	youngRect.SetMinSize(fyne.NewSize(12, 12))
	matureRect := canvas.NewRectangle(palette.mature[7])
	matureRect.SetMinSize(fyne.NewSize(12, 12))
	oldRect := canvas.NewRectangle(palette.old[15])
	oldRect.SetMinSize(fyne.NewSize(12, 12))
	
	// Compact meaning labels
	deadLabel := widget.NewLabel("Dead (0)")
	youngLabel := widget.NewLabel("Young (1-4)")
	matureLabel := widget.NewLabel("Mature (5-19)")
	oldLabel := widget.NewLabel("Old (20-49)")
	
	// Organize in lines
	legendRow1 := container.NewHBox(deadRect, deadLabel)
	legendRow2 := container.NewHBox(youngRect, youngLabel)
	legendRow3 := container.NewHBox(matureRect, matureLabel)
	legendRow4 := container.NewHBox(oldRect, oldLabel)
	
	legendBox := container.NewVBox(
		legendRow1,
		legendRow2,
		legendRow3,
		legendRow4,
	)
	
	// Function to update legend colors
	updateLegendColors := func() {
		deadRect.FillColor = palette.dead
		youngRect.FillColor = palette.young[2]
		matureRect.FillColor = palette.mature[7]
		oldRect.FillColor = palette.old[15]
		deadRect.Refresh()
		youngRect.Refresh()
		matureRect.Refresh()
		oldRect.Refresh()
	}
	
	// paletteSelect AFTER updateLegendColors declaration
	paletteSelect := widget.NewSelect([]string{"Original", "Rainbow", "Ocean", "Fire"}, func(s string) {
		switch s {
		case "Rainbow":
			state.paletteMode = 0
		case "Ocean":
			state.paletteMode = 1
		case "Fire":
			state.paletteMode = 2
		default:
			state.paletteMode = 3
		}
		// Update palette and legend
		palette = generateDynamicPalette(rng, 0, state.paletteMode)
		updateLegendColors()
		if !state.isStarted {
			drawGrid(grid, img, palette)
			canvasImg.Refresh()
		}
	})
	paletteSelect.SetSelected("Original")
	
	bloomCheck := widget.NewCheck("Bloom Effect", func(checked bool) {
		state.bloomEffect = checked
	})
	bloomCheck.Checked = true
	
	startButton := widget.NewButton("▶ Start", func() {})
	pauseButton := widget.NewButton("⏸ Pause", func() {})
	pauseButton.Disable()
	
	supernovaButton := widget.NewButton("💥 Supernova", func() {})
	supernovaButton.Disable()
	
	helpButton := widget.NewButton("❓ How it works?", func() {})
	
	statsLabel := widget.NewLabel("Stats: --")
	eventLog := widget.NewLabel("Log: Waiting for start...")
	eventLog.Wrapping = fyne.TextWrapWord
	
	controlsLeft := container.NewVBox(
		widget.NewLabel("🎮 Controls"),
		widget.NewSeparator(),
		growthLabel,
		growthSlider,
		mutationLabel,
		mutationSlider,
		pixelLabel,
		pixelSlider,
		speedLabel,
		speedSlider,
		paletteSelect,
		bloomCheck,
		container.NewGridWithColumns(2, startButton, pauseButton),
		supernovaButton,
		helpButton,
	)
	
	controlsRight := container.NewVBox(
		widget.NewLabel("📊 Statistics"),
		widget.NewSeparator(),
		statsLabel,
		widget.NewSeparator(),
		widget.NewLabel("📜 Event Log"),
		eventLog,
		widget.NewSeparator(),
		legendLabel,
		legendBox,
	)
	

	controls := container.NewGridWithColumns(2, controlsLeft, controlsRight)
	
	mainContainer := container.NewBorder(
		nil,
		container.NewVBox(statusLabel, controls),
		nil,
		nil,
		canvasImg,
	)

	w.SetContent(mainContainer)
	w.Resize(fyne.NewSize(float32(displaySize), float32(displaySize+280)))
	w.CenterOnScreen()
	// Allow free window resizing

	driver := a.Driver()
	
	// Help button - Display explanation
	helpButton.OnTapped = func() {
		helpText := `
LIVING NUMBERS GAME - Quick Guide

WHAT IS IT?
A simulation where cells are born, grow old, and compete for space. Each cell has an "age" from 1 to 50, represented by different colors.

WHAT HAPPENS AT START?

* Black screen = Empty grid (all cells are dead)
* Press Start -> Random cells appear (200-600 cells)
* Each new simulation has different starting positions
* This randomness creates unique evolution patterns

HOW DOES IT EVOLVE?

Every generation (20x per second), each cell follows simple rules:

1. BIRTH (empty cell becomes alive):
   - If neighbors are present, chance of birth
   - Growth Rate controls this probability
   - Higher growth = faster colonization

2. SURVIVAL (cell stays alive):
   - Needs at least 3 neighbors
   - Too isolated (<3) -> Dies (loneliness)
   - Survives in community

3. AGING (cell gets older):
   - When crowded (>20 neighbors) -> Age +1
   - Color changes: Green->Yellow->Red
   - At age 50 -> Resets to age 1 (rejuvenation)

4. MUTATIONS (random changes):
   - Mutation slider = chaos level
   - Randomly changes some cell ages
   - Creates unpredictable patterns

CONTROLS

* Growth Rate (0.05-0.5): Birth probability
  -> 0.05: Sparse, slow growth
  -> 0.30: Dense, rapid colonization

* Mutation (0-0.1): Random variations
  -> 0: Predictable, stable
  -> 0.05: Chaotic, surprising

* Color Palettes: Visual themes
  -> Change anytime to see patterns differently

* Supernova: Local extinction
  -> Creates "hole" in population
  -> Watch how life recovers

STATISTICS

* Population: Living cells count
* Density: % of grid occupied
* Avg Age: Population maturity
* Entropy: System disorder (0=order, 1=chaos)

SIMPLE EXPERIMENTS

1. Slow & Stable (growth=0.15, mutation=0)
   -> Observe gradual, predictable spread

2. Fast & Chaotic (growth=0.30, mutation=0.05)
   -> Watch explosive, random evolution

3. Recovery Test: Start -> Wait 50% -> Supernova
   -> See how fast population recovers

CELL COLORS

Dead (Black) -> Young (Green) -> Mature (Yellow) -> Old (Red)

The magic: Simple rules create complex, beautiful patterns!
Each simulation is unique due to random start.

Press Start to begin your experiment!
	`

		helpLabel := widget.NewLabel(helpText)
		helpLabel.Wrapping = fyne.TextWrapWord
		
		scrollHelp := container.NewScroll(helpLabel)
		scrollHelp.SetMinSize(fyne.NewSize(600, 400))
		
		d := dialog.NewCustom("How it works?", "Close", scrollHelp, w)
		d.Show()
	}

	// Function to reset grid
	resetGrid := func() {
		// Recreate grid with new size
		grid = make([][]Cell, state.gridSize)
		for i := range grid {
			grid[i] = make([]Cell, state.gridSize)
		}
		
		// Recreate image with new size
		img = image.NewRGBA(image.Rect(0, 0, displaySize, displaySize))
		
		// Add new cells
		newInitCount := 200 + rng.Intn(400)
		for i := 0; i < newInitCount; i++ {
			x := rng.Intn(state.gridSize)
			y := rng.Intn(state.gridSize)
			grid[y][x].val = rng.Intn(10) + 1
		}
		
		// Redraw grid
		palette = generateDynamicPalette(rng, 0, state.paletteMode)
		updateLegendColors()
		drawGridDynamic(grid, img, palette, state.cellSize, state.gridSize)
		canvasImg.Image = img
		canvasImg.Refresh()
	}

	startButton.OnTapped = func() {
		if !state.isStarted {
			// Reset grid with new parameters
			resetGrid()
			
			state.isStarted = true
			state.isPaused = false
			startButton.SetText("⏹ Stop")
			pauseButton.Enable()
			supernovaButton.Enable()
			
			// Lock controls during simulation
			growthSlider.Disable()
			mutationSlider.Disable()
			pixelSlider.Disable()
			speedSlider.Disable()
			paletteSelect.Disable()
			
			addEvent(state, "START", fmt.Sprintf("Simulation started (growth=%.2f, mutation=%.3f)", state.growthRate, state.mutationChance))
			eventLog.SetText("Simulation running...")
		} else {
			state.isStarted = false
			state.isPaused = false
			startButton.SetText("▶ Start")
			pauseButton.SetText("Pause")
			pauseButton.Disable()
			supernovaButton.Disable()
			
			// Unlock controls
			growthSlider.Enable()
			mutationSlider.Enable()
			pixelSlider.Enable()
			speedSlider.Enable()
			paletteSelect.Enable()
			
			addEvent(state, "STOP", "Simulation stopped")
		}
	}
	
	pauseButton.OnTapped = func() {
		if !state.isStarted {
			return
		}
		state.isPaused = !state.isPaused
		if state.isPaused {
			pauseButton.SetText("▶ Resume")
			addEvent(state, "PAUSE", "Simulation paused")
		} else {
			pauseButton.SetText("Pause")
			addEvent(state, "RESUME", "Simulation resumed")
		}
	}
	
	supernovaButton.OnTapped = func() {
		if !state.isStarted {
			return
		}
		// Supernova: reset random area
		centerX := rng.Intn(state.gridSize)
		centerY := rng.Intn(state.gridSize)
		radius := 10 + rng.Intn(15)
		
		for y := 0; y < state.gridSize; y++ {
			for x := 0; x < state.gridSize; x++ {
				dx := x - centerX
				dy := y - centerY
				if dx*dx+dy*dy < radius*radius {
					grid[y][x].val = 0
				}
			}
		}
		addEvent(state, "SUPERNOVA", fmt.Sprintf("Explosion at (%d,%d) radius %d", centerX, centerY, radius))
	}

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		generation := 0
		cycle := 0.0
		frameCounter := 0

		for range ticker.C {
			if !state.isStarted || state.isPaused {
				continue
			}
			
			// Speed control via counter
			frameCounter++
			if frameCounter < state.speed/10 {
				continue
			}
			frameCounter = 0
			
			generation++
			cycle += 0.05
			
			totalCells := state.gridSize * state.gridSize
			
			// Random events
			if rng.Float64() < state.mutationChance {
				// Genetic mutation
				for i := 0; i < 5+rng.Intn(10); i++ {
					x := rng.Intn(state.gridSize)
					y := rng.Intn(state.gridSize)
					if grid[y][x].val > 0 {
						grid[y][x].val = 1 + rng.Intn(20)
					}
				}
				addEvent(state, "MUTATION", "Genetic mutations detected")
			}
			
			evolve(grid, rng, state.growthRate)
			
			// Calculate stats
			state.stats = calculateStats(grid, generation, state.gridSize)
			
			// Dynamic palette based on average age
			palette = generateDynamicPalette(rng, cycle+state.stats.avgAge*0.1, state.paletteMode)
			
			drawGridDynamic(grid, img, palette, state.cellSize, state.gridSize)
			
			// Bloom effect
			if state.bloomEffect {
				applyBloom(img, 0.3)
			}

			if state.stats.population >= totalCells {
				finalMessage := fmt.Sprintf("COMPLETED - Generation %d - Grid filled!", generation)
				addEvent(state, "END", "Maximum population reached")
				state.isStarted = false
				generation = 0
				runOnMain(driver, func() {
					statusLabel.SetText(finalMessage)
					startButton.SetText("▶ Start")
					pauseButton.Disable()
					supernovaButton.Disable()
					growthSlider.Enable()
					mutationSlider.Enable()
					pixelSlider.Enable()
					speedSlider.Enable()
					paletteSelect.Enable()
					canvasImg.Refresh()
				})
				continue
			}
			
			// Detection of remarkable events
			if state.stats.density > 0.9 && generation%50 == 0 {
				addEvent(state, "DENSITY", fmt.Sprintf("Critical density: %.1f%%", state.stats.density*100))
			}

			runningMessage := fmt.Sprintf("Gen %d - Pop %d/%d (%.1f%%) - Avg age: %.1f - Entropy: %.3f",
				generation, state.stats.population, totalCells, state.stats.density*100, state.stats.avgAge, state.stats.entropy)
			
			statsText := fmt.Sprintf("Population: %d\nDensity: %.1f%%\nAvg age: %.1f\nEntropy: %.3f",
				state.stats.population, state.stats.density*100, state.stats.avgAge, state.stats.entropy)
			
			eventText := ""
			for i := len(state.events) - 1; i >= 0 && i >= len(state.events)-3; i-- {
				e := state.events[i]
				eventText += fmt.Sprintf("[Gen %d] %s: %s\n", e.generation, e.eventType, e.message)
			}
			
			runOnMain(driver, func() {
				statusLabel.SetText(runningMessage)
				statsLabel.SetText(statsText)
				eventLog.SetText(eventText)
				canvasImg.Refresh()
			})
		}
	}()

	w.ShowAndRun()
}

func drawGridDynamic(grid [][]Cell, img *image.RGBA, palette ColorPalette, cellSize int, gridSize int) {
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			c := getCellColor(grid[y][x].val, palette)
			for dy := 0; dy < cellSize; dy++ {
				for dx := 0; dx < cellSize; dx++ {
					img.Set(x*cellSize+dx, y*cellSize+dy, c)
				}
			}
		}
	}
}

func drawGrid(grid [][]Cell, img *image.RGBA, palette ColorPalette) {
	drawGridDynamic(grid, img, palette, currentCellSize, currentGridSize)
}

func getCellColor(val int, palette ColorPalette) color.Color {
	if val == 0 {
		return palette.dead
	} else if val < 5 {
		return palette.young[val-1]
	} else if val < 20 {
		return palette.mature[val-5]
	} else {
		idx := val - 20
		if idx >= len(palette.old) {
			idx = len(palette.old) - 1
		}
		return palette.old[idx]
	}
}

func evolve(g [][]Cell, rng *rand.Rand, growthRate float64) {
	h := len(g)
	w := len(g[0])
	newGrid := make([][]Cell, h)
	for y := range newGrid {
		newGrid[y] = make([]Cell, w)
		for x := range newGrid[y] {
			sum := neighbors(g, x, y)
			val := g[y][x].val
			if val == 0 && rng.Float64() < growthRate*(float64(sum)/50) {
				val = 1
			} else if val > 0 {
				if sum < 3 {
					val = 0
				} else if sum > 20 {
					val++
					if val > 50 {
						val = 1
					}
				}
			}
			newGrid[y][x].val = val
		}
	}
	for y := range g {
		copy(g[y], newGrid[y])
	}
}

func neighbors(g [][]Cell, x, y int) int {
	h := len(g)
	w := len(g[0])
	sum := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			ny := y + dy
			nx := x + dx
			if nx >= 0 && ny >= 0 && nx < w && ny < h {
				sum += g[ny][nx].val
			}
		}
	}
	return sum
}

# Living Numbers Game - Experimental Laboratory

![Demo](show.gif)

An interactive cellular automaton simulation inspired by Conway's Game of Life, enhanced with aging mechanics, dynamic mutations, and real-time statistical analysis.

## 🎯 Purpose

This application explores **emergent complexity** in artificial life systems by simulating cellular populations that grow, age, mutate, and interact according to simple local rules that generate complex global patterns.

### Scientific Questions Explored

- **Emergence**: How do simple local rules generate complex global patterns?
- **Parameter influence**: What balance between order and chaos produces the richest patterns?
- **System dynamics**: How do populations evolve from colonization to saturation?
- **Resilience**: How do systems recover from catastrophic events?

## 🚀 Quick Start

### Build & Run

```bash
go build -o living_numbers main.go
./living_numbers
```

### Requirements

- Go 1.16+
- Fyne v2 GUI library (automatically fetched via go.mod)

## 🎮 Controls

### Before Starting
- **Growth Rate slider** (0.05-0.5): Controls colonization speed
- **Mutation slider** (0-0.1): Introduces random genetic variations
- **Palette selector**: Choose visual color scheme (Original, Rainbow, Ocean, Fire)
- **Bloom Effect**: Toggle glow effect for enhanced visuals

### During Simulation
- **▶ Start / ⏹ Stop**: Launch or halt the simulation
- **⏸ Pause / ▶ Resume**: Freeze/unfreeze the simulation
- **💥 Supernova**: Trigger catastrophic local extinction event

## 📊 Real-Time Statistics

- **Population**: Number of living cells
- **Density**: Space occupation rate (%)
- **Average Age**: Population maturity indicator
- **Entropy**: System disorder measurement (0-1)
- **Event Log**: Last 3 significant events

## 🔬 Simulation Mechanics

### Cell Life Cycle

1. **Birth** (value = 1): Occurs when neighbor density exceeds growth threshold
2. **Aging** (1 → 50): Cells age each generation
3. **Death**: Occurs when neighbor density < 3 (isolation)
4. **Rejuvenation**: At age 50, cells restart at age 1

### Color Coding

- **Green/Cyan**: Young cells (1-4) - Pioneer colonizers
- **Yellow/Orange**: Mature cells (5-19) - Stable population
- **Red/Purple**: Old cells (20-49) - Ancient biomass

### Rules (per generation)

```
For each cell:
  neighbor_sum = sum of all 8 neighbor ages
  
  If cell is dead (0):
    Birth probability = growth_rate * (neighbor_sum / 50)
  
  If cell is alive (>0):
    If neighbor_sum < 3: Die (isolation)
    If neighbor_sum > 20: Age +1
    
  Random mutations occur with mutation_chance probability
```

## 🧪 Suggested Experiments

### Experiment 1: Percolation Threshold
**Question**: What is the minimum growth rate for complete grid filling?

1. Set mutation to 0.000
2. Start with growth rate 0.05
3. Increase incrementally until grid fills
4. **Hypothesis**: Critical threshold exists around 0.15-0.20

### Experiment 2: Mutation Impact
**Question**: Do mutations accelerate or slow colonization?

1. Run simulation with mutation = 0.000
2. Run simulation with mutation = 0.050
3. Compare time to 90% density
4. **Hypothesis**: Low mutations (<0.02) accelerate, high mutations slow down

### Experiment 3: Catastrophe Recovery
**Question**: How resilient is the system to catastrophes?

1. Start simulation until 50% density
2. Trigger Supernova
3. Measure recovery time to 50% density
4. **Hypothesis**: Recovery time proportional to growth rate

### Experiment 4: Visual Pattern Recognition
**Question**: Do different palettes reveal hidden structures?

1. Run same seed with different palettes
2. Compare visible spatial structures
3. **Observation**: Rainbow mode reveals age gradients, Ocean shows density waves

## 🎨 Visual Features

- **Dynamic Palettes**: 4 color modes with trigonometric cycling
- **Bloom Effect**: Post-processing glow based on cell density
- **Age-based Coloring**: Visual distinction of cell ages (young/mature/old)
- **Real-time Updates**: 20 FPS rendering (50ms per generation)

## 📐 Technical Details

### Architecture

```
Grid (60×60 cells) → Evolution Engine → Statistics Calculator
                              ↓
                    Visual Renderer (480×480px)
                              ↓
                    Dynamic Palette Generator
                              ↓
                    Optional Bloom Effect
```

### Key Functions

- [`evolve()`](main.go:669): Core cellular automaton logic
- [`calculateStats()`](main.go:224): Population metrics computation
- [`generateDynamicPalette()`](main.go:157): Animated color schemes
- [`applyBloom()`](main.go:280): Visual post-processing effect

### Performance

- Grid: 3,600 cells (60×60)
- Rendering: 230,400 pixels (480×480)
- Update rate: 20 generations/second
- Typical run: 500-2000 generations to completion

## 🌍 Biological/Ecological Analogies

| Simulation Element | Real-World Analogy |
|-------------------|-------------------|
| Young cells | Pioneer species |
| Mature cells | Climax community |
| Old cells | Senescent biomass |
| Mutations | Genetic variations |
| Supernova | Forest fire, meteor impact |
| Growth rate | Reproductive rate |
| Density | Carrying capacity |

## 📚 Educational Value

This laboratory demonstrates:

- **Complex systems theory**: Emergence from simple rules
- **Self-organization**: Pattern formation without central planning
- **Phase transitions**: Empty → Colonization → Saturation
- **Adaptive systems**: Response to perturbations
- **Quantitative analysis**: Measuring visual phenomena

## 🤝 Contributing

This is an experimental research tool. Modifications welcome for:
- New evolution rules
- Additional statistical metrics
- Alternative visualization modes
- Multi-grid comparisons

## 🛠️ Development

This project was developed with the assistance of **Kilo Code** and **Claude Sonnet 4.5** to learn and master:
- **Go programming**: Language fundamentals, compilation, and project structure
- **GUI development**: Fyne framework for cross-platform graphical interfaces
- **Mathematical modeling**: Cellular automaton algorithms and statistical analysis
- **Real-time rendering**: Dynamic visual generation and post-processing effects

## License

Open source - Educational and research purposes

## 🔗 References

- Conway's Game of Life (1970)
- Cellular Automata Theory (Wolfram)
- Complex Adaptive Systems (Holland)
- Artificial Life (Langton)
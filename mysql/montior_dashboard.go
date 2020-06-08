package mysql

import (
	"context"
	"fmt"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/container/grid"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/termbox"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/barchart"
	"github.com/mum4k/termdash/widgets/button"
	"github.com/mum4k/termdash/widgets/gauge"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/mum4k/termdash/widgets/sparkline"
	"github.com/mum4k/termdash/widgets/text"
	"math"
	"math/rand"
	"sync"
	"time"
)

const redrawInterval = 250 * time.Millisecond

type widgets struct {
	connectionGauge      *gauge.Gauge
	concurrenceGauge     *gauge.Gauge
	cacheHitRateGauge    *gauge.Gauge
	diskUsageRateGauge   *gauge.Gauge
	memoryUsageRateGauge *gauge.Gauge
	QPSText              *text.Text
	TPSText              *text.Text
	IOText               *text.Text
	leftB                *button.Button
	rightB               *button.Button
	sineLC               *linechart.LineChart
}

type monitorDashboard struct {
	monitor *monitor
	data    *monitorData
	widgets *widgets
}

func newMonitorDashboard(m *monitor) *monitorDashboard {
	return &monitorDashboard{
		monitor: m,
	}
}

func (p *monitorDashboard) newWidgets(ctx context.Context, c *container.Container) (*widgets, error) {

	connectionGauge, err := p.newConnectionGauge(ctx)
	if err != nil {
		return nil, err
	}

	concurrenceGauge, err := p.newConcurrenceGauge(ctx)
	if err != nil {
		return nil, err
	}

	cacheHitRateGauge, err := p.newCacheHitRateGauge(ctx)
	if err != nil {
		return nil, err
	}
	IOText, err := p.newIOText(ctx)
	if err != nil {
		return nil, err
	}

	QPSText, err := p.newQPSText(ctx)
	if err != nil {
		return nil, err
	}

	TPSText, err := p.newTPSText(ctx)
	if err != nil {
		return nil, err
	}

	leftB, rightB, sineLC, err := p.newSines(ctx)
	if err != nil {
		return nil, err
	}
	return &widgets{
		connectionGauge:   connectionGauge,
		concurrenceGauge:  concurrenceGauge,
		cacheHitRateGauge: cacheHitRateGauge,
		IOText:            IOText,
		QPSText:           QPSText,
		TPSText:           TPSText,
		leftB:             leftB,
		rightB:            rightB,
		sineLC:            sineLC,
	}, nil
}

func (p *monitorDashboard) refreshData() {
	p.monitor.setMonitorData()
	p.data = p.monitor.getMonitorData()
	p.refreshConnectionGauge()
	p.refreshConcurrenceGauge()
	p.refreshCacheHitRateGauge()
	p.refreshTPSText()
	p.refreshQPSText()
	p.refreshIOText()
}

// gridLayout prepares container options that represent the desired screen layout.
// This function demonstrates the use of the grid builder.
// gridLayout() and contLayout() demonstrate the two available layout APIs and
// both produce equivalent layouts for layoutType layoutAll.
func (p *monitorDashboard) gridLayout(w *widgets) ([]container.Option, error) {

	borderless, err := text.New()
	if err != nil {
		panic(err)
	}
	if err := borderless.Write("localhost_transaction(MySQL 5.6)"); err != nil {
		panic(err)
	}

	leftRows := []grid.Element{
		grid.RowHeightPerc(7,
			grid.Widget(borderless,
				container.Border(linestyle.Light),
			),
		),
		grid.RowHeightPerc(8,
			grid.Widget(w.connectionGauge,
				container.Border(linestyle.Light),
				container.BorderTitle("连接数"),
				container.BorderColor(cell.ColorNumber(39)),
			),
		),
		grid.RowHeightPerc(7,
			grid.Widget(w.concurrenceGauge,
				container.Border(linestyle.Light),
				container.BorderTitle("并发数"),
				container.BorderColor(cell.ColorNumber(39)),
			),
		),
		grid.RowHeightPerc(8,
			grid.Widget(w.TPSText,
				container.Border(linestyle.Light),
				container.BorderTitle("TPS"),
				container.BorderColor(cell.ColorNumber(39)),
			),
		),
		grid.RowHeightPerc(7,
			grid.Widget(w.QPSText,
				container.Border(linestyle.Light),
				container.BorderTitle("QPS"),
				container.BorderColor(cell.ColorNumber(39)),
			),
		),
		grid.RowHeightPerc(7,
			grid.Widget(w.cacheHitRateGauge,
				container.Border(linestyle.Light),
				container.BorderTitle("缓存命中率"),
				container.BorderColor(cell.ColorNumber(39)),
			),
		),
		grid.RowHeightPerc(7,
			grid.Widget(w.IOText,
				container.Border(linestyle.Light),
				container.BorderTitle("I/O Bytes"),
				container.BorderColor(cell.ColorNumber(39)),
			),
		),
		//grid.RowHeightPerc(7,
		//	grid.Widget(w.connectionGauge,
		//		container.Border(linestyle.Light),
		//		container.BorderTitle("慢查询计数"),
		//		container.BorderColor(cell.ColorNumber(39)),
		//	),
		//),
		//grid.RowHeightPerc(7,
		//	grid.Widget(w.connectionGauge,
		//		container.Border(linestyle.Light),
		//		container.BorderTitle("阻塞查询计数"),
		//		container.BorderColor(cell.ColorNumber(39)),
		//	),
		//),
		//grid.RowHeightPerc(7,
		//	grid.Widget(w.connectionGauge,
		//		container.Border(linestyle.Light),
		//		container.BorderTitle("死锁计数"),
		//		container.BorderColor(cell.ColorNumber(39)),
		//	),
		//),
		grid.RowHeightPerc(1),
	}

	builder := grid.New()
	builder.Add(
		grid.ColWidthPerc(20, leftRows...),
	)

	builder.Add(
		grid.ColWidthPerc(80,
			grid.RowHeightPerc(33,
				grid.Widget(w.sineLC,
					container.Border(linestyle.Light),
					container.BorderTitle("连接数/并发数"),
				),
			),
			grid.RowHeightPerc(33,
				grid.Widget(w.sineLC,
					container.Border(linestyle.Light),
					container.BorderTitle("TPS/QPS"),
				),
			),
			grid.RowHeightPerc(33,
				grid.Widget(w.sineLC,
					container.Border(linestyle.Light),
					container.BorderTitle("缓存命中率"),
				),
			),
		),
	)

	gridOpts, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return gridOpts, nil
}

// rootID is the ID assigned to the root container.
const rootID = "root"

func (p *monitorDashboard) run() {

	t, err := termbox.New(termbox.ColorMode(terminalapi.ColorMode256))
	if err != nil {
		panic(err)
	}
	defer t.Close()

	c, err := container.New(t, container.ID(rootID))
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	w, err := p.newWidgets(ctx, c)
	if err != nil {
		panic(err)
	}

	p.widgets = w

	p.refreshData()

	// 轮询获取统计数据
	go p.periodic(ctx, 800*time.Millisecond, func() error {
		p.refreshData()
		return nil
	})

	gridOpts, err := p.gridLayout(w) // equivalent to contLayout(w)
	if err != nil {
		panic(err)
	}

	if err := c.Update(rootID, gridOpts...); err != nil {
		panic(err)
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == keyboard.KeyEsc || k.Key == keyboard.KeyCtrlC {
			cancel()
		}
	}
	if err := termdash.Run(ctx, t, c, termdash.KeyboardSubscriber(quitter), termdash.RedrawInterval(redrawInterval)); err != nil {
		panic(err)
	}
}

// periodic executes the provided closure periodically every interval.
// Exits when the context expires.
func (p *monitorDashboard) periodic(ctx context.Context, interval time.Duration, fn func() error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := fn(); err != nil {
				panic(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// textState creates a rotated state for the text we are displaying.
func (p *monitorDashboard) textState(text string, capacity, step int) []rune {
	if capacity == 0 {
		return nil
	}

	var state []rune
	for i := 0; i < capacity; i++ {
		state = append(state, ' ')
	}
	state = append(state, []rune(text)...)
	step = step % len(state)
	return p.rotateRunes(state, step)
}

// newSparkLines creates two new sparklines displaying random values.
func (p *monitorDashboard) newSparkLines(ctx context.Context) (*sparkline.SparkLine, *sparkline.SparkLine, error) {
	spGreen, err := sparkline.New(
		sparkline.Color(cell.ColorGreen),
	)
	if err != nil {
		return nil, nil, err
	}

	const max = 100
	go p.periodic(ctx, 250*time.Millisecond, func() error {
		v := int(rand.Int31n(max + 1))
		return spGreen.Add([]int{v})
	})

	spRed, err := sparkline.New(
		sparkline.Color(cell.ColorRed),
	)
	if err != nil {
		return nil, nil, err
	}
	go p.periodic(ctx, 500*time.Millisecond, func() error {
		v := int(rand.Int31n(max + 1))
		return spRed.Add([]int{v})
	})
	return spGreen, spRed, nil

}

func (p *monitorDashboard) newConnectionGauge(ctx context.Context) (*gauge.Gauge, error) {
	g, err := gauge.New()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (p *monitorDashboard) refreshConnectionGauge() {
	progress := int(float64(p.data.Connections) / float64(p.data.MaxConnections) * 100)
	if progress > 100 {
		progress = 100
	}
	p.widgets.connectionGauge.Percent(
		progress,
		gauge.TextLabel(fmt.Sprintf("%d/%d", p.data.Connections, p.data.MaxConnections,
		)))
}

func (p *monitorDashboard) newConcurrenceGauge(ctx context.Context) (*gauge.Gauge, error) {
	g, err := gauge.New()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (p *monitorDashboard) refreshConcurrenceGauge() {
	progress := int(float64(p.data.Concurrences) / float64(p.data.MaxConnections) * 100)
	if progress > 100 {
		progress = 100
	}
	p.widgets.concurrenceGauge.Percent(
		progress,
		gauge.TextLabel(fmt.Sprintf("%d/%d", p.data.Concurrences, p.data.MaxConnections,
		)))
}

func (p *monitorDashboard) newCacheHitRateGauge(ctx context.Context) (*gauge.Gauge, error) {
	g, err := gauge.New()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (p *monitorDashboard) refreshCacheHitRateGauge() {
	progress := int(p.data.CacheHitRate)
	if progress > 100 {
		progress = 100
	}
	p.widgets.cacheHitRateGauge.Percent(progress)
}

func (p *monitorDashboard) newTPSText(ctx context.Context) (*text.Text, error) {
	g, err := text.New()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (p *monitorDashboard) refreshTPSText() {
	p.widgets.TPSText.Reset()
	p.widgets.TPSText.Write(fmt.Sprintf("%f", p.data.TPS))
}

func (p *monitorDashboard) newQPSText(ctx context.Context) (*text.Text, error) {
	g, err := text.New()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (p *monitorDashboard) refreshQPSText() {
	p.widgets.QPSText.Reset()
	p.widgets.QPSText.Write(fmt.Sprintf("%.0f/s", p.data.QPS))
}

func (p *monitorDashboard) newIOText(ctx context.Context) (*text.Text, error) {
	g, err := text.New()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (p *monitorDashboard) refreshIOText() {
	p.widgets.IOText.Reset()
	p.widgets.IOText.Write(fmt.Sprintf("%.0f/%.0f", p.data.Input, p.data.Output))
}

// newHeartbeat returns a line chart that displays a heartbeat-like progression.
func (p *monitorDashboard) newHeartbeat(ctx context.Context) (*linechart.LineChart, error) {
	var inputs []float64
	for i := 0; i < 100; i++ {
		v := math.Pow(math.Sin(float64(i)), 63) * math.Sin(float64(i)+1.5) * 8
		inputs = append(inputs, v)
	}

	lc, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorGreen)),
	)
	if err != nil {
		return nil, err
	}
	step := 0
	go p.periodic(ctx, redrawInterval/3, func() error {
		step = (step + 1) % len(inputs)
		return lc.Series("heartbeat", p.rotateFloats(inputs, step),
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(87))),
			linechart.SeriesXLabels(map[int]string{
				0: "zero",
			}),
		)
	})
	return lc, nil
}

// newBarChart returns a BarcChart that displays random values on multiple bars.
func (p *monitorDashboard) newBarChart(ctx context.Context) (*barchart.BarChart, error) {
	bc, err := barchart.New(
		barchart.BarColors([]cell.Color{
			cell.ColorNumber(33),
			cell.ColorNumber(39),
			cell.ColorNumber(45),
			cell.ColorNumber(51),
			cell.ColorNumber(81),
			cell.ColorNumber(87),
		}),
		barchart.ValueColors([]cell.Color{
			cell.ColorBlack,
			cell.ColorBlack,
			cell.ColorBlack,
			cell.ColorBlack,
			cell.ColorBlack,
			cell.ColorBlack,
		}),
		barchart.ShowValues(),
	)
	if err != nil {
		return nil, err
	}

	const (
		bars = 6
		max  = 100
	)
	values := make([]int, bars)
	go p.periodic(ctx, 1*time.Second, func() error {
		for i := range values {
			values[i] = int(rand.Int31n(max + 1))
		}

		return bc.Values(values, max)
	})
	return bc, nil
}

// distance is a thread-safe int value used by the newSince method.
// Buttons write it and the line chart reads it.
type distance struct {
	v  int
	mu sync.Mutex
}

// add adds the provided value to the one stored.
func (d *distance) add(v int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.v += v
}

// get returns the current value.
func (d *distance) get() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.v
}

// newSines returns a line chart that displays multiple sine series and two buttons.
// The left button shifts the second series relative to the first series to
// the left and the right button shifts it to the right.
func (p *monitorDashboard) newSines(ctx context.Context) (left, right *button.Button, lc *linechart.LineChart, err error) {
	var inputs []float64
	for i := 0; i < 200; i++ {
		v := math.Sin(float64(i) / 100 * math.Pi)
		inputs = append(inputs, v)
	}

	sineLc, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorGreen)),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	step1 := 0
	secondDist := &distance{v: 100}
	go p.periodic(ctx, redrawInterval/3, func() error {
		step1 = (step1 + 1) % len(inputs)
		if err := lc.Series("first", p.rotateFloats(inputs, step1),
			linechart.SeriesCellOpts(cell.FgColor(cell.ColorBlue)),
		); err != nil {
			return err
		}

		step2 := (step1 + secondDist.get()) % len(inputs)
		return lc.Series("second", p.rotateFloats(inputs, step2), linechart.SeriesCellOpts(cell.FgColor(cell.ColorWhite)))
	})

	// diff is the difference a single button press adds or removes to the
	// second series.
	const diff = 20
	leftB, err := button.New("(l)eft", func() error {
		secondDist.add(diff)
		return nil
	},
		button.GlobalKey('l'),
		button.WidthFor("(r)ight"),
		button.FillColor(cell.ColorNumber(220)),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	rightB, err := button.New("(r)ight", func() error {
		secondDist.add(-diff)
		return nil
	},
		button.GlobalKey('r'),
		button.FillColor(cell.ColorNumber(196)),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return leftB, rightB, sineLc, nil
}

// setLayout sets the specified layout.
func (p *monitorDashboard) setLayout(c *container.Container, w *widgets) error {
	gridOpts, err := p.gridLayout(w)
	if err != nil {
		return err
	}
	return c.Update(rootID, gridOpts...)
}

// rotateFloats returns a new slice with inputs rotated by step.
// I.e. for a step of one:
//   inputs[0] -> inputs[len(inputs)-1]
//   inputs[1] -> inputs[0]
// And so on.
func (p *monitorDashboard) rotateFloats(inputs []float64, step int) []float64 {
	return append(inputs[step:], inputs[:step]...)
}

// rotateRunes returns a new slice with inputs rotated by step.
// I.e. for a step of one:
//   inputs[0] -> inputs[len(inputs)-1]
//   inputs[1] -> inputs[0]
// And so on.
func (p *monitorDashboard) rotateRunes(inputs []rune, step int) []rune {
	return append(inputs[step:], inputs[:step]...)
}

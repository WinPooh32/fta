package main

import (
	"compress/gzip"
	"encoding/csv"
	"image/color"
	"os"
	"time"

	"github.com/pplcc/plotext"
	"github.com/pplcc/plotext/custplotter"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"github.com/WinPooh32/fta"
	"github.com/WinPooh32/series"
)

func main() {
	const period = 20

	ohlcv := readOHLCV("../BTCUSDT.csv.gz")

	// Calculate indicators.
	sma := fta.SMA(ohlcv.Close, period)

	// Fill empty values.
	sma.Pad()

	_, psarBull, psarBear := fta.PSAR(ohlcv.High, ohlcv.Low, ohlcv.Close, 0.02, 0.2)

	// Or just cut off empty parts.
	ohlcv = ohlcv.Slice(period, ohlcv.Len())
	sma = sma.Slice(period, sma.Len())

	psarBull = shrink(psarBull.Slice(period, psarBull.Len()))
	psarBear = shrink(psarBear.Slice(period, psarBear.Len()))

	// Prepare data for rendering.
	upperLine, lowerLine := makePsarScatters(psarBull, psarBear)

	smaLine, err := plotter.NewLine(sma)
	checkErr(err)

	candlesticks, err := custplotter.NewCandlesticks(ohlcv)
	checkErr(err)

	volumeBars, err := custplotter.NewVBars(ohlcv)
	checkErr(err)

	// Prepare plots.
	pricePlot, volumePlot := makePlots()

	pricePlot.Add(
		upperLine, lowerLine,
		smaLine,
		candlesticks,
	)

	// Compensate bars alignment.
	volumePlot.Y.Padding += (candlesticks.CandleWidth - volumeBars.LineStyle.Width) / 2

	volumePlot.Add(volumeBars)

	render("psar.png", pricePlot, volumePlot)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func newReader(file string) *csv.Reader {
	f, err := os.Open(file)
	checkErr(err)

	gzReader, err := gzip.NewReader(f)
	checkErr(err)

	csvReader := csv.NewReader(gzReader)
	csvReader.Comma = ','
	csvReader.ReuseRecord = true

	return csvReader
}

func readOHLCV(file string) fta.OHLCV {
	csvReader := newReader(file)

	ohlcv, err := fta.ReadCSV(csvReader, int64(time.Minute), fta.Seconds)
	checkErr(err)

	ohlcv = ohlcv.Resample(int64(time.Hour))

	return ohlcv
}

func makePlots() (pricePlot, volumePlot *plot.Plot) {
	pricePlot = plot.New()
	pricePlot.Title.Text = "BTCUSDT / PSAR"
	// p.X.Label.Text = "Time"
	pricePlot.Y.Label.Text = "Price"
	pricePlot.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	volumePlot = plot.New()
	volumePlot.X.Label.Text = "Time"
	volumePlot.Y.Label.Text = "Volume"
	volumePlot.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	return pricePlot, volumePlot
}

func makePsarScatters(upper, lower plotter.XYer) (top, bot *plotter.Scatter) {
	top, err := hplot.NewScatter(upper)
	if err != nil {
		panic(err)
	}
	top.GlyphStyle.Shape = draw.PlusGlyph{}
	top.GlyphStyle.Color = color.RGBA{R: 255, G: 0, B: 0, A: 255}

	bot, err = hplot.NewScatter(lower)
	if err != nil {
		panic(err)
	}
	bot.GlyphStyle.Shape = draw.PlusGlyph{}
	bot.GlyphStyle.Color = color.RGBA{R: 0, B: 0, G: 255, A: 255}

	return top, bot
}

func render(file string, pricePlot, volumePlot *plot.Plot) {
	// Make sure that the x axises have the same range anyway
	plotext.UniteAxisRanges([]*plot.Axis{&pricePlot.X, &volumePlot.X})

	// Create a table with one column and two rows
	table := plotext.Table{
		RowHeights: []float64{2, 1}, // 2/3 for candlesticks and 1/3 for volume bars
		ColWidths:  []float64{1},
	}

	plots := [][]*plot.Plot{
		{pricePlot},
		{volumePlot},
	}

	img := vgimg.New(150*4, 100*4)
	dc := draw.New(img)

	canvases := table.Align(plots, dc)
	plots[0][0].Draw(canvases[0][0])
	plots[1][0].Draw(canvases[1][0])

	w, err := os.Create(file)
	checkErr(err)

	png := vgimg.PngCanvas{Canvas: img}

	_, err = png.WriteTo(w)
	checkErr(err)
}

func shrink(data series.Data) series.Data {
	values := data.Values()
	index := data.Index()

	valuesShrinked := make([]fta.DType, 0, len(values))
	indexShrinked := make([]int64, 0, len(index))

	for i, v := range values {
		if v > 0 {
			valuesShrinked = append(valuesShrinked, v)
			indexShrinked = append(indexShrinked, index[i])
		}
	}

	return series.MakeData(data.Freq(), indexShrinked, valuesShrinked)
}

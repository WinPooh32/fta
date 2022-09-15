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
)

func main() {
	const (
		period        = 20
		stdMultiplier = 2.1
	)

	ohlcv := readOHLCV("../BTCUSDT.csv.gz")

	// Calculate indicators.
	sma := fta.SMA(ohlcv.Close, period)

	// Fill empty values.
	sma.Pad()

	bbUpper, bbLower := fta.BBANDS(ohlcv.Close, sma, period, stdMultiplier)

	// Fill empty values.
	// bbUpper.Pad()
	// bbLower.Pad()

	// Or just cut off empty parts.
	ohlcv = ohlcv.Slice(period, ohlcv.Len())
	sma = sma.Slice(period, sma.Len())

	bbUpper = bbUpper.Slice(period, bbUpper.Len())
	bbLower = bbLower.Slice(period, bbLower.Len())

	// Prepare data for rendering.
	upperLine, lowerLine, band := makeBandLines(bbUpper, bbLower)

	smaLine, err := plotter.NewLine(sma)
	checkErr(err)

	candlesticks, err := custplotter.NewCandlesticks(ohlcv)
	checkErr(err)

	volumeBars, err := custplotter.NewVBars(ohlcv)
	checkErr(err)

	// Prepare plots.
	pricePlot, volumePlot := makePlots()

	pricePlot.Add(
		upperLine, lowerLine, band,
		smaLine,
		candlesticks,
	)

	// Compensate bars alignment.
	volumePlot.Y.Padding += (candlesticks.CandleWidth - volumeBars.LineStyle.Width) / 2

	volumePlot.Add(volumeBars)

	render("bbands.png", pricePlot, volumePlot)
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
	pricePlot.Title.Text = "BTCUSDT / Bollinger Bands®"
	// p.X.Label.Text = "Time"
	pricePlot.Y.Label.Text = "Price"
	pricePlot.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	volumePlot = plot.New()
	volumePlot.X.Label.Text = "Time"
	volumePlot.Y.Label.Text = "Volume"
	volumePlot.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}

	return pricePlot, volumePlot
}

func makeBandLines(upper, lower plotter.XYer) (top, bot *plotter.Line, band *hplot.Band) {
	top, err := hplot.NewLine(upper)
	if err != nil {
		panic(err)
	}
	top.LineStyle.Color = color.RGBA{R: 255, A: 255}

	bot, err = hplot.NewLine(lower)
	if err != nil {
		panic(err)
	}
	bot.LineStyle.Color = color.RGBA{B: 255, A: 255}

	band = hplot.NewBand(color.Gray{200}, upper, lower)

	return top, bot, band
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

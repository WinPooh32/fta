package fta

import "github.com/WinPooh32/series"

// Simple moving average - rolling mean in pandas lingo. Also known as 'MA'.
// The simple moving average (SMA) is the most basic of the moving averages used for trading.
func SMA(column series.Data, period int) series.Data {
	return column.Rolling(period).Mean()
}

// The Rate-of-Change (ROC) indicator, which is also referred to as simply Momentum,
// is a pure momentum oscillator that measures the percent change in price from one period to the next.
// The ROC calculation compares the current price with the price “n” periods ago.
func ROC(column series.Data, period int) series.Data {
	diff := column.Rolling(period).Diff()
	shift := column.Rolling(period).Shift()
	return diff.Div(shift).MulScalar(100)
}

// Know Sure Thing (KST) is a momentum oscillator based on the smoothed rate-of-change for four different time frames.
// KST measures price momentum for four different price cycles. It can be used just like any momentum oscillator.
// Chartists can look for divergences, overbought/oversold readings, signal line crossovers and centerline crossovers.
func KST(column series.Data, r1, r2, r3, r4 int) (k, signal series.Data) {
	const window = 10
	var (
		roc1 = ROC(column, r1).Rolling(window).Mean()
		roc2 = ROC(column, r2).Rolling(window).Mean()
		roc3 = ROC(column, r3).Rolling(window).Mean()
		roc4 = ROC(column, r4).Rolling(window).Mean()
	)

	k = roc1.
		Add(roc2.MulScalar(2)).
		Add(roc3.MulScalar(3)).
		Add(roc4.MulScalar(4))

	signal = k.Rolling(window).Mean()

	return k, signal
}

// Fisher Transform was presented by John Ehlers.
// It assumes that price distributions behave like square waves.
func FISH(low, high series.Data, period int, adjust bool) (fish series.Data) {
	var (
		med      = high.Clone().Add(low).MulScalar(0.5)
		ndaylow  = med.Rolling(period).Min()
		ndayhigh = med.Rolling(period).Max()
		raw      = med.Sub(ndaylow).Div(ndayhigh.Sub(ndaylow)).MulScalar(2).SubScalar(1)
		smooth   = raw.EWM(series.AlphaSpan, 5, adjust, false).Mean().Fillna(0, true)

		a   = smooth.Clone().AddScalar(1)       // 1 + smooth
		b   = smooth.MulScalar(-1).AddScalar(1) // 1 - smooth
		log = a.Div(b).Log()
	)
	fish = log.EWM(series.AlphaSpan, 3, adjust, false).Mean()
	return
}

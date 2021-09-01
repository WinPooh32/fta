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

// MACD, MACD Signal and MACD difference.
// The MACD Line oscillates above and below the zero line, which is also known as the centerline.
// These crossovers signal that the 12-day EMA has crossed the 26-day EMA. The direction, of course, depends on the direction of the moving average cross.
// Positive MACD indicates that the 12-day EMA is above the 26-day EMA. Positive values increase as the shorter EMA diverges further from the longer EMA.
// This means upside momentum is increasing. Negative MACD values indicates that the 12-day EMA is below the 26-day EMA.
//
// Negative values increase as the shorter EMA diverges further below the longer EMA. This means downside momentum is increasing.
// Signal line crossovers are the most common MACD signals. The signal line is a 9-day EMA of the MACD Line.
// As a moving average of the indicator, it trails the MACD and makes it easier to spot MACD turns.
// A bullish crossover occurs when the MACD turns up and crosses above the signal line.
// A bearish crossover occurs when the MACD turns down and crosses below the signal line.
func MACD(
	column series.Data,
	periodFast float32,
	periodSlow float32,
	signal float32,
	adjust bool,
) (macd, macdSignal series.Data) {

	var (
		emaFast = column.EWM(series.AlphaSpan, periodFast, adjust, false).Mean()
		emaSlow = column.EWM(series.AlphaSpan, periodSlow, adjust, false).Mean()
	)

	macd = emaFast.Sub(emaSlow)
	macdSignal = macd.EWM(series.AlphaSpan, signal, adjust, false).Mean()

	return macd, macdSignal
}
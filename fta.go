package fta

import (
	"github.com/WinPooh32/series"
	"github.com/WinPooh32/series/math"
)

type DType = series.DType

// Simple moving average - rolling mean in pandas lingo. Also known as 'MA'.
// The simple moving average (SMA) is the most basic of the moving averages used for trading.
func SMA(column series.Data, period int) (sma series.Data) {
	sma = column.Rolling(period).Mean()
	return sma
}

// Simple moving median, an alternative to moving average. SMA, when used to estimate the underlying trend in a time series,
// is susceptible to rare events such as rapid shocks or other anomalies. A more robust estimate of the trend is the simple moving median over n time periods.
func SMM(column series.Data, period int) (smm series.Data) {
	smm = column.Rolling(period).Median()
	return smm
}

// Smoothed simple moving average.
func SSMA(column series.Data, period int, adjust bool) (ssma series.Data) {
	ssma = column.
		EWM(series.Alpha, 1/DType(period), adjust, false).
		Mean()
	return ssma
}

// Exponential Weighted Moving Average - Like all moving average indicators, they are much better suited for trending markets.
// When the market is in a strong and sustained uptrend, the EMA indicator line will also show an uptrend and vice-versa for a down trend.
// EMAs are commonly used in conjunction with other indicators to confirm significant market moves and to gauge their validity.
func EMA(column series.Data, period int, adjust bool) (ema series.Data) {
	ema = column.
		EWM(series.AlphaSpan, DType(period), adjust, false).
		Mean()
	return ema
}

// WMA stands for weighted moving average. It helps to smooth the price curve for better trend identification.
// It places even greater importance on recent data than the EMA does.
func WMA(column series.Data, period int) (wma series.Data) {
	denominator := DType(period*(period+1)) / 2.0

	weights := series.MakeValues(make([]DType, period))
	for i, w := 0, weights.Values(); i < len(w); i++ {
		w[i] = DType(i + 1)
	}

	// Reduce allocations and use temporary array for inplace operations.
	tmp := series.MakeValues(make([]series.DType, 0, period))

	fn := func(data series.Data) series.DType {
		if size := len(data.Values()); size < period {
			weights = weights.Slice(0, size)
		}
		// Make copy of data to tmp.
		tmp = tmp.Slice(0, 0).Append(data)
		return series.Sum(tmp.Mul(weights)) / denominator
	}

	wma = column.Rolling(period).Apply(fn)

	return wma
}

// HMA indicator is a common abbreviation of Hull Moving Average.
// The average was developed by Allan Hull and is used mainly to identify the current market trend.
// Unlike SMA (simple moving average) the curve of Hull moving average is considerably smoother.
// Moreover, because its aim is to minimize the lag between HMA and price it does follow the price activity much closer.
// It is used especially for middle-term and long-term trading.
func HMA(column series.Data, period int) (hma series.Data) {
	halfLength := period / 2
	sqrtLength := int(math.Sqrt(DType(period)))

	wmaf := WMA(column.Clone(), halfLength)
	wmas := WMA(column.Clone(), period)

	deltawma := wmaf.MulScalar(2).Sub(wmas)

	hma = WMA(deltawma, sqrtLength)

	return hma
}

// The Rate-of-Change (ROC) indicator, which is also referred to as simply Momentum,
// is a pure momentum oscillator that measures the percent change in price from one period to the next.
// The ROC calculation compares the current price with the price “n” periods ago.
func ROC(column series.Data, period int) (roc series.Data) {
	diff := column.Diff(period)
	shift := column.Shift(period)
	roc = diff.Div(shift).MulScalar(100)
	return roc
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
		smooth   = raw.EWM(series.AlphaSpan, 5, adjust, false).Mean().Fillna(0)

		a   = smooth.Clone().AddScalar(1)       // 1 + smooth
		b   = smooth.MulScalar(-1).AddScalar(1) // 1 - smooth
		log = a.Div(b).Log()
	)
	fish = log.EWM(series.AlphaSpan, 3, adjust, false).Mean()
	return fish
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
func MACD(column series.Data, periodFast float64, periodSlow float64, signal float64, adjust bool) (macd, macdSignal series.Data) {
	var (
		emaFast = column.EWM(series.AlphaSpan, DType(periodFast), adjust, false).Mean()
		emaSlow = column.EWM(series.AlphaSpan, DType(periodSlow), adjust, false).Mean()
	)

	macd = emaFast.Sub(emaSlow)
	macdSignal = macd.EWM(series.AlphaSpan, DType(signal), adjust, false).Mean()

	return macd, macdSignal
}

// Developed by John Bollinger, Bollinger Bands® are volatility bands placed above and below a moving average.
// Volatility is based on the standard deviation, which changes as volatility increases and decreases.
// The bands automatically widen when volatility increases and narrow when volatility decreases.
func BBANDS(column series.Data, ma series.Data, period int, stdMultiplier float64) (upper, lower series.Data) {
	std := column.Rolling(period).Std(ma, 1).Fillna(0).MulScalar(DType(stdMultiplier))
	upper = ma.Clone().Add(std)
	lower = ma.Clone().Sub(std)
	return upper, lower
}

// %b (pronounced 'percent b') is derived from the formula for Stochastics and shows where price is in relation to the bands.
// %b equals 1 at the upper band and 0 at the lower band.
func PercentB(column series.Data, ma series.Data, period int, stdMultiplier float64) (percentB series.Data) {
	bbLower, bbUpper := BBANDS(column, ma, period, stdMultiplier)
	percentB = column.Clone().Sub(bbLower).Div(bbUpper.Sub(bbLower))
	return percentB
}

// Relative Strength Index (RSI) is a momentum oscillator that measures the speed and change of price movements.
// RSI oscillates between zero and 100. Traditionally, and according to Wilder, RSI is considered overbought when above 70 and oversold when below 30.
// Signals can also be generated by looking for divergences, failure swings and centerline crossovers.
// RSI can also be used to identify the general trend.
func RSI(column series.Data, period int, adjust bool) (rsi series.Data) {
	column = column.Clone()

	var (
		up   = column.Diff(1)
		down = up.Clone()
	)

	upValues := up.Values()
	for i, v := range upValues {
		if v < 0 {
			upValues[i] = 0
		}
	}

	downValues := down.Values()
	for i, v := range downValues {
		if v > 0 {
			downValues[i] = 0
		}
	}

	var (
		alphaParam = 1.0 / float64(period)
		gain       = up.EWM(series.Alpha, DType(alphaParam), adjust, true).Mean()
		loss       = down.Abs().EWM(series.Alpha, DType(alphaParam), adjust, true).Mean()

		rs = gain.Div(loss).Fillna(0)
	)

	rsi = rs.Apply(func(v DType) DType {
		return 100 - (100 / (1 + v))
	})

	return rsi
}

// Connors RSI (CRSI) is a technical analysis indicator created by Larry Connors that is actually a composite of three separate components.
// The Relative Strength Index (RSI), developed by J. Welles Wilder, plays an integral role in Connors RSI.
// Connors RSI outputs a value between 0 and 100, which is then used to identify short-term overbought and oversold conditions.
func CRSI(close series.Data, period int, periodUpDown int, periodrRoc int, adjust bool) (crsi series.Data) {
	var streak DType

	close = close.Clone()

	updown := close.Diff(1).Apply(func(v DType) DType {
		switch {
		case v > 0:
			if streak <= 0 {
				streak = 1
			} else {
				streak++
			}
		case v < 0:
			if streak >= 0 {
				streak = -1
			} else {
				streak--
			}
		default:
			streak = 0
		}
		return streak
	})

	var (
		rsi       = RSI(close, period, adjust)
		rsiUpDown = RSI(updown, periodUpDown, adjust)
		roc       = ROC(close, periodrRoc).Fillna(0)
	)

	crsi = rsi.Add(rsiUpDown).Add(roc).DivScalar(3)

	return crsi
}

// VZO uses price, previous price and moving averages to compute its oscillating value.
// It is a leading indicator that calculates buy and sell signals based on oversold / overbought conditions.
// Oscillations between the 5% and 40% levels mark a bullish trend zone, while oscillations between -40% and 5% mark a bearish trend zone.
// Meanwhile, readings above 40% signal an overbought condition, while readings above 60% signal an extremely overbought condition.
// Alternatively, readings below -40% indicate an oversold condition, which becomes extremely oversold below -60%.
func VZO(price, volume series.Data, period int, adjust bool) (vzo series.Data) {
	sign := func(x DType) DType {
		return math.Copysign(1, x)
	}

	v := price.Clone().
		Diff(1).Apply(sign).
		Mul(volume)

	dvma := v.EWM(series.AlphaSpan, DType(period), adjust, false).Mean()
	vma := volume.Clone().EWM(series.AlphaSpan, DType(period), adjust, false).Mean()

	vzo = dvma.MulScalar(100).Div(vma)

	return vzo
}

// Stochastic oscillator %K
// The stochastic oscillator is a momentum indicator comparing the closing price of a security
// to the range of its prices over a certain period of time.
// The sensitivity of the oscillator to market movements is reducible by adjusting that time
// period or by taking a moving average of the result.
func STOCH(high, low, close series.Data, period int) (stoch series.Data) {
	close = close.Clone()

	highestHigh := high.Rolling(period).Max()
	lowestLow := low.Rolling(period).Min()

	stoch = close.
		Sub(lowestLow).
		Div(
			highestHigh.Sub(lowestLow),
		)

	return stoch
}

// Stochastic oscillator %D
// STOCH%D is a 3 period simple moving average of %K.
func STOCHD(high, low, close series.Data, period int) (stochd series.Data) {
	return STOCH(high, low, close, period).Rolling(period).Mean()
}

// StochRSI is an oscillator that measures the level of RSI relative to its high-low range over a set time period.
// StochRSI applies the Stochastics formula to RSI values, instead of price values. This makes it an indicator of an indicator.
// The result is an oscillator that fluctuates between 0 and 1.
func StochRSI(price series.Data, rsiPeriod, stochPeriod int, adjust bool) (stochRSI series.Data) {
	rsi := RSI(price, rsiPeriod, adjust)
	min := series.Min(rsi)
	max := series.Max(rsi)

	stochRSI = rsi.
		SubScalar(min).
		DivScalar(max - min).
		Rolling(stochPeriod).
		Mean()

	return stochRSI
}

// The accumulation/distribution line was created by Marc Chaikin to determine the flow of money into or out of a security.
// It should not be confused with the advance/decline line. While their initials might be the same, these are entirely different indicators,
// and their uses are different as well. Whereas the advance/decline line can provide insight into market movements,
// the accumulation/distribution line is of use to traders looking to measure buy/sell pressure on a security or confirm the strength of a trend.
func ADL(high, low, close series.Data) (adl series.Data) {
	subCloseLow := close.Clone().Sub(low)
	subHighClose := high.Clone().Sub(close)
	subHighLow := high.Clone().Sub(low)

	mfv := subCloseLow.Sub(subHighClose).Div(subHighLow)

	return mfv.Cumsum()
}

// Chaikin Oscillator, named after its creator, Marc Chaikin, the Chaikin oscillator is an oscillator that measures the accumulation/distribution
// line of the moving average convergence divergence (MACD). The Chaikin oscillator is calculated by subtracting a 10-day exponential moving average (EMA)
// of the accumulation/distribution line from a three-day EMA of the accumulation/distribution line, and highlights the momentum implied by the
// accumulation/distribution line.
func CHAIKIN(high, low, close series.Data, adjust bool) (chaikin series.Data) {
	adl := ADL(high, low, close)

	short := adl.EWM(series.AlphaSpan, 3, adjust, false).Mean()
	long := adl.EWM(series.AlphaSpan, 10, adjust, false).Mean()

	chaikin = short.Sub(long)

	return chaikin
}

// The parabolic SAR indicator, developed by J. Wells Wilder, is used by traders to determine trend direction and potential reversals in price.
// The indicator uses a trailing stop and reverse method called "SAR," or stop and reverse, to identify suitable exit and entry points.
// Traders also refer to the indicator as the parabolic stop and reverse, parabolic SAR, or PSAR.
// https://www.investopedia.com/terms/p/parabolicindicator.asp
// https://virtualizedfrog.wordpress.com/2014/12/09/parabolic-sar-implementation-in-python/
func PSAR(high, low, close series.Data, iaf float64, maxaf float64) (psarSeries, bullSeries, bearSeries series.Data) {
	length := close.Len()

	highValues := high.Values()
	lowValues := low.Values()

	psar := append([]DType(nil), close.Values()...)
	psarBull := make([]DType, length)
	psarBear := make([]DType, length)

	bull := true
	af := DType(iaf)
	hp := highValues[0]
	lp := lowValues[0]

	for i := 2; i < length; i++ {
		if bull {
			psar[i] = psar[i-1] + af*(hp-psar[i-1])
		} else {
			psar[i] = psar[i-1] + af*(lp-psar[i-1])
		}

		reverse := false

		if bull {
			if lowValues[i] < psar[i] {
				bull = false
				reverse = true
				psar[i] = hp
				lp = lowValues[i]
				af = DType(iaf)
			}
		} else {
			if highValues[i] > psar[i] {
				bull = true
				reverse = true
				psar[i] = lp
				hp = highValues[i]
				af = DType(iaf)
			}
		}

		if !reverse {
			if bull {
				if highValues[i] > hp {
					hp = highValues[i]
					af = math.Min(af+DType(iaf), DType(maxaf))
				}
				if lowValues[i-1] < psar[i] {
					psar[i] = lowValues[i-1]
				}
				if lowValues[i-2] < psar[i] {
					psar[i] = lowValues[i-2]
				}
			} else {
				if lowValues[i] < lp {
					lp = lowValues[i]
					af = math.Min(af+DType(iaf), DType(maxaf))
				}
				if highValues[i-1] > psar[i] {
					psar[i] = highValues[i-1]
				}
				if highValues[i-2] > psar[i] {
					psar[i] = highValues[i-2]
				}
			}
		}

		if bull {
			psarBull[i] = psar[i]
		} else {
			psarBear[i] = psar[i]
		}
	}

	index := append([]int64(nil), close.Index()...)

	psarSeries = series.MakeData(close.Freq(), index, psar)
	bullSeries = series.MakeData(close.Freq(), index, psarBull)
	bearSeries = series.MakeData(close.Freq(), index, psarBear)

	return psarSeries, bullSeries, bearSeries
}

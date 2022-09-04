package fta

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/WinPooh32/series"
)

// OHLCV is a data frame of open, high, close, volume columns.
// Implements github.com/pplcc/plotext.TOHLCVer interface.
type OHLCV struct{ Open, High, Low, Close, Volume series.Data }

// Clone returns full copy of ohlcv.
func (ohlcv OHLCV) Clone() OHLCV {
	return OHLCV{
		Open:   ohlcv.Open.Clone(),
		High:   ohlcv.High.Clone(),
		Low:    ohlcv.Low.Clone(),
		Close:  ohlcv.Close.Clone(),
		Volume: ohlcv.Volume.Clone(),
	}
}

// Resample returns resampled copy of ohlcv.
// Interval is the length of one sample in seconds.
func (ohlcv OHLCV) Resample(interval int64) OHLCV {
	const origin = series.OriginEpoch

	ohlcv = ohlcv.Clone()

	return OHLCV{
		Open:   ohlcv.Open.Resample(interval, origin).First(),
		High:   ohlcv.High.Resample(interval, origin).Max(),
		Low:    ohlcv.Low.Resample(interval, origin).Min(),
		Close:  ohlcv.Close.Resample(interval, origin).Last(),
		Volume: ohlcv.Volume.Resample(interval, origin).Sum(),
	}
}

// Slice slices ohlcv frame.
func (ohlcv OHLCV) Slice(begin, end int) OHLCV {
	return OHLCV{
		Open:   ohlcv.Open.Slice(begin, end),
		High:   ohlcv.High.Slice(begin, end),
		Low:    ohlcv.Low.Slice(begin, end),
		Close:  ohlcv.Close.Slice(begin, end),
		Volume: ohlcv.Volume.Slice(begin, end),
	}
}

// Len returns the number of time, open, high, low, close, volume tuples.
func (ohlcv OHLCV) Len() int {
	return ohlcv.Open.Len()
}

// TOHLCV returns an time, open, high, low, close, volume tuple.
func (ohlcv OHLCV) TOHLCV(i int) (t float64, o float64, h float64, l float64, c float64, v float64) {
	t = float64(ohlcv.Open.Index()[i])
	o = float64(ohlcv.Open.At(i))
	h = float64(ohlcv.High.At(i))
	l = float64(ohlcv.Low.At(i))
	c = float64(ohlcv.Close.At(i))
	v = float64(ohlcv.Volume.At(i))
	return
}

// ReadCSV parses ohlcv from csv reader.
// The columns are read at this order: Time Open High Low Close Volume.
// freq is a sample size, usually it's time.Second or time.Millisecond.
func ReadCSV(reader *csv.Reader, freq int64) (ohlcv OHLCV, err error) {
	const (
		Time = iota
		Open
		High
		Low
		Close
		Volume
	)

	var (
		T []int64
		O,
		H,
		L,
		C,
		V []series.DType
	)

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return ohlcv, fmt.Errorf("read csv: %w", err)
		}

		ts, err := strconv.ParseInt(record[Time], 10, 64)
		if err != nil {
			return ohlcv, fmt.Errorf("parse int: field 'Time': %w", err)
		}

		o, err := strconv.ParseFloat(record[Open], 64)
		if err != nil {
			return ohlcv, fmt.Errorf("parse float: field 'Open': %w", err)
		}

		h, err := strconv.ParseFloat(record[High], 64)
		if err != nil {
			return ohlcv, fmt.Errorf("parse float: field 'High': %w", err)
		}

		l, err := strconv.ParseFloat(record[Low], 64)
		if err != nil {
			return ohlcv, fmt.Errorf("parse float: field 'Low': %w", err)
		}

		c, err := strconv.ParseFloat(record[Close], 64)
		if err != nil {
			return ohlcv, fmt.Errorf("parse float: field 'Close': %w", err)
		}

		v, err := strconv.ParseFloat(record[Volume], 64)
		if err != nil {
			return ohlcv, fmt.Errorf("parse float: field 'Volume': %w", err)
		}

		T = append(T, ts)
		O = append(O, DType(o))
		H = append(H, DType(h))
		L = append(L, DType(l))
		C = append(C, DType(c))
		V = append(V, DType(v))
	}

	ohlcv = OHLCV{
		Open:   series.MakeData(freq, T, O),
		High:   series.MakeData(freq, T, H),
		Low:    series.MakeData(freq, T, L),
		Close:  series.MakeData(freq, T, C),
		Volume: series.MakeData(freq, T, V),
	}

	return ohlcv, nil
}

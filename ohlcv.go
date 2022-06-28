package fta

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/WinPooh32/series"
)

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

// ReadCSV parses ohlcv from csv reader.
// The columns are read at this order: Time Open High Low Close Volume.
// Time is expected to be in seconds from January 1st, 1970 at UTC.
func ReadCSV(reader *csv.Reader) (ohlcv OHLCV, err error) {
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

	const freq = int64(time.Second)

	ohlcv = OHLCV{
		Open:   series.MakeData(freq, T, O),
		High:   series.MakeData(freq, T, H),
		Low:    series.MakeData(freq, T, L),
		Close:  series.MakeData(freq, T, C),
		Volume: series.MakeData(freq, T, V),
	}

	return ohlcv, nil
}

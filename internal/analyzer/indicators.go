package analyzer

import (
	"math"

	"github.com/trading-chitti/market-bridge/internal/broker"
)

// CalculateATR calculates Average True Range
func CalculateATR(candles []broker.Candle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	trueRanges := make([]float64, len(candles))

	// First true range is just high - low
	trueRanges[0] = candles[0].High - candles[0].Low

	// Calculate true range for each candle
	for i := 1; i < len(candles); i++ {
		highLow := candles[i].High - candles[i].Low
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)

		trueRanges[i] = math.Max(highLow, math.Max(highClose, lowClose))
	}

	// Calculate ATR using Wilder's smoothing
	atr := make([]float64, len(candles))

	// First ATR is simple average
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += trueRanges[i]
	}
	atr[period-1] = sum / float64(period)

	// Subsequent ATRs use Wilder's smoothing
	for i := period; i < len(candles); i++ {
		atr[i] = (atr[i-1]*float64(period-1) + trueRanges[i]) / float64(period)
	}

	return atr
}

// CalculateVWAP calculates Volume Weighted Average Price
func CalculateVWAP(candles []broker.Candle) []float64 {
	vwap := make([]float64, len(candles))

	cumulativeTPV := 0.0 // Typical Price * Volume
	cumulativeVolume := 0.0

	for i := 0; i < len(candles); i++ {
		typicalPrice := (candles[i].High + candles[i].Low + candles[i].Close) / 3.0
		tpv := typicalPrice * float64(candles[i].Volume)

		cumulativeTPV += tpv
		cumulativeVolume += float64(candles[i].Volume)

		if cumulativeVolume > 0 {
			vwap[i] = cumulativeTPV / cumulativeVolume
		}
	}

	return vwap
}

// SuperTrendResult contains SuperTrend indicator values
type SuperTrendResult struct {
	Trend      []string  // "UP" or "DOWN"
	SuperTrend []float64 // SuperTrend line values
	Signals    []string  // "BUY", "SELL", or ""
}

// CalculateSuperTrend calculates SuperTrend indicator
func CalculateSuperTrend(candles []broker.Candle, period int, multiplier float64) *SuperTrendResult {
	if len(candles) < period {
		return &SuperTrendResult{
			Trend:      []string{},
			SuperTrend: []float64{},
			Signals:    []string{},
		}
	}

	atr := CalculateATR(candles, period)

	basicUpperBand := make([]float64, len(candles))
	basicLowerBand := make([]float64, len(candles))
	finalUpperBand := make([]float64, len(candles))
	finalLowerBand := make([]float64, len(candles))
	superTrend := make([]float64, len(candles))
	trend := make([]string, len(candles))
	signals := make([]string, len(candles))

	for i := 0; i < len(candles); i++ {
		hl2 := (candles[i].High + candles[i].Low) / 2.0

		if i >= period-1 && atr[i] > 0 {
			basicUpperBand[i] = hl2 + (multiplier * atr[i])
			basicLowerBand[i] = hl2 - (multiplier * atr[i])

			// Calculate final bands
			if i == period-1 {
				finalUpperBand[i] = basicUpperBand[i]
				finalLowerBand[i] = basicLowerBand[i]
			} else {
				// Upper band
				if basicUpperBand[i] < finalUpperBand[i-1] || candles[i-1].Close > finalUpperBand[i-1] {
					finalUpperBand[i] = basicUpperBand[i]
				} else {
					finalUpperBand[i] = finalUpperBand[i-1]
				}

				// Lower band
				if basicLowerBand[i] > finalLowerBand[i-1] || candles[i-1].Close < finalLowerBand[i-1] {
					finalLowerBand[i] = basicLowerBand[i]
				} else {
					finalLowerBand[i] = finalLowerBand[i-1]
				}
			}

			// Determine trend
			if i == period-1 {
				if candles[i].Close <= finalUpperBand[i] {
					superTrend[i] = finalUpperBand[i]
					trend[i] = "DOWN"
				} else {
					superTrend[i] = finalLowerBand[i]
					trend[i] = "UP"
				}
			} else {
				if trend[i-1] == "UP" {
					if candles[i].Close <= finalLowerBand[i] {
						superTrend[i] = finalUpperBand[i]
						trend[i] = "DOWN"
						signals[i] = "SELL"
					} else {
						superTrend[i] = finalLowerBand[i]
						trend[i] = "UP"
					}
				} else {
					if candles[i].Close >= finalUpperBand[i] {
						superTrend[i] = finalLowerBand[i]
						trend[i] = "UP"
						signals[i] = "BUY"
					} else {
						superTrend[i] = finalUpperBand[i]
						trend[i] = "DOWN"
					}
				}
			}
		}
	}

	return &SuperTrendResult{
		Trend:      trend,
		SuperTrend: superTrend,
		Signals:    signals,
	}
}

// CalculateStochasticRSI calculates Stochastic RSI
func CalculateStochasticRSI(rsi []float64, period int) []float64 {
	if len(rsi) < period {
		return []float64{}
	}

	stochRSI := make([]float64, len(rsi))

	for i := period - 1; i < len(rsi); i++ {
		// Find min and max RSI in the period
		minRSI := rsi[i-period+1]
		maxRSI := rsi[i-period+1]

		for j := i - period + 1; j <= i; j++ {
			if rsi[j] < minRSI {
				minRSI = rsi[j]
			}
			if rsi[j] > maxRSI {
				maxRSI = rsi[j]
			}
		}

		// Calculate Stochastic RSI
		if maxRSI-minRSI != 0 {
			stochRSI[i] = (rsi[i] - minRSI) / (maxRSI - minRSI) * 100
		}
	}

	return stochRSI
}

// CalculateADX calculates Average Directional Index
func CalculateADX(candles []broker.Candle, period int) []float64 {
	if len(candles) < period+1 {
		return []float64{}
	}

	plusDM := make([]float64, len(candles))
	minusDM := make([]float64, len(candles))
	tr := make([]float64, len(candles))

	// Calculate +DM, -DM, and TR
	for i := 1; i < len(candles); i++ {
		highDiff := candles[i].High - candles[i-1].High
		lowDiff := candles[i-1].Low - candles[i].Low

		if highDiff > lowDiff && highDiff > 0 {
			plusDM[i] = highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			minusDM[i] = lowDiff
		}

		highLow := candles[i].High - candles[i].Low
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)
		tr[i] = math.Max(highLow, math.Max(highClose, lowClose))
	}

	// Smooth +DM, -DM, and TR
	smoothPlusDM := make([]float64, len(candles))
	smoothMinusDM := make([]float64, len(candles))
	smoothTR := make([]float64, len(candles))

	// Initial sum
	for i := 1; i <= period; i++ {
		smoothPlusDM[period] += plusDM[i]
		smoothMinusDM[period] += minusDM[i]
		smoothTR[period] += tr[i]
	}

	// Wilder's smoothing
	for i := period + 1; i < len(candles); i++ {
		smoothPlusDM[i] = smoothPlusDM[i-1] - (smoothPlusDM[i-1] / float64(period)) + plusDM[i]
		smoothMinusDM[i] = smoothMinusDM[i-1] - (smoothMinusDM[i-1] / float64(period)) + minusDM[i]
		smoothTR[i] = smoothTR[i-1] - (smoothTR[i-1] / float64(period)) + tr[i]
	}

	// Calculate +DI and -DI
	plusDI := make([]float64, len(candles))
	minusDI := make([]float64, len(candles))

	for i := period; i < len(candles); i++ {
		if smoothTR[i] != 0 {
			plusDI[i] = (smoothPlusDM[i] / smoothTR[i]) * 100
			minusDI[i] = (smoothMinusDM[i] / smoothTR[i]) * 100
		}
	}

	// Calculate DX and ADX
	dx := make([]float64, len(candles))
	adx := make([]float64, len(candles))

	for i := period; i < len(candles); i++ {
		if plusDI[i]+minusDI[i] != 0 {
			dx[i] = math.Abs(plusDI[i]-minusDI[i]) / (plusDI[i] + minusDI[i]) * 100
		}
	}

	// Smooth DX to get ADX
	sum := 0.0
	for i := period; i < period*2; i++ {
		sum += dx[i]
	}
	adx[period*2-1] = sum / float64(period)

	for i := period * 2; i < len(candles); i++ {
		adx[i] = (adx[i-1]*float64(period-1) + dx[i]) / float64(period)
	}

	return adx
}

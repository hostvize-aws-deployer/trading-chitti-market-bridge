package analyzer

import (
	"math"
	"time"

	"github.com/trading-chitti/market-bridge/internal/broker"
)

// Pattern represents a detected pattern
type Pattern struct {
	Type        string    `json:"type"`         // Pattern name
	Category    string    `json:"category"`     // "candlestick" or "chart"
	Signal      string    `json:"signal"`       // "bullish", "bearish", or "neutral"
	Confidence  float64   `json:"confidence"`   // 0.0 to 1.0
	StartIndex  int       `json:"start_index"`  // Starting candle index
	EndIndex    int       `json:"end_index"`    // Ending candle index
	StartDate   time.Time `json:"start_date"`   // Pattern start date
	EndDate     time.Time `json:"end_date"`     // Pattern end date
	Description string    `json:"description"`  // Human-readable description
	KeyLevels   []float64 `json:"key_levels"`   // Important price levels
}

// PatternScanner scans for patterns in OHLCV data
type PatternScanner struct {
	MinConfidence float64
}

// NewPatternScanner creates a new pattern scanner
func NewPatternScanner() *PatternScanner {
	return &PatternScanner{
		MinConfidence: 0.65, // Default minimum confidence threshold
	}
}

// ScanAllPatterns scans for all supported patterns
func (ps *PatternScanner) ScanAllPatterns(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	// Candlestick patterns
	patterns = append(patterns, ps.DetectDoji(candles)...)
	patterns = append(patterns, ps.DetectHammer(candles)...)
	patterns = append(patterns, ps.DetectShootingStar(candles)...)
	patterns = append(patterns, ps.DetectEngulfing(candles)...)
	patterns = append(patterns, ps.DetectMorningStar(candles)...)
	patterns = append(patterns, ps.DetectEveningStar(candles)...)
	patterns = append(patterns, ps.DetectThreeWhiteSoldiers(candles)...)
	patterns = append(patterns, ps.DetectThreeBlackCrows(candles)...)

	// Chart patterns
	patterns = append(patterns, ps.DetectHeadAndShoulders(candles)...)
	patterns = append(patterns, ps.DetectDoubleTopBottom(candles)...)
	patterns = append(patterns, ps.DetectTriangle(candles)...)
	patterns = append(patterns, ps.DetectFlag(candles)...)
	patterns = append(patterns, ps.DetectWedge(candles)...)

	// Filter by minimum confidence
	filtered := []Pattern{}
	for _, p := range patterns {
		if p.Confidence >= ps.MinConfidence {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

// ============================================================================
// CANDLESTICK PATTERNS
// ============================================================================

// DetectDoji detects Doji candlestick pattern (indecision)
func (ps *PatternScanner) DetectDoji(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 0; i < len(candles); i++ {
		c := candles[i]
		body := math.Abs(c.Close - c.Open)
		range_ := c.High - c.Low

		// Doji: tiny body relative to range
		if range_ > 0 && body/range_ < 0.1 {
			confidence := 1.0 - (body / range_)
			patterns = append(patterns, Pattern{
				Type:        "Doji",
				Category:    "candlestick",
				Signal:      "neutral",
				Confidence:  confidence,
				StartIndex:  i,
				EndIndex:    i,
				StartDate:   c.Date,
				EndDate:     c.Date,
				Description: "Indecision candle with small body and long wicks",
				KeyLevels:   []float64{c.High, c.Low},
			})
		}
	}

	return patterns
}

// DetectHammer detects Hammer pattern (bullish reversal)
func (ps *PatternScanner) DetectHammer(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 1; i < len(candles); i++ {
		c := candles[i]
		prev := candles[i-1]

		body := math.Abs(c.Close - c.Open)
		range_ := c.High - c.Low
		lowerWick := math.Min(c.Open, c.Close) - c.Low
		upperWick := c.High - math.Max(c.Open, c.Close)

		// Hammer: small body at top, long lower wick, in downtrend
		if range_ > 0 &&
			body/range_ < 0.3 &&
			lowerWick > body*2 &&
			upperWick < body &&
			c.Close < prev.Close {

			confidence := math.Min(lowerWick/(body*2), 1.0)
			patterns = append(patterns, Pattern{
				Type:        "Hammer",
				Category:    "candlestick",
				Signal:      "bullish",
				Confidence:  confidence,
				StartIndex:  i,
				EndIndex:    i,
				StartDate:   c.Date,
				EndDate:     c.Date,
				Description: "Bullish reversal with long lower wick after downtrend",
				KeyLevels:   []float64{c.Low},
			})
		}
	}

	return patterns
}

// DetectShootingStar detects Shooting Star pattern (bearish reversal)
func (ps *PatternScanner) DetectShootingStar(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 1; i < len(candles); i++ {
		c := candles[i]
		prev := candles[i-1]

		body := math.Abs(c.Close - c.Open)
		range_ := c.High - c.Low
		upperWick := c.High - math.Max(c.Open, c.Close)
		lowerWick := math.Min(c.Open, c.Close) - c.Low

		// Shooting Star: small body at bottom, long upper wick, in uptrend
		if range_ > 0 &&
			body/range_ < 0.3 &&
			upperWick > body*2 &&
			lowerWick < body &&
			c.Close > prev.Close {

			confidence := math.Min(upperWick/(body*2), 1.0)
			patterns = append(patterns, Pattern{
				Type:        "Shooting Star",
				Category:    "candlestick",
				Signal:      "bearish",
				Confidence:  confidence,
				StartIndex:  i,
				EndIndex:    i,
				StartDate:   c.Date,
				EndDate:     c.Date,
				Description: "Bearish reversal with long upper wick after uptrend",
				KeyLevels:   []float64{c.High},
			})
		}
	}

	return patterns
}

// DetectEngulfing detects Bullish/Bearish Engulfing patterns
func (ps *PatternScanner) DetectEngulfing(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 1; i < len(candles); i++ {
		c := candles[i]
		prev := candles[i-1]

		// Bullish Engulfing
		if prev.Close < prev.Open && // Previous bearish
			c.Close > c.Open && // Current bullish
			c.Open <= prev.Close && // Opens at or below prev close
			c.Close >= prev.Open { // Closes at or above prev open

			bodyRatio := (c.Close - c.Open) / (prev.Open - prev.Close)
			confidence := math.Min(bodyRatio/1.5, 1.0)

			patterns = append(patterns, Pattern{
				Type:        "Bullish Engulfing",
				Category:    "candlestick",
				Signal:      "bullish",
				Confidence:  confidence,
				StartIndex:  i - 1,
				EndIndex:    i,
				StartDate:   prev.Date,
				EndDate:     c.Date,
				Description: "Bullish candle engulfs previous bearish candle",
				KeyLevels:   []float64{prev.Low, c.Close},
			})
		}

		// Bearish Engulfing
		if prev.Close > prev.Open && // Previous bullish
			c.Close < c.Open && // Current bearish
			c.Open >= prev.Close && // Opens at or above prev close
			c.Close <= prev.Open { // Closes at or below prev open

			bodyRatio := (c.Open - c.Close) / (prev.Close - prev.Open)
			confidence := math.Min(bodyRatio/1.5, 1.0)

			patterns = append(patterns, Pattern{
				Type:        "Bearish Engulfing",
				Category:    "candlestick",
				Signal:      "bearish",
				Confidence:  confidence,
				StartIndex:  i - 1,
				EndIndex:    i,
				StartDate:   prev.Date,
				EndDate:     c.Date,
				Description: "Bearish candle engulfs previous bullish candle",
				KeyLevels:   []float64{prev.High, c.Close},
			})
		}
	}

	return patterns
}

// DetectMorningStar detects Morning Star pattern (bullish reversal)
func (ps *PatternScanner) DetectMorningStar(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 2; i < len(candles); i++ {
		c1 := candles[i-2] // First candle
		c2 := candles[i-1] // Star
		c3 := candles[i]   // Third candle

		body1 := c1.Open - c1.Close
		body2 := math.Abs(c2.Close - c2.Open)
		body3 := c3.Close - c3.Open

		// Morning Star: bearish, small body (gap down), bullish (gap up)
		if body1 > 0 && // First is bearish
			body2 < body1*0.3 && // Star has small body
			body3 > 0 && // Third is bullish
			c2.High < c1.Close && // Gap down
			c3.Open > c2.Close { // Gap up

			confidence := 0.75 + (body3 / (body1 + body3) * 0.25)
			patterns = append(patterns, Pattern{
				Type:        "Morning Star",
				Category:    "candlestick",
				Signal:      "bullish",
				Confidence:  confidence,
				StartIndex:  i - 2,
				EndIndex:    i,
				StartDate:   c1.Date,
				EndDate:     c3.Date,
				Description: "Three-candle bullish reversal pattern",
				KeyLevels:   []float64{c2.Low, c3.Close},
			})
		}
	}

	return patterns
}

// DetectEveningStar detects Evening Star pattern (bearish reversal)
func (ps *PatternScanner) DetectEveningStar(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 2; i < len(candles); i++ {
		c1 := candles[i-2]
		c2 := candles[i-1]
		c3 := candles[i]

		body1 := c1.Close - c1.Open
		body2 := math.Abs(c2.Close - c2.Open)
		body3 := c3.Open - c3.Close

		// Evening Star: bullish, small body (gap up), bearish (gap down)
		if body1 > 0 && // First is bullish
			body2 < body1*0.3 && // Star has small body
			body3 > 0 && // Third is bearish
			c2.Low > c1.Close && // Gap up
			c3.Open < c2.Close { // Gap down

			confidence := 0.75 + (body3 / (body1 + body3) * 0.25)
			patterns = append(patterns, Pattern{
				Type:        "Evening Star",
				Category:    "candlestick",
				Signal:      "bearish",
				Confidence:  confidence,
				StartIndex:  i - 2,
				EndIndex:    i,
				StartDate:   c1.Date,
				EndDate:     c3.Date,
				Description: "Three-candle bearish reversal pattern",
				KeyLevels:   []float64{c2.High, c3.Close},
			})
		}
	}

	return patterns
}

// DetectThreeWhiteSoldiers detects Three White Soldiers (strong bullish)
func (ps *PatternScanner) DetectThreeWhiteSoldiers(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 2; i < len(candles); i++ {
		c1 := candles[i-2]
		c2 := candles[i-1]
		c3 := candles[i]

		// All three bullish with higher closes
		if c1.Close > c1.Open &&
			c2.Close > c2.Open &&
			c3.Close > c3.Open &&
			c2.Close > c1.Close &&
			c3.Close > c2.Close &&
			c2.Open > c1.Open &&
			c2.Open < c1.Close && // Opens within previous body
			c3.Open > c2.Open &&
			c3.Open < c2.Close {

			confidence := 0.85
			patterns = append(patterns, Pattern{
				Type:        "Three White Soldiers",
				Category:    "candlestick",
				Signal:      "bullish",
				Confidence:  confidence,
				StartIndex:  i - 2,
				EndIndex:    i,
				StartDate:   c1.Date,
				EndDate:     c3.Date,
				Description: "Three consecutive strong bullish candles",
				KeyLevels:   []float64{c1.Low, c3.High},
			})
		}
	}

	return patterns
}

// DetectThreeBlackCrows detects Three Black Crows (strong bearish)
func (ps *PatternScanner) DetectThreeBlackCrows(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}

	for i := 2; i < len(candles); i++ {
		c1 := candles[i-2]
		c2 := candles[i-1]
		c3 := candles[i]

		// All three bearish with lower closes
		if c1.Close < c1.Open &&
			c2.Close < c2.Open &&
			c3.Close < c3.Open &&
			c2.Close < c1.Close &&
			c3.Close < c2.Close &&
			c2.Open < c1.Open &&
			c2.Open > c1.Close && // Opens within previous body
			c3.Open < c2.Open &&
			c3.Open > c2.Close {

			confidence := 0.85
			patterns = append(patterns, Pattern{
				Type:        "Three Black Crows",
				Category:    "candlestick",
				Signal:      "bearish",
				Confidence:  confidence,
				StartIndex:  i - 2,
				EndIndex:    i,
				StartDate:   c1.Date,
				EndDate:     c3.Date,
				Description: "Three consecutive strong bearish candles",
				KeyLevels:   []float64{c1.High, c3.Low},
			})
		}
	}

	return patterns
}

// ============================================================================
// CHART PATTERNS
// ============================================================================

// DetectHeadAndShoulders detects Head and Shoulders pattern (bearish reversal)
func (ps *PatternScanner) DetectHeadAndShoulders(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}
	minPatternLength := 15

	if len(candles) < minPatternLength {
		return patterns
	}

	// Find local peaks (highs)
	peaks := findLocalPeaks(candles, 5)

	// Need at least 3 peaks for H&S
	for i := 2; i < len(peaks); i++ {
		leftShoulder := peaks[i-2]
		head := peaks[i-1]
		rightShoulder := peaks[i]

		// Head should be higher than shoulders
		// Shoulders should be roughly equal
		if head.High > leftShoulder.High &&
			head.High > rightShoulder.High &&
			math.Abs(leftShoulder.High-rightShoulder.High)/leftShoulder.High < 0.03 {

			// Calculate neckline (support level connecting troughs)
			neckline := findNeckline(candles, leftShoulder.Index, rightShoulder.Index)

			confidence := 0.7
			if neckline > 0 {
				confidence = 0.85
			}

			patterns = append(patterns, Pattern{
				Type:        "Head and Shoulders",
				Category:    "chart",
				Signal:      "bearish",
				Confidence:  confidence,
				StartIndex:  leftShoulder.Index,
				EndIndex:    rightShoulder.Index,
				StartDate:   candles[leftShoulder.Index].Date,
				EndDate:     candles[rightShoulder.Index].Date,
				Description: "Bearish reversal with left shoulder, head, right shoulder",
				KeyLevels:   []float64{head.High, neckline},
			})
		}

		// Inverse Head and Shoulders (bullish)
		if head.Low < leftShoulder.Low &&
			head.Low < rightShoulder.Low &&
			math.Abs(leftShoulder.Low-rightShoulder.Low)/leftShoulder.Low < 0.03 {

			neckline := findNecklineInverse(candles, leftShoulder.Index, rightShoulder.Index)

			confidence := 0.7
			if neckline > 0 {
				confidence = 0.85
			}

			patterns = append(patterns, Pattern{
				Type:        "Inverse Head and Shoulders",
				Category:    "chart",
				Signal:      "bullish",
				Confidence:  confidence,
				StartIndex:  leftShoulder.Index,
				EndIndex:    rightShoulder.Index,
				StartDate:   candles[leftShoulder.Index].Date,
				EndDate:     candles[rightShoulder.Index].Date,
				Description: "Bullish reversal with inverted head and shoulders",
				KeyLevels:   []float64{head.Low, neckline},
			})
		}
	}

	return patterns
}

// DetectDoubleTopBottom detects Double Top/Bottom patterns
func (ps *PatternScanner) DetectDoubleTopBottom(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}
	minPatternLength := 10

	if len(candles) < minPatternLength {
		return patterns
	}

	peaks := findLocalPeaks(candles, 5)

	// Double Top
	for i := 1; i < len(peaks); i++ {
		first := peaks[i-1]
		second := peaks[i]

		// Tops should be roughly equal (within 2%)
		priceDiff := math.Abs(first.High-second.High) / first.High
		timeDiff := second.Index - first.Index

		if priceDiff < 0.02 && timeDiff > 5 && timeDiff < 40 {
			// Find trough between peaks
			trough := findLowestBetween(candles, first.Index, second.Index)

			confidence := 0.75 - (priceDiff * 10) // Higher confidence for closer tops
			patterns = append(patterns, Pattern{
				Type:        "Double Top",
				Category:    "chart",
				Signal:      "bearish",
				Confidence:  confidence,
				StartIndex:  first.Index,
				EndIndex:    second.Index,
				StartDate:   candles[first.Index].Date,
				EndDate:     candles[second.Index].Date,
				Description: "Bearish reversal with two equal peaks",
				KeyLevels:   []float64{first.High, trough},
			})
		}
	}

	// Double Bottom
	troughs := findLocalTroughs(candles, 5)
	for i := 1; i < len(troughs); i++ {
		first := troughs[i-1]
		second := troughs[i]

		priceDiff := math.Abs(first.Low-second.Low) / first.Low
		timeDiff := second.Index - first.Index

		if priceDiff < 0.02 && timeDiff > 5 && timeDiff < 40 {
			peak := findHighestBetween(candles, first.Index, second.Index)

			confidence := 0.75 - (priceDiff * 10)
			patterns = append(patterns, Pattern{
				Type:        "Double Bottom",
				Category:    "chart",
				Signal:      "bullish",
				Confidence:  confidence,
				StartIndex:  first.Index,
				EndIndex:    second.Index,
				StartDate:   candles[first.Index].Date,
				EndDate:     candles[second.Index].Date,
				Description: "Bullish reversal with two equal troughs",
				KeyLevels:   []float64{first.Low, peak},
			})
		}
	}

	return patterns
}

// DetectTriangle detects Triangle patterns (ascending, descending, symmetrical)
func (ps *PatternScanner) DetectTriangle(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}
	minLength := 15

	if len(candles) < minLength {
		return patterns
	}

	// Look for converging trendlines
	for i := minLength; i < len(candles); i++ {
		window := candles[i-minLength : i]

		upperSlope, upperIntercept := findUpperTrendline(window)
		lowerSlope, lowerIntercept := findLowerTrendline(window)

		// Ascending Triangle: flat top, rising bottom
		if math.Abs(upperSlope) < 0.001 && lowerSlope > 0.002 {
			patterns = append(patterns, Pattern{
				Type:        "Ascending Triangle",
				Category:    "chart",
				Signal:      "bullish",
				Confidence:  0.70,
				StartIndex:  i - minLength,
				EndIndex:    i,
				StartDate:   window[0].Date,
				EndDate:     window[len(window)-1].Date,
				Description: "Bullish continuation with horizontal resistance and rising support",
				KeyLevels:   []float64{upperIntercept, lowerIntercept},
			})
		}

		// Descending Triangle: falling top, flat bottom
		if upperSlope < -0.002 && math.Abs(lowerSlope) < 0.001 {
			patterns = append(patterns, Pattern{
				Type:        "Descending Triangle",
				Category:    "chart",
				Signal:      "bearish",
				Confidence:  0.70,
				StartIndex:  i - minLength,
				EndIndex:    i,
				StartDate:   window[0].Date,
				EndDate:     window[len(window)-1].Date,
				Description: "Bearish continuation with falling resistance and horizontal support",
				KeyLevels:   []float64{upperIntercept, lowerIntercept},
			})
		}

		// Symmetrical Triangle: converging lines
		if upperSlope < -0.001 && lowerSlope > 0.001 &&
			math.Abs(upperSlope+lowerSlope) < 0.002 {

			patterns = append(patterns, Pattern{
				Type:        "Symmetrical Triangle",
				Category:    "chart",
				Signal:      "neutral",
				Confidence:  0.65,
				StartIndex:  i - minLength,
				EndIndex:    i,
				StartDate:   window[0].Date,
				EndDate:     window[len(window)-1].Date,
				Description: "Continuation pattern with converging trendlines",
				KeyLevels:   []float64{upperIntercept, lowerIntercept},
			})
		}
	}

	return patterns
}

// DetectFlag detects Flag patterns (bullish/bearish continuation)
func (ps *PatternScanner) DetectFlag(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}
	minFlagLength := 5
	minPoleLength := 8

	if len(candles) < minPoleLength+minFlagLength {
		return patterns
	}

	for i := minPoleLength + minFlagLength; i < len(candles); i++ {
		// Look for strong move (flagpole)
		pole := candles[i-minPoleLength-minFlagLength : i-minFlagLength]
		flag := candles[i-minFlagLength : i]

		poleMove := (pole[len(pole)-1].Close - pole[0].Open) / pole[0].Open
		flagMove := (flag[len(flag)-1].Close - flag[0].Close) / flag[0].Close

		// Bullish Flag: strong up move, slight consolidation down
		if poleMove > 0.05 && flagMove < 0 && math.Abs(flagMove) < poleMove*0.5 {
			patterns = append(patterns, Pattern{
				Type:        "Bullish Flag",
				Category:    "chart",
				Signal:      "bullish",
				Confidence:  0.75,
				StartIndex:  i - minPoleLength - minFlagLength,
				EndIndex:    i,
				StartDate:   pole[0].Date,
				EndDate:     flag[len(flag)-1].Date,
				Description: "Bullish continuation after strong rally",
				KeyLevels:   []float64{pole[len(pole)-1].High, flag[0].Low},
			})
		}

		// Bearish Flag: strong down move, slight consolidation up
		if poleMove < -0.05 && flagMove > 0 && flagMove < math.Abs(poleMove)*0.5 {
			patterns = append(patterns, Pattern{
				Type:        "Bearish Flag",
				Category:    "chart",
				Signal:      "bearish",
				Confidence:  0.75,
				StartIndex:  i - minPoleLength - minFlagLength,
				EndIndex:    i,
				StartDate:   pole[0].Date,
				EndDate:     flag[len(flag)-1].Date,
				Description: "Bearish continuation after strong decline",
				KeyLevels:   []float64{pole[len(pole)-1].Low, flag[0].High},
			})
		}
	}

	return patterns
}

// DetectWedge detects Rising/Falling Wedge patterns
func (ps *PatternScanner) DetectWedge(candles []broker.Candle) []Pattern {
	patterns := []Pattern{}
	minLength := 15

	if len(candles) < minLength {
		return patterns
	}

	for i := minLength; i < len(candles); i++ {
		window := candles[i-minLength : i]

		upperSlope, upperIntercept := findUpperTrendline(window)
		lowerSlope, lowerIntercept := findLowerTrendline(window)

		// Rising Wedge: both slopes positive, converging (bearish)
		if upperSlope > 0 && lowerSlope > 0 && upperSlope < lowerSlope*0.8 {
			patterns = append(patterns, Pattern{
				Type:        "Rising Wedge",
				Category:    "chart",
				Signal:      "bearish",
				Confidence:  0.70,
				StartIndex:  i - minLength,
				EndIndex:    i,
				StartDate:   window[0].Date,
				EndDate:     window[len(window)-1].Date,
				Description: "Bearish reversal with rising converging trendlines",
				KeyLevels:   []float64{upperIntercept, lowerIntercept},
			})
		}

		// Falling Wedge: both slopes negative, converging (bullish)
		if upperSlope < 0 && lowerSlope < 0 && upperSlope > lowerSlope*0.8 {
			patterns = append(patterns, Pattern{
				Type:        "Falling Wedge",
				Category:    "chart",
				Signal:      "bullish",
				Confidence:  0.70,
				StartIndex:  i - minLength,
				EndIndex:    i,
				StartDate:   window[0].Date,
				EndDate:     window[len(window)-1].Date,
				Description: "Bullish reversal with falling converging trendlines",
				KeyLevels:   []float64{upperIntercept, lowerIntercept},
			})
		}
	}

	return patterns
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

type Peak struct {
	Index int
	High  float64
	Low   float64
}

func findLocalPeaks(candles []broker.Candle, window int) []Peak {
	peaks := []Peak{}

	for i := window; i < len(candles)-window; i++ {
		isMax := true
		for j := i - window; j <= i+window; j++ {
			if j != i && candles[j].High >= candles[i].High {
				isMax = false
				break
			}
		}
		if isMax {
			peaks = append(peaks, Peak{Index: i, High: candles[i].High, Low: candles[i].Low})
		}
	}

	return peaks
}

func findLocalTroughs(candles []broker.Candle, window int) []Peak {
	troughs := []Peak{}

	for i := window; i < len(candles)-window; i++ {
		isMin := true
		for j := i - window; j <= i+window; j++ {
			if j != i && candles[j].Low <= candles[i].Low {
				isMin = false
				break
			}
		}
		if isMin {
			troughs = append(troughs, Peak{Index: i, High: candles[i].High, Low: candles[i].Low})
		}
	}

	return troughs
}

func findNeckline(candles []broker.Candle, start, end int) float64 {
	minLow := math.MaxFloat64
	for i := start; i <= end && i < len(candles); i++ {
		if candles[i].Low < minLow {
			minLow = candles[i].Low
		}
	}
	if minLow == math.MaxFloat64 {
		return 0
	}
	return minLow
}

func findNecklineInverse(candles []broker.Candle, start, end int) float64 {
	maxHigh := 0.0
	for i := start; i <= end && i < len(candles); i++ {
		if candles[i].High > maxHigh {
			maxHigh = candles[i].High
		}
	}
	return maxHigh
}

func findLowestBetween(candles []broker.Candle, start, end int) float64 {
	minLow := math.MaxFloat64
	for i := start; i <= end && i < len(candles); i++ {
		if candles[i].Low < minLow {
			minLow = candles[i].Low
		}
	}
	if minLow == math.MaxFloat64 {
		return 0
	}
	return minLow
}

func findHighestBetween(candles []broker.Candle, start, end int) float64 {
	maxHigh := 0.0
	for i := start; i <= end && i < len(candles); i++ {
		if candles[i].High > maxHigh {
			maxHigh = candles[i].High
		}
	}
	return maxHigh
}

// Simple linear regression for trendline
func findUpperTrendline(candles []broker.Candle) (slope, intercept float64) {
	n := float64(len(candles))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, c := range candles {
		x := float64(i)
		y := c.High
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept = (sumY - slope*sumX) / n

	return slope, intercept
}

func findLowerTrendline(candles []broker.Candle) (slope, intercept float64) {
	n := float64(len(candles))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, c := range candles {
		x := float64(i)
		y := c.Low
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept = (sumY - slope*sumX) / n

	return slope, intercept
}

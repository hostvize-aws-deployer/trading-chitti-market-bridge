package analyzer

import (
	"fmt"
	"math"
	"time"
	
	"github.com/trading-chitti/market-bridge/internal/broker"
	"github.com/sirupsen/logrus"
)

// Analyzer52D analyzes 52 days of historical data
type Analyzer52D struct {
	logger *logrus.Logger
}

// NewAnalyzer52D creates a new 52-day analyzer
func NewAnalyzer52D() *Analyzer52D {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	
	return &Analyzer52D{logger: logger}
}

// Analysis represents complete 52-day analysis results
type Analysis struct {
	Symbol       string                 `json:"symbol"`
	PeriodDays   int                    `json:"period_days"`
	StartDate    time.Time              `json:"start_date"`
	EndDate      time.Time              `json:"end_date"`
	Trend        TrendAnalysis          `json:"trend"`
	Volatility   VolatilityAnalysis     `json:"volatility"`
	Volume       VolumeAnalysis         `json:"volume"`
	Support      []float64              `json:"support"`
	Resistance   []float64              `json:"resistance"`
	Indicators   TechnicalIndicators    `json:"indicators"`
	RiskMetrics  RiskMetrics            `json:"risk_metrics"`
	Signals      []Signal               `json:"signals"`
}

// TrendAnalysis represents trend information
type TrendAnalysis struct {
	Direction string  `json:"direction"` // STRONG_UPTREND, UPTREND, SIDEWAYS, DOWNTREND, STRONG_DOWNTREND
	Slope     float64 `json:"slope"`
	RSquared  float64 `json:"r_squared"`
	Strength  float64 `json:"strength"`
}

// VolatilityAnalysis represents volatility metrics
type VolatilityAnalysis struct {
	Annualized     float64 `json:"annualized"`
	DailyStd       float64 `json:"daily_std"`
	ATR            float64 `json:"atr"`
	Classification string  `json:"classification"` // HIGH, MODERATE, LOW
}

// VolumeAnalysis represents volume analysis
type VolumeAnalysis struct {
	Average        int64   `json:"average"`
	RecentAverage  int64   `json:"recent_average"`
	TrendPercent   float64 `json:"trend_pct"`
	Classification string  `json:"classification"` // INCREASING, DECREASING, STABLE
}

// TechnicalIndicators represents technical indicators
type TechnicalIndicators struct {
	SMA20          float64 `json:"sma_20"`
	SMA50          float64 `json:"sma_50"`
	EMA12          float64 `json:"ema_12"`
	EMA26          float64 `json:"ema_26"`
	RSI            float64 `json:"rsi"`
	MACD           float64 `json:"macd"`
	MACDSignal     float64 `json:"macd_signal"`
	BBUpper        float64 `json:"bb_upper"`
	BBMiddle       float64 `json:"bb_middle"`
	BBLower        float64 `json:"bb_lower"`
	BBPosition     string  `json:"bb_position"` // OVERBOUGHT, OVERSOLD, NEUTRAL
}

// RiskMetrics represents risk-adjusted metrics
type RiskMetrics struct {
	SharpeRatio  float64 `json:"sharpe_ratio"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	WinRate      float64 `json:"win_rate"`
}

// Signal represents a trading signal
type Signal struct {
	Type        string  `json:"type"`         // BUY or SELL
	Strategy    string  `json:"strategy"`     
	Confidence  float64 `json:"confidence"`   // 0.0 to 1.0
	EntryPrice  float64 `json:"entry_price"`
	StopLoss    float64 `json:"stop_loss"`
	TakeProfit  float64 `json:"take_profit"`
	Reason      string  `json:"reason"`
}

// Analyze performs 52-day analysis on candle data
func (a *Analyzer52D) Analyze(symbol string, candles []broker.Candle) (*Analysis, error) {
	if len(candles) < 30 {
		return nil, fmt.Errorf("insufficient data: need at least 30 candles, got %d", len(candles))
	}
	
	a.logger.Infof("📊 Analyzing %d days of data for %s", len(candles), symbol)
	
	analysis := &Analysis{
		Symbol:     symbol,
		PeriodDays: len(candles),
		StartDate:  candles[0].Date,
		EndDate:    candles[len(candles)-1].Date,
	}
	
	// Extract price and volume slices
	closes := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	volumes := make([]int64, len(candles))
	
	for i, c := range candles {
		closes[i] = c.Close
		highs[i] = c.High
		lows[i] = c.Low
		volumes[i] = c.Volume
	}
	
	// Perform analysis
	analysis.Trend = a.analyzeTrend(closes)
	analysis.Volatility = a.analyzeVolatility(candles)
	analysis.Volume = a.analyzeVolume(volumes)
	analysis.Support, analysis.Resistance = a.findSupportResistance(highs, lows)
	analysis.Indicators = a.calculateIndicators(candles)
	analysis.RiskMetrics = a.calculateRiskMetrics(closes)
	analysis.Signals = a.generateSignals(analysis)
	
	a.logger.Infof("✅ Analysis complete: %d signals generated", len(analysis.Signals))
	
	return analysis, nil
}

// analyzeTrend identifies overall trend using linear regression
func (a *Analyzer52D) analyzeTrend(prices []float64) TrendAnalysis {
	n := float64(len(prices))
	sumX := n * (n - 1) / 2
	sumY := sum(prices)
	sumXY := 0.0
	sumX2 := 0.0
	
	for i, y := range prices {
		x := float64(i)
		sumXY += x * y
		sumX2 += x * x
	}
	
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n
	
	// Calculate R²
	meanY := sumY / n
	ssTotal := 0.0
	ssResidual := 0.0
	
	for i, y := range prices {
		x := float64(i)
		predicted := slope*x + intercept
		ssTotal += math.Pow(y-meanY, 2)
		ssResidual += math.Pow(y-predicted, 2)
	}
	
	rSquared := 1 - (ssResidual / ssTotal)
	
	// Classify trend
	direction := "SIDEWAYS"
	if slope > 0.5 {
		direction = "STRONG_UPTREND"
	} else if slope > 0.1 {
		direction = "UPTREND"
	} else if slope < -0.5 {
		direction = "STRONG_DOWNTREND"
	} else if slope < -0.1 {
		direction = "DOWNTREND"
	}
	
	return TrendAnalysis{
		Direction: direction,
		Slope:     slope,
		RSquared:  rSquared,
		Strength:  math.Abs(math.Sqrt(rSquared)),
	}
}

// analyzeVolatility calculates volatility metrics
func (a *Analyzer52D) analyzeVolatility(candles []broker.Candle) VolatilityAnalysis {
	returns := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		returns[i-1] = (candles[i].Close - candles[i-1].Close) / candles[i-1].Close
	}
	
	dailyStd := stdDev(returns)
	annualized := dailyStd * math.Sqrt(252)
	
	atr := calculateATR(candles, 14)
	
	classification := "LOW"
	if annualized > 0.4 {
		classification = "HIGH"
	} else if annualized > 0.2 {
		classification = "MODERATE"
	}
	
	return VolatilityAnalysis{
		Annualized:     annualized,
		DailyStd:       dailyStd,
		ATR:            atr,
		Classification: classification,
	}
}

// analyzeVolume analyzes volume patterns
func (a *Analyzer52D) analyzeVolume(volumes []int64) VolumeAnalysis {
	avgVolume := int64(sum(intToFloat(volumes)) / float64(len(volumes)))
	
	recentVolumes := volumes
	if len(volumes) > 5 {
		recentVolumes = volumes[len(volumes)-5:]
	}
	recentAvg := int64(sum(intToFloat(recentVolumes)) / float64(len(recentVolumes)))
	
	trendPct := float64(recentAvg-avgVolume) / float64(avgVolume) * 100
	
	classification := "STABLE"
	if trendPct > 20 {
		classification = "INCREASING"
	} else if trendPct < -20 {
		classification = "DECREASING"
	}
	
	return VolumeAnalysis{
		Average:        avgVolume,
		RecentAverage:  recentAvg,
		TrendPercent:   trendPct,
		Classification: classification,
	}
}

// findSupportResistance finds support and resistance levels
func (a *Analyzer52D) findSupportResistance(highs, lows []float64) ([]float64, []float64) {
	// Get recent 20-day highs and lows
	recentHighs := highs
	recentLows := lows
	if len(highs) > 20 {
		recentHighs = highs[len(highs)-20:]
		recentLows = lows[len(lows)-20:]
	}
	
	// Find top 3 highs and lows
	resistance := findTopN(recentHighs, 3)
	support := findBottomN(recentLows, 3)
	
	return support, resistance
}

// calculateIndicators calculates technical indicators
func (a *Analyzer52D) calculateIndicators(candles []broker.Candle) TechnicalIndicators {
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}
	
	sma20 := sma(closes, 20)
	sma50 := 0.0
	if len(closes) >= 50 {
		sma50 = sma(closes, 50)
	}
	
	ema12 := ema(closes, 12)
	ema26 := ema(closes, 26)
	
	rsi := calculateRSI(closes, 14)
	
	macd := ema12 - ema26
	macdSignal := ema(closes, 9)
	
	bbMiddle := sma20
	bbStd := stdDev(closes[len(closes)-20:])
	bbUpper := bbMiddle + 2*bbStd
	bbLower := bbMiddle - 2*bbStd
	
	currentPrice := closes[len(closes)-1]
	bbPosition := "NEUTRAL"
	if bbUpper > 0 && bbLower > 0 {
		position := (currentPrice - bbLower) / (bbUpper - bbLower) * 100
		if position > 80 {
			bbPosition = "OVERBOUGHT"
		} else if position < 20 {
			bbPosition = "OVERSOLD"
		}
	}
	
	return TechnicalIndicators{
		SMA20:      sma20,
		SMA50:      sma50,
		EMA12:      ema12,
		EMA26:      ema26,
		RSI:        rsi,
		MACD:       macd,
		MACDSignal: macdSignal,
		BBUpper:    bbUpper,
		BBMiddle:   bbMiddle,
		BBLower:    bbLower,
		BBPosition: bbPosition,
	}
}

// calculateRiskMetrics calculates risk-adjusted metrics
func (a *Analyzer52D) calculateRiskMetrics(prices []float64) RiskMetrics {
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}
	
	meanReturn := sum(returns) / float64(len(returns))
	stdReturn := stdDev(returns)
	
	riskFreeRate := 0.06 / 252 // Daily risk-free rate (6% annual)
	sharpeRatio := 0.0
	if stdReturn > 0 {
		sharpeRatio = (meanReturn - riskFreeRate) / stdReturn * math.Sqrt(252)
	}
	
	// Calculate max drawdown
	maxDrawdown := 0.0
	peak := prices[0]
	for _, price := range prices {
		if price > peak {
			peak = price
		}
		drawdown := (peak - price) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	
	// Calculate win rate
	wins := 0
	for _, r := range returns {
		if r > 0 {
			wins++
		}
	}
	winRate := float64(wins) / float64(len(returns))
	
	return RiskMetrics{
		SharpeRatio: sharpeRatio,
		MaxDrawdown: maxDrawdown,
		WinRate:     winRate,
	}
}

// generateSignals generates trading signals based on analysis
func (a *Analyzer52D) generateSignals(analysis *Analysis) []Signal {
	signals := []Signal{}
	currentPrice := analysis.Indicators.SMA20 // Approximation
	
	// Signal 1: RSI Oversold/Overbought
	if analysis.Indicators.RSI < 30 {
		confidence := math.Min(0.9, (30-analysis.Indicators.RSI)/30)
		signals = append(signals, Signal{
			Type:       "BUY",
			Strategy:   "RSI_OVERSOLD",
			Confidence: confidence,
			EntryPrice: currentPrice,
			StopLoss:   currentPrice * 0.97,
			TakeProfit: currentPrice * 1.05,
			Reason:     fmt.Sprintf("RSI at %.1f indicates oversold condition", analysis.Indicators.RSI),
		})
	} else if analysis.Indicators.RSI > 70 {
		confidence := math.Min(0.9, (analysis.Indicators.RSI-70)/30)
		signals = append(signals, Signal{
			Type:       "SELL",
			Strategy:   "RSI_OVERBOUGHT",
			Confidence: confidence,
			EntryPrice: currentPrice,
			StopLoss:   currentPrice * 1.03,
			TakeProfit: currentPrice * 0.95,
			Reason:     fmt.Sprintf("RSI at %.1f indicates overbought condition", analysis.Indicators.RSI),
		})
	}
	
	// Signal 2: Trend + Volume
	if (analysis.Trend.Direction == "UPTREND" || analysis.Trend.Direction == "STRONG_UPTREND") && 
	   analysis.Volume.Classification == "INCREASING" {
		confidence := math.Min(0.85, analysis.Trend.Strength+0.2)
		stopLoss := currentPrice * 0.95
		if len(analysis.Support) > 0 {
			stopLoss = analysis.Support[0]
		}
		takeProfit := currentPrice * 1.10
		if len(analysis.Resistance) > 0 {
			takeProfit = analysis.Resistance[0]
		}
		
		signals = append(signals, Signal{
			Type:       "BUY",
			Strategy:   "TREND_VOLUME",
			Confidence: confidence,
			EntryPrice: currentPrice,
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
			Reason:     fmt.Sprintf("%s with increasing volume", analysis.Trend.Direction),
		})
	}
	
	// Signal 3: Bollinger Band Bounce
	if analysis.Indicators.BBPosition == "OVERSOLD" {
		signals = append(signals, Signal{
			Type:       "BUY",
			Strategy:   "BOLLINGER_BOUNCE",
			Confidence: 0.70,
			EntryPrice: currentPrice,
			StopLoss:   analysis.Indicators.BBLower * 0.99,
			TakeProfit: analysis.Indicators.BBMiddle,
			Reason:     "Price near lower Bollinger Band, potential bounce",
		})
	} else if analysis.Indicators.BBPosition == "OVERBOUGHT" {
		signals = append(signals, Signal{
			Type:       "SELL",
			Strategy:   "BOLLINGER_REVERSAL",
			Confidence: 0.70,
			EntryPrice: currentPrice,
			StopLoss:   analysis.Indicators.BBUpper * 1.01,
			TakeProfit: analysis.Indicators.BBMiddle,
			Reason:     "Price near upper Bollinger Band, potential reversal",
		})
	}
	
	// Filter signals by confidence
	filtered := []Signal{}
	for _, sig := range signals {
		if sig.Confidence >= 0.65 {
			filtered = append(filtered, sig)
		}
	}
	
	return filtered
}

// Helper functions

func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func stdDev(values []float64) float64 {
	mean := sum(values) / float64(len(values))
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	return math.Sqrt(variance / float64(len(values)))
}

func sma(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	recent := prices[len(prices)-period:]
	return sum(recent) / float64(period)
}

func ema(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	multiplier := 2.0 / float64(period+1)
	ema := prices[0]
	for i := 1; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}
	return ema
}

func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0
	}
	
	gains := 0.0
	losses := 0.0
	
	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	
	if avgLoss == 0 {
		return 100
	}
	
	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))
	
	return rsi
}

func calculateATR(candles []broker.Candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}
	
	trueRanges := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		highLow := candles[i].High - candles[i].Low
		highClose := math.Abs(candles[i].High - candles[i-1].Close)
		lowClose := math.Abs(candles[i].Low - candles[i-1].Close)
		
		trueRanges[i-1] = math.Max(highLow, math.Max(highClose, lowClose))
	}
	
	recent := trueRanges[len(trueRanges)-period:]
	return sum(recent) / float64(period)
}

func intToFloat(values []int64) []float64 {
	result := make([]float64, len(values))
	for i, v := range values {
		result[i] = float64(v)
	}
	return result
}

func findTopN(values []float64, n int) []float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Simple bubble sort for top N
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] < sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	if len(sorted) < n {
		return sorted
	}
	return sorted[:n]
}

func findBottomN(values []float64, n int) []float64 {
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Simple bubble sort for bottom N
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	if len(sorted) < n {
		return sorted
	}
	return sorted[:n]
}

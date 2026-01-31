package watchlist

// Watchlist represents a predefined list of symbols
type Watchlist struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Symbols     []string `json:"symbols"`
	Category    string   `json:"category"`
	Exchange    string   `json:"exchange"`
}

// GetAllWatchlists returns all predefined watchlists
func GetAllWatchlists() []Watchlist {
	return []Watchlist{
		Nifty50(),
		BankNifty(),
		NiftyNext50(),
		NiftyMidcap50(),
		TopGainers(),
		TopLosers(),
		MostActive(),
		Pharma(),
		IT(),
		Auto(),
		Metal(),
		Energy(),
		FMCG(),
		Realty(),
		Media(),
	}
}

// GetWatchlist returns a specific watchlist by name
func GetWatchlist(name string) *Watchlist {
	watchlists := GetAllWatchlists()
	for _, wl := range watchlists {
		if wl.Name == name {
			return &wl
		}
	}
	return nil
}

// ListWatchlistNames returns names of all watchlists
func ListWatchlistNames() []string {
	watchlists := GetAllWatchlists()
	names := make([]string, len(watchlists))
	for i, wl := range watchlists {
		names[i] = wl.Name
	}
	return names
}

// ============================================================================
// INDEX WATCHLISTS
// ============================================================================

// Nifty50 returns Nifty 50 index constituents
func Nifty50() Watchlist {
	return Watchlist{
		Name:        "NIFTY50",
		Description: "Nifty 50 Index constituents",
		Category:    "index",
		Exchange:    "NSE",
		Symbols: []string{
			"RELIANCE", "TCS", "HDFCBANK", "INFY", "ICICIBANK",
			"HINDUNILVR", "ITC", "SBIN", "BHARTIARTL", "KOTAKBANK",
			"LT", "AXISBANK", "ASIANPAINT", "MARUTI", "HCLTECH",
			"SUNPHARMA", "BAJFINANCE", "TITAN", "ULTRACEMCO", "WIPRO",
			"ONGC", "NTPC", "POWERGRID", "M&M", "TATASTEEL",
			"TATAMOTORS", "TECHM", "BAJAJFINSV", "ADANIPORTS", "COALINDIA",
			"NESTLEIND", "DRREDDY", "JSWSTEEL", "INDUSINDBK", "DIVISLAB",
			"GRASIM", "CIPLA", "HINDALCO", "HEROMOTOCO", "EICHERMOT",
			"BRITANNIA", "APOLLOHOSP", "UPL", "SBILIFE", "TATACONSUM",
			"BAJAJ-AUTO", "HDFCLIFE", "ADANIENT", "BPCL", "TATAPOWER",
		},
	}
}

// BankNifty returns Bank Nifty index constituents
func BankNifty() Watchlist {
	return Watchlist{
		Name:        "BANKNIFTY",
		Description: "Bank Nifty Index constituents",
		Category:    "index",
		Exchange:    "NSE",
		Symbols: []string{
			"HDFCBANK", "ICICIBANK", "SBIN", "KOTAKBANK", "AXISBANK",
			"INDUSINDBK", "BANKBARODA", "PNB", "IDFCFIRSTB", "FEDERALBNK",
			"AUBANK", "BANDHANBNK",
		},
	}
}

// NiftyNext50 returns Nifty Next 50 constituents
func NiftyNext50() Watchlist {
	return Watchlist{
		Name:        "NIFTYNEXT50",
		Description: "Nifty Next 50 Index constituents",
		Category:    "index",
		Exchange:    "NSE",
		Symbols: []string{
			"ADANIGREEN", "ACC", "AMBUJACEM", "BERGEPAINT", "BOSCHLTD",
			"CHOLAFIN", "COLPAL", "DABUR", "DLF", "GODREJCP",
			"HAVELLS", "ICICIPRULI", "INDIGO", "LUPIN", "MARICO",
			"MCDOWELL-N", "NMDC", "PAGEIND", "PETRONET", "PIDILITIND",
			"PGHH", "SBICARD", "SHREECEM", "SIEMENS", "TATAPOWER",
			"TORNTPHARM", "ZEEL", "VEDL", "VOLTAS", "MUTHOOTFIN",
		},
	}
}

// NiftyMidcap50 returns Nifty Midcap 50 constituents
func NiftyMidcap50() Watchlist {
	return Watchlist{
		Name:        "NIFTYMIDCAP50",
		Description: "Nifty Midcap 50 Index constituents",
		Category:    "index",
		Exchange:    "NSE",
		Symbols: []string{
			"ABFRL", "AUBANK", "BANDHANBNK", "BHARATFORG", "BIOCON",
			"CROMPTON", "CONCOR", "COFORGE", "DIXON", "GMRINFRA",
			"GAIL", "GODREJPROP", "HINDPETRO", "IDFCFIRSTB", "IRCTC",
			"JINDALSTEL", "JUBLFOOD", "LICHSGFIN", "LALPATHLAB", "MRF",
			"MOTHERSON", "MPHASIS", "NAUKRI", "OBEROIRLTY", "PEL",
			"PERSISTENT", "PIIND", "RECLTD", "SRF", "SAIL",
		},
	}
}

// ============================================================================
// MARKET MOVERS
// ============================================================================

// TopGainers returns top gainers watchlist
func TopGainers() Watchlist {
	return Watchlist{
		Name:        "TOP_GAINERS",
		Description: "Top gaining stocks (updated daily)",
		Category:    "movers",
		Exchange:    "NSE",
		Symbols: []string{
			// This would be dynamically updated based on market data
			"RELIANCE", "TCS", "INFY", "HDFC", "ICICIBANK",
			"SBIN", "BHARTIARTL", "ITC", "KOTAKBANK", "LT",
		},
	}
}

// TopLosers returns top losers watchlist
func TopLosers() Watchlist {
	return Watchlist{
		Name:        "TOP_LOSERS",
		Description: "Top losing stocks (updated daily)",
		Category:    "movers",
		Exchange:    "NSE",
		Symbols: []string{
			// This would be dynamically updated based on market data
			"TATASTEEL", "HINDALCO", "VEDL", "COALINDIA", "JSWSTEEL",
		},
	}
}

// MostActive returns most active stocks by volume
func MostActive() Watchlist {
	return Watchlist{
		Name:        "MOST_ACTIVE",
		Description: "Most actively traded stocks by volume",
		Category:    "movers",
		Exchange:    "NSE",
		Symbols: []string{
			"RELIANCE", "SBIN", "TATASTEEL", "ICICIBANK", "HDFCBANK",
			"INFY", "TCS", "AXISBANK", "KOTAKBANK", "ITC",
		},
	}
}

// ============================================================================
// SECTOR WATCHLISTS
// ============================================================================

// Pharma returns pharma sector stocks
func Pharma() Watchlist {
	return Watchlist{
		Name:        "PHARMA",
		Description: "Pharmaceutical sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"SUNPHARMA", "DRREDDY", "CIPLA", "DIVISLAB", "AUROPHARMA",
			"LUPIN", "BIOCON", "TORNTPHARM", "ALKEM", "ABBOTINDIA",
			"CADILAHC", "GLENMARK", "IPCALAB", "LAURUSLABS", "NATCOPHARMA",
		},
	}
}

// IT returns IT sector stocks
func IT() Watchlist {
	return Watchlist{
		Name:        "IT",
		Description: "Information Technology sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"TCS", "INFY", "WIPRO", "HCLTECH", "TECHM",
			"LTIM", "MPHASIS", "COFORGE", "PERSISTENT", "MINDTREE",
		},
	}
}

// Auto returns auto sector stocks
func Auto() Watchlist {
	return Watchlist{
		Name:        "AUTO",
		Description: "Automobile sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"MARUTI", "M&M", "TATAMOTORS", "HEROMOTOCO", "EICHERMOT",
			"BAJAJ-AUTO", "ASHOKLEY", "MOTHERSON", "BHARATFORG", "APOLLOTYRE",
			"MRF", "BALKRISIND", "TVSMOTOR", "ESCORTS", "EXIDEIND",
		},
	}
}

// Metal returns metal sector stocks
func Metal() Watchlist {
	return Watchlist{
		Name:        "METAL",
		Description: "Metals & Mining sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"TATASTEEL", "JSWSTEEL", "HINDALCO", "VEDL", "COALINDIA",
			"NMDC", "SAIL", "JINDALSTEL", "NATIONALUM", "HINDZINC",
			"APLAPOLLO", "RATNAMANI", "WELSPUNIND", "MOIL", "GMRINFRA",
		},
	}
}

// Energy returns energy sector stocks
func Energy() Watchlist {
	return Watchlist{
		Name:        "ENERGY",
		Description: "Oil, Gas & Energy sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"RELIANCE", "ONGC", "BPCL", "IOC", "HINDPETRO",
			"GAIL", "POWERGRID", "NTPC", "ADANIGREEN", "TATAPOWER",
			"ADANIPOWER", "COALINDIA", "OIL", "PETRONET", "GSPL",
		},
	}
}

// FMCG returns FMCG sector stocks
func FMCG() Watchlist {
	return Watchlist{
		Name:        "FMCG",
		Description: "Fast Moving Consumer Goods sector",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"HINDUNILVR", "ITC", "NESTLEIND", "BRITANNIA", "TATACONSUM",
			"DABUR", "GODREJCP", "MARICO", "COLPAL", "MCDOWELL-N",
			"UBL", "EMAMILTD", "VBL", "RADICO", "TATACONS",
		},
	}
}

// Realty returns real estate sector stocks
func Realty() Watchlist {
	return Watchlist{
		Name:        "REALTY",
		Description: "Real Estate sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"DLF", "GODREJPROP", "OBEROIRLTY", "PHOENIXLTD", "PRESTIGE",
			"BRIGADE", "SOBHA", "MAHLIFE", "SUNTECK", "LODHA",
		},
	}
}

// Media returns media & entertainment stocks
func Media() Watchlist {
	return Watchlist{
		Name:        "MEDIA",
		Description: "Media & Entertainment sector stocks",
		Category:    "sector",
		Exchange:    "NSE",
		Symbols: []string{
			"ZEEL", "SUNTV", "PVR", "INOXLEISUR", "TVTODAY",
			"NETWORK18", "TV18BRDCST", "DBCORP", "HT", "JAGRAN",
		},
	}
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// GetWatchlistsByCategory returns all watchlists in a category
func GetWatchlistsByCategory(category string) []Watchlist {
	allWatchlists := GetAllWatchlists()
	filtered := []Watchlist{}

	for _, wl := range allWatchlists {
		if wl.Category == category {
			filtered = append(filtered, wl)
		}
	}

	return filtered
}

// GetCategories returns all watchlist categories
func GetCategories() []string {
	return []string{"index", "movers", "sector"}
}

// MergeWatchlists combines multiple watchlists into one
func MergeWatchlists(names []string) *Watchlist {
	symbolsMap := make(map[string]bool)
	description := "Combined watchlist: "

	for i, name := range names {
		wl := GetWatchlist(name)
		if wl == nil {
			continue
		}

		for _, symbol := range wl.Symbols {
			symbolsMap[symbol] = true
		}

		if i > 0 {
			description += ", "
		}
		description += wl.Name
	}

	symbols := make([]string, 0, len(symbolsMap))
	for symbol := range symbolsMap {
		symbols = append(symbols, symbol)
	}

	return &Watchlist{
		Name:        "CUSTOM",
		Description: description,
		Category:    "custom",
		Exchange:    "NSE",
		Symbols:     symbols,
	}
}

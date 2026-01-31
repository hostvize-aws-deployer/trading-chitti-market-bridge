"""
Unit tests for watchlists module

Tests predefined watchlists and helper functions.
"""

import pytest
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../..'))

from internal.watchlist.watchlists import (
    Watchlist,
    GetAllWatchlists,
    GetWatchlist,
    ListWatchlistNames,
    GetWatchlistsByCategory,
    GetCategories,
    MergeWatchlists,
    Nifty50,
    BankNifty,
    NiftyNext50,
    TopGainers,
    IT,
    Pharma,
)


class TestWatchlistStructure:
    """Test Watchlist data structure"""

    def test_nifty50_structure(self):
        """Test NIFTY50 watchlist has correct structure"""
        wl = Nifty50()

        assert isinstance(wl, Watchlist)
        assert wl.Name == "NIFTY50"
        assert wl.Description == "Nifty 50 Index constituents"
        assert wl.Category == "index"
        assert wl.Exchange == "NSE"
        assert isinstance(wl.Symbols, list)
        assert len(wl.Symbols) == 50

    def test_banknifty_structure(self):
        """Test BANKNIFTY watchlist"""
        wl = BankNifty()

        assert wl.Name == "BANKNIFTY"
        assert wl.Category == "index"
        assert wl.Exchange == "NSE"
        assert len(wl.Symbols) == 12

    def test_sector_watchlist_structure(self):
        """Test sector watchlist structure"""
        wl = IT()

        assert wl.Name == "IT"
        assert wl.Category == "sector"
        assert wl.Exchange == "NSE"
        assert len(wl.Symbols) >= 10

    def test_symbols_are_strings(self):
        """Test all symbols are non-empty strings"""
        wl = Nifty50()

        for symbol in wl.Symbols:
            assert isinstance(symbol, str)
            assert len(symbol) > 0
            assert symbol == symbol.upper()  # Should be uppercase


class TestWatchlistRetrieval:
    """Test watchlist retrieval functions"""

    def test_get_all_watchlists(self):
        """Test GetAllWatchlists returns all watchlists"""
        watchlists = GetAllWatchlists()

        assert isinstance(watchlists, list)
        assert len(watchlists) == 15  # Total predefined watchlists
        assert all(isinstance(wl, Watchlist) for wl in watchlists)

    def test_list_watchlist_names(self):
        """Test ListWatchlistNames returns all names"""
        names = ListWatchlistNames()

        assert isinstance(names, list)
        assert len(names) == 15
        assert "NIFTY50" in names
        assert "BANKNIFTY" in names
        assert "IT" in names
        assert "PHARMA" in names

    def test_get_watchlist_by_name(self):
        """Test GetWatchlist retrieves correct watchlist"""
        wl = GetWatchlist("NIFTY50")

        assert wl is not None
        assert wl.Name == "NIFTY50"
        assert len(wl.Symbols) == 50

    def test_get_nonexistent_watchlist(self):
        """Test GetWatchlist returns None for non-existent watchlist"""
        wl = GetWatchlist("NONEXISTENT")

        assert wl is None

    def test_get_categories(self):
        """Test GetCategories returns all categories"""
        categories = GetCategories()

        assert isinstance(categories, list)
        assert len(categories) == 3
        assert "index" in categories
        assert "movers" in categories
        assert "sector" in categories


class TestWatchlistFiltering:
    """Test watchlist filtering by category"""

    def test_get_index_watchlists(self):
        """Test filtering by 'index' category"""
        watchlists = GetWatchlistsByCategory("index")

        assert isinstance(watchlists, list)
        assert len(watchlists) == 4  # NIFTY50, BANKNIFTY, NIFTYNEXT50, NIFTYMIDCAP50
        assert all(wl.Category == "index" for wl in watchlists)

    def test_get_sector_watchlists(self):
        """Test filtering by 'sector' category"""
        watchlists = GetWatchlistsByCategory("sector")

        assert isinstance(watchlists, list)
        assert len(watchlists) == 8  # IT, PHARMA, AUTO, METAL, ENERGY, FMCG, REALTY, MEDIA
        assert all(wl.Category == "sector" for wl in watchlists)

    def test_get_movers_watchlists(self):
        """Test filtering by 'movers' category"""
        watchlists = GetWatchlistsByCategory("movers")

        assert isinstance(watchlists, list)
        assert len(watchlists) == 3  # TOP_GAINERS, TOP_LOSERS, MOST_ACTIVE
        assert all(wl.Category == "movers" for wl in watchlists)

    def test_get_empty_category(self):
        """Test filtering by non-existent category"""
        watchlists = GetWatchlistsByCategory("nonexistent")

        assert isinstance(watchlists, list)
        assert len(watchlists) == 0


class TestWatchlistMerge:
    """Test watchlist merging functionality"""

    def test_merge_two_watchlists(self):
        """Test merging two watchlists"""
        merged = MergeWatchlists(["NIFTY50", "BANKNIFTY"])

        assert merged is not None
        assert merged.Name == "CUSTOM"
        assert merged.Category == "custom"

        # Should contain symbols from both (with duplicates removed)
        nifty50 = GetWatchlist("NIFTY50")
        banknifty = GetWatchlist("BANKNIFTY")

        # Count unique symbols
        all_symbols = set(nifty50.Symbols + banknifty.Symbols)
        assert len(merged.Symbols) == len(all_symbols)

    def test_merge_removes_duplicates(self):
        """Test merge removes duplicate symbols"""
        # NIFTY50 contains some BANKNIFTY symbols
        merged = MergeWatchlists(["NIFTY50", "BANKNIFTY"])

        # Check no duplicates
        assert len(merged.Symbols) == len(set(merged.Symbols))

    def test_merge_single_watchlist(self):
        """Test merging single watchlist"""
        merged = MergeWatchlists(["IT"])

        assert merged is not None
        it_wl = GetWatchlist("IT")
        assert len(merged.Symbols) == len(it_wl.Symbols)

    def test_merge_nonexistent_watchlist(self):
        """Test merging with non-existent watchlist name"""
        merged = MergeWatchlists(["NIFTY50", "NONEXISTENT"])

        # Should only include symbols from NIFTY50
        assert merged is not None
        nifty50 = GetWatchlist("NIFTY50")
        assert len(merged.Symbols) == len(nifty50.Symbols)


class TestSymbolContent:
    """Test specific watchlist symbol content"""

    def test_nifty50_contains_major_stocks(self):
        """Test NIFTY50 contains expected major stocks"""
        wl = Nifty50()

        # Major stocks that should be in NIFTY50
        expected = ["RELIANCE", "TCS", "HDFCBANK", "INFY", "ICICIBANK"]

        for symbol in expected:
            assert symbol in wl.Symbols

    def test_banknifty_contains_banks(self):
        """Test BANKNIFTY contains bank stocks"""
        wl = BankNifty()

        # Major banks
        expected = ["HDFCBANK", "ICICIBANK", "SBIN", "KOTAKBANK", "AXISBANK"]

        for symbol in expected:
            assert symbol in wl.Symbols

    def test_it_watchlist_contains_it_stocks(self):
        """Test IT watchlist contains IT companies"""
        wl = IT()

        expected = ["TCS", "INFY", "WIPRO", "HCLTECH", "TECHM"]

        for symbol in expected:
            assert symbol in wl.Symbols

    def test_pharma_watchlist_contains_pharma_stocks(self):
        """Test PHARMA watchlist contains pharma companies"""
        wl = Pharma()

        expected = ["SUNPHARMA", "DRREDDY", "CIPLA", "DIVISLAB", "LUPIN"]

        for symbol in expected:
            assert symbol in wl.Symbols


@pytest.mark.parametrize("watchlist_name,expected_min_symbols", [
    ("NIFTY50", 50),
    ("BANKNIFTY", 12),
    ("NIFTYNEXT50", 30),
    ("NIFTYMIDCAP50", 30),
    ("IT", 10),
    ("PHARMA", 10),
    ("AUTO", 10),
    ("METAL", 10),
])
def test_watchlist_minimum_symbols(watchlist_name, expected_min_symbols):
    """Test watchlists have minimum expected symbols (parametrized)"""
    wl = GetWatchlist(watchlist_name)

    assert wl is not None
    assert len(wl.Symbols) >= expected_min_symbols


@pytest.mark.parametrize("category,expected_count", [
    ("index", 4),
    ("sector", 8),
    ("movers", 3),
])
def test_category_counts(category, expected_count):
    """Test correct number of watchlists per category (parametrized)"""
    watchlists = GetWatchlistsByCategory(category)

    assert len(watchlists) == expected_count


def test_all_watchlists_unique_names():
    """Test all watchlist names are unique"""
    watchlists = GetAllWatchlists()
    names = [wl.Name for wl in watchlists]

    assert len(names) == len(set(names))


def test_all_watchlists_have_symbols():
    """Test all watchlists have at least one symbol"""
    watchlists = GetAllWatchlists()

    for wl in watchlists:
        assert len(wl.Symbols) > 0


def test_watchlist_description_not_empty():
    """Test all watchlists have descriptions"""
    watchlists = GetAllWatchlists()

    for wl in watchlists:
        assert wl.Description is not None
        assert len(wl.Description) > 0


if __name__ == '__main__':
    pytest.main([__file__, '-v'])

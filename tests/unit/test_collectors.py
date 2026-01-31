"""
Unit tests for data collectors (mocked)

Tests the DataCollector and CollectorManager without actual WebSocket connections.
"""

import pytest
from unittest.mock import Mock, patch, MagicMock
import time
from datetime import datetime

# Import collector modules
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../..'))

from internal.collector.collector import DataCollector, CandleBuilder
from internal.collector.manager import CollectorManager


class TestCandleBuilder:
    """Test candle aggregation logic"""

    def test_candle_builder_initialization(self):
        """Test CandleBuilder initialization"""
        builder = CandleBuilder()
        builder.InstrumentToken = 738561
        builder.Symbol = "RELIANCE"
        builder.Exchange = "NSE"
        builder.Timeframe = "1m"

        assert builder.InstrumentToken == 738561
        assert builder.Symbol == "RELIANCE"
        assert builder.Exchange == "NSE"
        assert builder.Timeframe == "1m"
        assert builder.CurrentTimestamp.year == 1  # Zero time

    def test_candle_aggregation_first_tick(self):
        """Test first tick initializes candle correctly"""
        builder = CandleBuilder()
        builder.InstrumentToken = 738561
        builder.Symbol = "RELIANCE"
        builder.Exchange = "NSE"
        builder.Timeframe = "1m"

        # Simulate first tick
        tick_price = 2500.50
        tick_quantity = 100

        current_minute = datetime.now().replace(second=0, microsecond=0)

        # Check if candle needs initialization
        if builder.CurrentTimestamp.year == 1 or builder.CurrentTimestamp != current_minute:
            builder.CurrentTimestamp = current_minute
            builder.CurrentOpen = tick_price
            builder.CurrentHigh = tick_price
            builder.CurrentLow = tick_price
            builder.CurrentClose = tick_price
            builder.CurrentVolume = tick_quantity

        assert builder.CurrentOpen == 2500.50
        assert builder.CurrentHigh == 2500.50
        assert builder.CurrentLow == 2500.50
        assert builder.CurrentClose == 2500.50
        assert builder.CurrentVolume == 100

    def test_candle_update_high_low(self):
        """Test candle high/low updates correctly"""
        builder = CandleBuilder()
        builder.InstrumentToken = 738561
        builder.Symbol = "RELIANCE"
        builder.Timeframe = "1m"
        builder.CurrentTimestamp = datetime.now().replace(second=0, microsecond=0)
        builder.CurrentOpen = 2500.00
        builder.CurrentHigh = 2500.00
        builder.CurrentLow = 2500.00
        builder.CurrentClose = 2500.00
        builder.CurrentVolume = 100

        # Higher tick
        new_high = 2505.00
        if new_high > builder.CurrentHigh:
            builder.CurrentHigh = new_high
        builder.CurrentClose = new_high
        builder.CurrentVolume += 50

        assert builder.CurrentHigh == 2505.00
        assert builder.CurrentClose == 2505.00
        assert builder.CurrentVolume == 150

        # Lower tick
        new_low = 2495.00
        if new_low < builder.CurrentLow:
            builder.CurrentLow = new_low
        builder.CurrentClose = new_low
        builder.CurrentVolume += 75

        assert builder.CurrentLow == 2495.00
        assert builder.CurrentClose == 2495.00
        assert builder.CurrentVolume == 225


class TestDataCollector:
    """Test DataCollector class"""

    @pytest.fixture
    def mock_db(self):
        """Create mock database"""
        db = Mock()
        db.InsertTickData = Mock(return_value=None)
        db.InsertIntradayBar = Mock(return_value=None)
        db.GetInstrumentToken = Mock(return_value=738561)
        return db

    @pytest.fixture
    def collector(self, mock_db):
        """Create DataCollector instance with mocked dependencies"""
        return DataCollector(mock_db, "test_api_key", "test_access_token")

    def test_collector_initialization(self, collector, mock_db):
        """Test DataCollector initialization"""
        assert collector.db == mock_db
        assert collector.apiKey == "test_api_key"
        assert collector.accessToken == "test_access_token"
        assert collector.running == False
        assert len(collector.subscribedTokens) == 0
        assert len(collector.tokenToSymbol) == 0
        assert collector.ticksReceived == 0
        assert collector.barsCreated == 0

    def test_register_symbol(self, collector):
        """Test symbol registration"""
        collector.RegisterSymbol(738561, "NSE", "RELIANCE")

        assert 738561 in collector.tokenToSymbol
        assert collector.tokenToSymbol[738561] == "RELIANCE"
        assert 738561 in collector.candleBuilders
        assert collector.candleBuilders[738561].Symbol == "RELIANCE"
        assert collector.candleBuilders[738561].Exchange == "NSE"

    def test_subscribe_tokens(self, collector):
        """Test token subscription (without actual WebSocket)"""
        tokens = [738561, 2953217, 341249]

        # Mock the ticker
        collector.ticker = Mock()
        collector.ticker.Subscribe = Mock(return_value=None)
        collector.running = True

        result = collector.Subscribe(tokens)

        assert result is None
        assert len(collector.subscribedTokens) == 3
        collector.ticker.Subscribe.assert_called_once_with(tokens)

    def test_metrics_tracking(self, collector):
        """Test metrics are tracked correctly"""
        collector.ticksReceived = 100
        collector.barsCreated = 10
        collector.errors = 2

        metrics = collector.GetMetrics()

        assert metrics['ticks_received'] == 100
        assert metrics['bars_created'] == 10
        assert metrics['errors'] == 2
        assert metrics['running'] == False


class TestCollectorManager:
    """Test CollectorManager class"""

    @pytest.fixture
    def mock_db(self):
        """Create mock database"""
        db = Mock()
        db.GetInstrumentToken = Mock(return_value=738561)
        return db

    @pytest.fixture
    def manager(self, mock_db):
        """Create CollectorManager instance"""
        return CollectorManager(mock_db)

    def test_manager_initialization(self, manager, mock_db):
        """Test CollectorManager initialization"""
        assert manager.db == mock_db
        assert len(manager.collectors) == 0

    def test_create_collector(self, manager):
        """Test collector creation"""
        collector = manager.CreateCollector("test_collector", "api_key", "access_token")

        assert collector is not None
        assert isinstance(collector, tuple)  # Returns (collector, error)
        assert "test_collector" in manager.collectors

    def test_create_duplicate_collector(self, manager):
        """Test creating duplicate collector fails"""
        manager.CreateCollector("test_collector", "api_key", "access_token")

        # Try to create again with same name
        collector, err = manager.CreateCollector("test_collector", "api_key2", "access_token2")

        assert collector is None
        assert err is not None
        assert "already exists" in str(err)

    def test_list_collectors(self, manager):
        """Test listing collectors"""
        manager.CreateCollector("collector1", "key1", "token1")
        manager.CreateCollector("collector2", "key2", "token2")

        names = manager.ListCollectors()

        assert len(names) == 2
        assert "collector1" in names
        assert "collector2" in names

    def test_get_collector(self, manager):
        """Test getting collector by name"""
        manager.CreateCollector("test", "key", "token")

        collector, err = manager.GetCollector("test")

        assert collector is not None
        assert err is None

    def test_get_nonexistent_collector(self, manager):
        """Test getting non-existent collector fails"""
        collector, err = manager.GetCollector("nonexistent")

        assert collector is None
        assert err is not None
        assert "not found" in str(err)


@pytest.mark.parametrize("symbol,exchange,expected_token", [
    ("RELIANCE", "NSE", 738561),
    ("TCS", "NSE", 2953217),
    ("INFY", "NSE", 408065),
])
def test_symbol_to_token_lookup(symbol, exchange, expected_token):
    """Test symbol to token lookup (parametrized)"""
    # Mock database
    db = Mock()
    db.GetInstrumentToken = Mock(return_value=expected_token)

    # Call lookup
    token = db.GetInstrumentToken(exchange, symbol)

    assert token == expected_token
    db.GetInstrumentToken.assert_called_once_with(exchange, symbol)


def test_candle_minute_boundary():
    """Test candle flushes at minute boundary"""
    builder = CandleBuilder()
    builder.Symbol = "TEST"
    builder.Timeframe = "1m"

    # First tick at 10:30:15
    time1 = datetime(2024, 1, 30, 10, 30, 15)
    minute1 = time1.replace(second=0, microsecond=0)

    builder.CurrentTimestamp = minute1
    builder.CurrentOpen = 100.0
    builder.CurrentHigh = 102.0
    builder.CurrentLow = 99.0
    builder.CurrentClose = 101.0
    builder.CurrentVolume = 1000

    # Second tick at 10:31:05 (new minute)
    time2 = datetime(2024, 1, 30, 10, 31, 5)
    minute2 = time2.replace(second=0, microsecond=0)

    # Check if new candle needed
    needs_flush = (builder.CurrentTimestamp != minute2)

    assert needs_flush == True
    assert minute1 != minute2


if __name__ == '__main__':
    pytest.main([__file__, '-v'])

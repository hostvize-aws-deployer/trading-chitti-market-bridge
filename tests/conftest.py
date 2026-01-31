"""
Shared pytest fixtures for all tests

This file provides common fixtures used across unit and integration tests.
"""

import pytest
import os
import sys
import tempfile
from unittest.mock import Mock

# Add project root to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))


@pytest.fixture
def mock_database():
    """Create a mock database instance"""
    db = Mock()

    # Mock common database methods
    db.GetInstrumentToken = Mock(return_value=738561)
    db.InsertTickData = Mock(return_value=None)
    db.InsertIntradayBar = Mock(return_value=None)
    db.BulkInsertIntradayBars = Mock(return_value=None)
    db.GetIntradayBars = Mock(return_value=[])
    db.GetLatestIntradayBar = Mock(return_value=None)
    db.Close = Mock()

    return db


@pytest.fixture
def sample_tick_data():
    """Sample tick data for testing"""
    return {
        'InstrumentToken': 738561,
        'Symbol': 'RELIANCE',
        'Exchange': 'NSE',
        'LastPrice': 2500.50,
        'LastQuantity': 100,
        'Timestamp': '2024-01-30T10:30:00Z',
    }


@pytest.fixture
def sample_candle_data():
    """Sample OHLCV candle data"""
    return {
        'Exchange': 'NSE',
        'Symbol': 'RELIANCE',
        'InstrumentToken': 738561,
        'BarTimestamp': '2024-01-30T10:30:00Z',
        'Timeframe': '1m',
        'Open': 2500.00,
        'High': 2505.00,
        'Low': 2495.00,
        'Close': 2502.00,
        'Volume': 5000,
        'Source': 'zerodha',
    }


@pytest.fixture
def sample_symbols():
    """Sample symbol list for testing"""
    return ['RELIANCE', 'TCS', 'INFY', 'HDFCBANK', 'ICICIBANK']


@pytest.fixture
def sample_prices():
    """Sample price series for indicator calculations"""
    return [100.0, 102.0, 101.0, 103.0, 105.0, 104.0, 106.0, 108.0, 107.0, 109.0]


@pytest.fixture
def temp_database():
    """Create temporary SQLite database for testing"""
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
        db_path = f.name

    yield db_path

    # Cleanup
    if os.path.exists(db_path):
        os.unlink(db_path)


@pytest.fixture
def temp_config_file():
    """Create temporary configuration file"""
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
        f.write("""
collectors:
  - name: test_collector
    api_key: ${TEST_API_KEY}
    access_token: ${TEST_ACCESS_TOKEN}
    auto_start: false
    symbols:
      - RELIANCE
      - TCS
    mode: full
""")
        config_path = f.name

    yield config_path

    # Cleanup
    if os.path.exists(config_path):
        os.unlink(config_path)


@pytest.fixture
def mock_zerodha_response():
    """Mock Zerodha API response"""
    return {
        'status': 'success',
        'data': {
            'user_id': 'TEST123',
            'email': 'test@example.com',
            'user_name': 'Test User',
            'user_shortname': 'Test',
            'broker': 'ZERODHA',
            'exchanges': ['NSE', 'BSE'],
            'products': ['CNC', 'MIS', 'NRML'],
            'order_types': ['MARKET', 'LIMIT', 'SL', 'SL-M'],
        }
    }


@pytest.fixture
def mock_market_quote():
    """Mock market quote data"""
    return {
        'NSE:RELIANCE': {
            'instrument_token': 738561,
            'last_price': 2500.50,
            'ohlc': {
                'open': 2495.00,
                'high': 2510.00,
                'low': 2490.00,
                'close': 2485.00,
            },
            'volume': 1250000,
            'last_quantity': 100,
            'average_price': 2502.25,
            'last_trade_time': '2024-01-30 15:29:00',
            'oi': 0,
            'oi_day_high': 0,
            'oi_day_low': 0,
            'depth': {
                'buy': [
                    {'price': 2500.00, 'quantity': 500, 'orders': 5},
                    {'price': 2499.50, 'quantity': 1000, 'orders': 10},
                ],
                'sell': [
                    {'price': 2501.00, 'quantity': 750, 'orders': 7},
                    {'price': 2501.50, 'quantity': 1200, 'orders': 12},
                ],
            },
        }
    }


@pytest.fixture(scope="session")
def test_environment():
    """Set up test environment variables"""
    # Save original values
    original_env = dict(os.environ)

    # Set test values
    os.environ['TEST_API_KEY'] = 'test_key_123'
    os.environ['TEST_ACCESS_TOKEN'] = 'test_token_abc'
    os.environ['TRADING_CHITTI_PG_DSN'] = 'postgresql://test@localhost:5432/test_db'

    yield

    # Restore original environment
    os.environ.clear()
    os.environ.update(original_env)


# Pytest configuration
def pytest_configure(config):
    """Configure pytest markers"""
    config.addinivalue_line(
        "markers", "integration: marks tests as integration tests (require running services)"
    )
    config.addinivalue_line(
        "markers", "slow: marks tests as slow running"
    )
    config.addinivalue_line(
        "markers", "db: marks tests requiring database"
    )
    config.addinivalue_line(
        "markers", "websocket: marks tests requiring WebSocket connection"
    )


# Pytest hooks
def pytest_collection_modifyitems(config, items):
    """Auto-mark integration tests"""
    for item in items:
        # Auto-mark tests in integration folder
        if "integration" in str(item.fspath):
            item.add_marker(pytest.mark.integration)

        # Auto-mark tests with 'db' in name
        if "db" in item.name or "database" in item.name:
            item.add_marker(pytest.mark.db)


# Custom assertions
def assert_valid_timestamp(timestamp_str):
    """Assert timestamp is valid ISO format"""
    from datetime import datetime
    try:
        datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
        return True
    except ValueError:
        return False


def assert_valid_ohlc(ohlc_data):
    """Assert OHLC data is valid"""
    assert 'open' in ohlc_data
    assert 'high' in ohlc_data
    assert 'low' in ohlc_data
    assert 'close' in ohlc_data

    # High should be highest
    assert ohlc_data['high'] >= ohlc_data['open']
    assert ohlc_data['high'] >= ohlc_data['close']

    # Low should be lowest
    assert ohlc_data['low'] <= ohlc_data['open']
    assert ohlc_data['low'] <= ohlc_data['close']


# Add custom assertions to pytest namespace
pytest.assert_valid_timestamp = assert_valid_timestamp
pytest.assert_valid_ohlc = assert_valid_ohlc

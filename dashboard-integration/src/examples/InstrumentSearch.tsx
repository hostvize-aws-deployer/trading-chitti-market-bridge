import React, { useState, useCallback, useEffect } from 'react';
import { useInstruments, Instrument } from '../hooks';

export interface InstrumentSearchProps {
  onSelect: (instrument: Instrument) => void;
  placeholder?: string;
}

export function InstrumentSearch({
  onSelect,
  placeholder = 'Search stocks...',
}: InstrumentSearchProps) {
  const { searchInstruments, loading } = useInstruments();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Instrument[]>([]);
  const [showResults, setShowResults] = useState(false);

  // Debounced search
  useEffect(() => {
    if (query.length < 2) {
      setResults([]);
      return;
    }

    const timer = setTimeout(async () => {
      const instruments = await searchInstruments(query, 10);
      setResults(instruments);
      setShowResults(true);
    }, 300);

    return () => clearTimeout(timer);
  }, [query, searchInstruments]);

  const handleSelect = (instrument: Instrument) => {
    setQuery(instrument.tradingsymbol);
    setShowResults(false);
    onSelect(instrument);
  };

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onFocus={() => setShowResults(true)}
        onBlur={() => setTimeout(() => setShowResults(false), 200)}
        placeholder={placeholder}
        style={{
          width: '100%',
          padding: '12px',
          fontSize: '16px',
          border: '1px solid #ddd',
          borderRadius: '8px',
        }}
      />

      {loading && (
        <div style={{ position: 'absolute', right: '12px', top: '12px' }}>
          <span>⏳</span>
        </div>
      )}

      {showResults && results.length > 0 && (
        <div
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            right: 0,
            marginTop: '4px',
            backgroundColor: 'white',
            border: '1px solid #ddd',
            borderRadius: '8px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.1)',
            maxHeight: '300px',
            overflowY: 'auto',
            zIndex: 1000,
          }}
        >
          {results.map((instrument) => (
            <div
              key={instrument.instrument_token}
              onClick={() => handleSelect(instrument)}
              style={{
                padding: '12px',
                cursor: 'pointer',
                borderBottom: '1px solid #f0f0f0',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = '#f8f8f8';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = 'white';
              }}
            >
              <div style={{ fontWeight: 'bold' }}>{instrument.tradingsymbol}</div>
              <div style={{ fontSize: '12px', color: '#666', marginTop: '2px' }}>
                {instrument.name}
              </div>
              <div style={{ fontSize: '10px', color: '#999', marginTop: '2px' }}>
                {instrument.exchange} • {instrument.segment}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

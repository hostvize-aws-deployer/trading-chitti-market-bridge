import { useState, useEffect, useCallback, useRef } from 'react';

export interface Tick {
  type: 'tick';
  instrument_token: number;
  last_price: number;
  last_quantity: number;
  volume: number;
  timestamp: string;
  ohlc: {
    open: number;
    high: number;
    low: number;
    close: number;
  };
}

export interface OrderUpdate {
  type: 'order_update';
  order_id: string;
  status: string;
  tradingsymbol: string;
  exchange: string;
  transaction_type: string;
  quantity: number;
  filled_quantity: number;
  pending_quantity: number;
  price: number;
  average_price: number;
  status_message: string;
  timestamp: string;
}

export interface StatusMessage {
  type: 'status';
  status: 'reconnecting' | 'disconnected' | 'connected';
  attempt?: number;
  delay?: string;
  message?: string;
}

export type WebSocketMessage = Tick | OrderUpdate | StatusMessage;

export interface UseWebSocketOptions {
  url?: string;
  autoConnect?: boolean;
  onTick?: (tick: Tick) => void;
  onOrderUpdate?: (order: OrderUpdate) => void;
  onStatusChange?: (status: StatusMessage) => void;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const {
    url = 'ws://localhost:6005/ws/market',
    autoConnect = true,
    onTick,
    onOrderUpdate,
    onStatusChange,
  } = options;

  const [connected, setConnected] = useState(false);
  const [status, setStatus] = useState<StatusMessage['status']>('disconnected');
  const [ticks, setTicks] = useState<Tick[]>([]);
  const [orders, setOrders] = useState<OrderUpdate[]>([]);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const ws = new WebSocket(url);

    ws.onopen = () => {
      console.log('✅ WebSocket connected');
      setConnected(true);
      setStatus('connected');
    };

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);

        switch (message.type) {
          case 'tick':
            setTicks((prev) => {
              const filtered = prev.filter(
                (t) => t.instrument_token !== message.instrument_token
              );
              return [...filtered, message].slice(-100); // Keep last 100 ticks
            });
            onTick?.(message);
            break;

          case 'order_update':
            setOrders((prev) => [...prev, message].slice(-50));
            onOrderUpdate?.(message);
            break;

          case 'status':
            setStatus(message.status);
            onStatusChange?.(message);
            break;
        }
      } catch (error) {
        console.error('WebSocket message parse error:', error);
      }
    };

    ws.onerror = (error) => {
      console.error('❌ WebSocket error:', error);
    };

    ws.onclose = () => {
      console.log('⚠️  WebSocket disconnected');
      setConnected(false);
      setStatus('disconnected');

      // Auto-reconnect after 3 seconds
      if (autoConnect) {
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('🔄 Reconnecting...');
          connect();
        }, 3000);
      }
    };

    wsRef.current = ws;
  }, [url, autoConnect, onTick, onOrderUpdate, onStatusChange]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    wsRef.current?.close();
    wsRef.current = null;
    setConnected(false);
    setStatus('disconnected');
  }, []);

  const subscribe = useCallback((tokens: number[]) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          action: 'subscribe',
          tokens,
        })
      );
      console.log('📊 Subscribed to instruments:', tokens);
    } else {
      console.warn('WebSocket not connected. Cannot subscribe.');
    }
  }, []);

  const unsubscribe = useCallback((tokens: number[]) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          action: 'unsubscribe',
          tokens,
        })
      );
      console.log('📊 Unsubscribed from instruments:', tokens);
    }
  }, []);

  const getTickByToken = useCallback(
    (token: number) => {
      return ticks.find((tick) => tick.instrument_token === token);
    },
    [ticks]
  );

  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [autoConnect, connect, disconnect]);

  return {
    connected,
    status,
    ticks,
    orders,
    connect,
    disconnect,
    subscribe,
    unsubscribe,
    getTickByToken,
  };
}

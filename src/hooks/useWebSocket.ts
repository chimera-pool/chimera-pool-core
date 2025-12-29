import { useState, useEffect, useRef, useCallback } from 'react';

export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error' | 'reconnecting';

export interface WebSocketMessage {
  type: string;
  payload?: any;
}

export interface UseWebSocketOptions {
  onMessage?: (message: WebSocketMessage) => void;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onReconnectAttempt?: (attempt: number, delay: number) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  /** Enable exponential backoff for reconnection delays */
  useExponentialBackoff?: boolean;
  /** Maximum delay in ms for exponential backoff (default: 30000) */
  maxBackoffDelay?: number;
  /** Jitter factor to add randomness to backoff (0-1, default: 0.1) */
  backoffJitter?: number;
}

export interface UseWebSocketReturn {
  connectionState: ConnectionState;
  sendMessage: (message: WebSocketMessage) => void;
  disconnect: () => void;
  reconnect: () => void;
  /** Current reconnection attempt number */
  reconnectAttempt: number;
  /** Time until next reconnection attempt in ms */
  nextReconnectDelay: number;
}

export const useWebSocket = (
  url: string,
  options: UseWebSocketOptions = {}
): UseWebSocketReturn => {
  const {
    onMessage,
    onError,
    onConnect,
    onDisconnect,
    onReconnectAttempt,
    reconnectInterval = 3000,
    maxReconnectAttempts = 10,
    useExponentialBackoff = true,
    maxBackoffDelay = 30000,
    backoffJitter = 0.1,
  } = options;

  /**
   * Calculate reconnection delay with exponential backoff
   * Formula: min(baseDelay * 2^attempt + jitter, maxDelay)
   */
  const calculateBackoffDelay = useCallback((attempt: number): number => {
    if (!useExponentialBackoff) {
      return reconnectInterval;
    }
    
    const exponentialDelay = reconnectInterval * Math.pow(2, attempt);
    const jitter = exponentialDelay * backoffJitter * Math.random();
    return Math.min(exponentialDelay + jitter, maxBackoffDelay);
  }, [reconnectInterval, useExponentialBackoff, maxBackoffDelay, backoffJitter]);

  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting');
  const [reconnectAttempt, setReconnectAttempt] = useState(0);
  const [nextReconnectDelay, setNextReconnectDelay] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const shouldReconnectRef = useRef(true);

  const connect = useCallback(() => {
    try {
      setConnectionState('connecting');
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.addEventListener('open', () => {
        setConnectionState('connected');
        reconnectAttemptsRef.current = 0;
        setReconnectAttempt(0);
        setNextReconnectDelay(0);
        onConnect?.();
      });

      ws.addEventListener('message', (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          onMessage?.(message);
        } catch (error) {
          onError?.(new Error(`Failed to parse WebSocket message: ${event.data}`));
        }
      });

      ws.addEventListener('error', (event) => {
        setConnectionState('error');
        onError?.(new Error('WebSocket connection error'));
      });

      ws.addEventListener('close', () => {
        setConnectionState('disconnected');
        onDisconnect?.();

        // Attempt reconnection if enabled and within limits
        if (
          shouldReconnectRef.current &&
          reconnectAttemptsRef.current < maxReconnectAttempts
        ) {
          const currentAttempt = reconnectAttemptsRef.current;
          reconnectAttemptsRef.current++;
          setReconnectAttempt(currentAttempt + 1);
          setConnectionState('reconnecting');
          
          const delay = calculateBackoffDelay(currentAttempt);
          setNextReconnectDelay(delay);
          onReconnectAttempt?.(currentAttempt + 1, delay);
          
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        }
      });
    } catch (error) {
      setConnectionState('error');
      onError?.(error as Error);
    }
  }, [url, onMessage, onError, onConnect, onDisconnect, onReconnectAttempt, maxReconnectAttempts, calculateBackoffDelay]);

  const sendMessage = useCallback((message: WebSocketMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message));
    }
  }, []);

  const disconnect = useCallback(() => {
    shouldReconnectRef.current = false;
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    
    setConnectionState('disconnected');
  }, []);

  const reconnect = useCallback(() => {
    shouldReconnectRef.current = true;
    reconnectAttemptsRef.current = 0;
    setReconnectAttempt(0);
    setNextReconnectDelay(0);
    
    if (wsRef.current) {
      wsRef.current.close();
    }
    
    connect();
  }, [connect]);

  useEffect(() => {
    connect();

    return () => {
      shouldReconnectRef.current = false;
      
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      
      if (wsRef.current && typeof wsRef.current.close === 'function') {
        wsRef.current.close();
      }
    };
  }, [connect]);

  return {
    connectionState,
    sendMessage,
    disconnect,
    reconnect,
    reconnectAttempt,
    nextReconnectDelay,
  };
};
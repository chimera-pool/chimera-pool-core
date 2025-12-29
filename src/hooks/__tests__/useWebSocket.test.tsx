import { renderHook, act } from '@testing-library/react';
import { useWebSocket } from '../useWebSocket';

// Create a proper mock WebSocket class
class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  
  url: string;
  readyState: number = MockWebSocket.CONNECTING; // CONNECTING
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  
  send = jest.fn();
  close = jest.fn();
  addEventListener = jest.fn((event: string, handler: Function) => {
    if (event === 'open') this.onopen = handler as any;
    if (event === 'close') this.onclose = handler as any;
    if (event === 'message') this.onmessage = handler as any;
    if (event === 'error') this.onerror = handler as any;
  });
  removeEventListener = jest.fn();
  
  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }
  
  // Helper to simulate events
  simulateOpen() {
    this.readyState = 1; // OPEN
    if (this.onopen) this.onopen(new Event('open'));
  }
  
  simulateClose() {
    this.readyState = 3; // CLOSED
    if (this.onclose) this.onclose(new CloseEvent('close'));
  }
  
  simulateMessage(data: string) {
    if (this.onmessage) this.onmessage(new MessageEvent('message', { data }));
  }
  
  simulateError() {
    if (this.onerror) this.onerror(new Event('error'));
  }
}

// Replace global WebSocket
(global as any).WebSocket = MockWebSocket;

describe('useWebSocket', () => {
  beforeEach(() => {
    MockWebSocket.instances = [];
  });

  it('should establish WebSocket connection', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    expect(MockWebSocket.instances.length).toBe(1);
    expect(MockWebSocket.instances[0].url).toBe('ws://localhost:8080');
    expect(result.current.connectionState).toBe('connecting');
  });

  it('should handle connection open event', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    expect(result.current.connectionState).toBe('connected');
  });

  it('should handle incoming messages', () => {
    const onMessage = jest.fn();
    renderHook(() => useWebSocket('ws://localhost:8080', { onMessage }));
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    const mockMessage = JSON.stringify({
      type: 'POOL_STATS_UPDATE',
      payload: { hashrate: '1.5 TH/s', miners: 42 }
    });
    
    act(() => {
      ws.simulateMessage(mockMessage);
    });
    
    expect(onMessage).toHaveBeenCalledWith({
      type: 'POOL_STATS_UPDATE',
      payload: { hashrate: '1.5 TH/s', miners: 42 }
    });
  });

  it('should handle connection errors', () => {
    const onError = jest.fn();
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080', { onError }));
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateError();
    });
    
    expect(result.current.connectionState).toBe('error');
  });

  it('should handle connection close and attempt reconnect', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    act(() => {
      ws.simulateClose();
    });
    
    // Hook attempts reconnection, so state goes to reconnecting
    expect(['disconnected', 'reconnecting']).toContain(result.current.connectionState);
  });

  it('should send messages when connected', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    const message = { type: 'SUBSCRIBE_POOL_STATS' };
    
    act(() => {
      result.current.sendMessage(message);
    });
    
    // The hook checks readyState === WebSocket.OPEN (1)
    // Our mock sets readyState to 1 in simulateOpen
    expect(ws.send).toHaveBeenCalledWith(JSON.stringify(message));
  });

  it('should not send messages when disconnected', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const ws = MockWebSocket.instances[0];
    // readyState is 0 (CONNECTING) by default
    
    const message = { type: 'SUBSCRIBE_POOL_STATS' };
    
    act(() => {
      result.current.sendMessage(message);
    });
    
    expect(ws.send).not.toHaveBeenCalled();
  });

  it('should clean up on unmount', () => {
    const { unmount } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const ws = MockWebSocket.instances[0];
    
    unmount();
    
    expect(ws.close).toHaveBeenCalled();
  });

  it('should handle malformed JSON messages gracefully', () => {
    const onError = jest.fn();
    renderHook(() => useWebSocket('ws://localhost:8080', { onError }));
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    act(() => {
      ws.simulateMessage('invalid json');
    });
    
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
  });

  it('should expose reconnectAttempt and nextReconnectDelay', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    expect(result.current.reconnectAttempt).toBe(0);
    expect(result.current.nextReconnectDelay).toBe(0);
  });

  it('should call onReconnectAttempt callback with attempt number and delay', () => {
    jest.useFakeTimers();
    const onReconnectAttempt = jest.fn();
    const { result } = renderHook(() => 
      useWebSocket('ws://localhost:8080', { 
        onReconnectAttempt,
        reconnectInterval: 1000,
        useExponentialBackoff: true
      })
    );
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    act(() => {
      ws.simulateClose();
    });
    
    // Should call onReconnectAttempt with attempt 1 and calculated delay
    expect(onReconnectAttempt).toHaveBeenCalledWith(1, expect.any(Number));
    expect(result.current.reconnectAttempt).toBe(1);
    expect(result.current.nextReconnectDelay).toBeGreaterThan(0);
    
    jest.useRealTimers();
  });

  it('should use fixed interval when exponential backoff is disabled', () => {
    jest.useFakeTimers();
    const onReconnectAttempt = jest.fn();
    const fixedInterval = 5000;
    
    renderHook(() => 
      useWebSocket('ws://localhost:8080', { 
        onReconnectAttempt,
        reconnectInterval: fixedInterval,
        useExponentialBackoff: false
      })
    );
    
    const ws = MockWebSocket.instances[0];
    
    act(() => {
      ws.simulateOpen();
    });
    
    act(() => {
      ws.simulateClose();
    });
    
    // With exponential backoff disabled, delay should be the fixed interval
    expect(onReconnectAttempt).toHaveBeenCalledWith(1, fixedInterval);
    
    jest.useRealTimers();
  });

  it('should reset reconnect state on successful connection', () => {
    jest.useFakeTimers();
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const ws = MockWebSocket.instances[0];
    
    // Simulate connection and disconnect to trigger reconnect
    act(() => {
      ws.simulateOpen();
    });
    
    act(() => {
      ws.simulateClose();
    });
    
    expect(result.current.reconnectAttempt).toBe(1);
    
    // Fast forward to trigger reconnect
    act(() => {
      jest.advanceTimersByTime(5000);
    });
    
    // Get the new WebSocket instance and simulate open
    const newWs = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    
    act(() => {
      newWs.simulateOpen();
    });
    
    // Reconnect state should be reset
    expect(result.current.reconnectAttempt).toBe(0);
    expect(result.current.nextReconnectDelay).toBe(0);
    
    jest.useRealTimers();
  });
});
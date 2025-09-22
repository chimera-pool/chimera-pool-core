import { renderHook, act } from '@testing-library/react';
import { useWebSocket } from '../useWebSocket';

// Mock WebSocket
const mockWebSocket = {
  send: jest.fn(),
  close: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  readyState: WebSocket.OPEN,
};

global.WebSocket = jest.fn(() => mockWebSocket) as any;

describe('useWebSocket', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should establish WebSocket connection', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    expect(WebSocket).toHaveBeenCalledWith('ws://localhost:8080');
    expect(result.current.connectionState).toBe('connecting');
  });

  it('should handle connection open event', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    // Simulate connection open
    const openHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )[1];
    
    act(() => {
      openHandler();
    });
    
    expect(result.current.connectionState).toBe('connected');
  });

  it('should handle incoming messages', () => {
    const onMessage = jest.fn();
    renderHook(() => useWebSocket('ws://localhost:8080', { onMessage }));
    
    const messageHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'message'
    )[1];
    
    const mockMessage = {
      data: JSON.stringify({
        type: 'POOL_STATS_UPDATE',
        payload: { hashrate: '1.5 TH/s', miners: 42 }
      })
    };
    
    act(() => {
      messageHandler(mockMessage);
    });
    
    expect(onMessage).toHaveBeenCalledWith({
      type: 'POOL_STATS_UPDATE',
      payload: { hashrate: '1.5 TH/s', miners: 42 }
    });
  });

  it('should handle connection errors', () => {
    const onError = jest.fn();
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080', { onError }));
    
    const errorHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'error'
    )[1];
    
    const mockError = new Error('Connection failed');
    
    act(() => {
      errorHandler(mockError);
    });
    
    expect(result.current.connectionState).toBe('error');
    expect(onError).toHaveBeenCalledWith(mockError);
  });

  it('should handle connection close', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const closeHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'close'
    )[1];
    
    act(() => {
      closeHandler();
    });
    
    expect(result.current.connectionState).toBe('disconnected');
  });

  it('should send messages when connected', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    // Simulate connection open
    const openHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )[1];
    
    act(() => {
      openHandler();
    });
    
    const message = { type: 'SUBSCRIBE_POOL_STATS' };
    
    act(() => {
      result.current.sendMessage(message);
    });
    
    expect(mockWebSocket.send).toHaveBeenCalledWith(JSON.stringify(message));
  });

  it('should not send messages when disconnected', () => {
    const { result } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    const message = { type: 'SUBSCRIBE_POOL_STATS' };
    
    act(() => {
      result.current.sendMessage(message);
    });
    
    expect(mockWebSocket.send).not.toHaveBeenCalled();
  });

  it('should attempt reconnection on connection loss', () => {
    jest.useFakeTimers();
    
    const { result } = renderHook(() => 
      useWebSocket('ws://localhost:8080', { reconnectInterval: 1000 })
    );
    
    const closeHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'close'
    )[1];
    
    act(() => {
      closeHandler();
    });
    
    expect(result.current.connectionState).toBe('disconnected');
    
    act(() => {
      jest.advanceTimersByTime(1000);
    });
    
    expect(result.current.connectionState).toBe('reconnecting');
    
    jest.useRealTimers();
  });

  it('should clean up on unmount', () => {
    const { unmount } = renderHook(() => useWebSocket('ws://localhost:8080'));
    
    unmount();
    
    expect(mockWebSocket.close).toHaveBeenCalled();
  });

  it('should handle malformed JSON messages gracefully', () => {
    const onError = jest.fn();
    renderHook(() => useWebSocket('ws://localhost:8080', { onError }));
    
    const messageHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'message'
    )[1];
    
    const malformedMessage = { data: 'invalid json' };
    
    act(() => {
      messageHandler(malformedMessage);
    });
    
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
  });
});
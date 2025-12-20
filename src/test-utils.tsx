import React, { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';

// Mock providers for testing (simplified - no router dependency)
const MockProviders: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return <>{children}</>;
};

// Custom render function that includes providers
const customRender = (
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) => render(ui, { wrapper: MockProviders, ...options });

// Re-export everything
export * from '@testing-library/react';

// Override render method
export { customRender as render };

// Test utilities for common patterns
export const createMockWebSocket = () => ({
  close: jest.fn(),
  send: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  readyState: WebSocket.OPEN,
});

export function createMockApiResponse<T>(data: T, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  };
}

export const waitForLoadingToFinish = () => {
  return new Promise(resolve => setTimeout(resolve, 0));
};

// Mock data generators
export const generateMockMinerStats = () => {
  return {
    id: 'miner-' + Math.random().toString(36).substr(2, 9),
    name: 'Test Miner',
    hashrate: Math.floor(Math.random() * 1000000),
    shares: Math.floor(Math.random() * 1000),
    lastSeen: new Date().toISOString(),
    status: 'active' as const,
  };
};

export const generateMockPoolStats = () => {
  return {
    totalHashrate: Math.floor(Math.random() * 10000000),
    activeMiners: Math.floor(Math.random() * 100),
    blocksFound: Math.floor(Math.random() * 50),
    totalShares: Math.floor(Math.random() * 100000),
    networkDifficulty: Math.floor(Math.random() * 1000000),
    lastBlockTime: new Date().toISOString(),
  };
};
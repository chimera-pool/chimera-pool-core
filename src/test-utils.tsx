import React, { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';

// Mock providers for testing
const MockProviders: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <BrowserRouter>
      {children}
    </BrowserRouter>
  );
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

export const createMockApiResponse = <T>(data: T, status = 200) => ({
  ok: status >= 200 && status < 300,
  status,
  json: () => Promise.resolve(data),
  text: () => Promise.resolve(JSON.stringify(data)),
});

export const waitForLoadingToFinish = () => 
  new Promise(resolve => setTimeout(resolve, 0));

// Mock data generators
export const generateMockMinerStats = () => ({
  id: 'miner-' + Math.random().toString(36).substr(2, 9),
  name: 'Test Miner',
  hashrate: Math.floor(Math.random() * 1000000),
  shares: Math.floor(Math.random() * 1000),
  lastSeen: new Date().toISOString(),
  status: 'active' as const,
});

export const generateMockPoolStats = () => ({
  totalHashrate: Math.floor(Math.random() * 10000000),
  activeMiners: Math.floor(Math.random() * 100),
  blocksFound: Math.floor(Math.random() * 50),
  totalShares: Math.floor(Math.random() * 100000),
  networkDifficulty: Math.floor(Math.random() * 1000000),
  lastBlockTime: new Date().toISOString(),
});
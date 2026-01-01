import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { WalletManager, UserWallet, WalletSummary } from './WalletManager';

// Mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

const mockShowMessage = jest.fn();

const mockWallets: UserWallet[] = [
  {
    id: 1,
    address: 'ltc1qtest123456789abcdef',
    label: 'Main Wallet',
    percentage: 60,
    is_primary: true,
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    address: 'ltc1qtest987654321fedcba',
    label: 'Secondary',
    percentage: 40,
    is_primary: false,
    is_active: true,
    created_at: '2024-01-02T00:00:00Z',
  },
];

const mockSummary: WalletSummary = {
  total_wallets: 2,
  active_wallets: 2,
  total_percentage: 100,
  remaining_percentage: 0,
  has_primary_wallet: true,
};

describe('WalletManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ wallets: mockWallets, summary: mockSummary }),
    });
  });

  test('renders loading state initially', () => {
    render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
    expect(screen.getByText(/Loading wallet settings/i)).toBeInTheDocument();
  });

  test('renders wallet list after loading', async () => {
    render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
    
    await waitFor(() => {
      expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
    }, { timeout: 3000 });
    
    // Use getAllByText since there may be multiple instances
    expect(screen.getAllByText('Main Wallet').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Secondary').length).toBeGreaterThan(0);
  });

  test('handles empty wallet list', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ wallets: [], summary: null }),
    });

    render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
    
    await waitFor(() => {
      expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
    }, { timeout: 3000 });
    
    expect(screen.getByText(/No wallets configured yet/i)).toBeInTheDocument();
  });

  test('shows add wallet button after loading', async () => {
    render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
    
    await waitFor(() => {
      expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
    }, { timeout: 3000 });
    
    expect(screen.getByText('+ Add Wallet')).toBeInTheDocument();
  });

  test('displays summary information after loading', async () => {
    render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
    
    await waitFor(() => {
      expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
    }, { timeout: 3000 });
    
    expect(screen.getByText('Active Wallets')).toBeInTheDocument();
  });

  test('shows toggle switches for wallets after loading', async () => {
    render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
    
    await waitFor(() => {
      expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
    }, { timeout: 3000 });
    
    expect(screen.getByTestId('wallet-toggle-1')).toBeInTheDocument();
    expect(screen.getByTestId('wallet-toggle-2')).toBeInTheDocument();
  });
});

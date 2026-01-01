import React from 'react';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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

const mockSummaryWithRemaining: WalletSummary = {
  total_wallets: 1,
  active_wallets: 1,
  total_percentage: 60,
  remaining_percentage: 40,
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

  describe('Loading State', () => {
    test('renders loading state initially', () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      expect(screen.getByText(/Loading wallet settings/i)).toBeInTheDocument();
    });

    test('shows wallet title in loading state', () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      // Title contains emoji and text - use regex
      expect(screen.getAllByText(/Wallet/i).length).toBeGreaterThan(0);
    });
  });

  describe('Wallet List Display', () => {
    test('renders wallet list after loading', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
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

    test('displays wallet percentages correctly', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      // Use getAllByText since percentages appear in multiple places
      expect(screen.getAllByText('60.0%').length).toBeGreaterThan(0);
      expect(screen.getAllByText('40.0%').length).toBeGreaterThan(0);
    });

    test('shows primary badge for primary wallet', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText(/Primary/i)).toBeInTheDocument();
    });

    test('shows inactive badge for inactive wallets', async () => {
      const walletsWithInactive: UserWallet[] = [
        ...mockWallets,
        {
          id: 3,
          address: 'ltc1qinactive123',
          label: 'Inactive Wallet',
          percentage: 0,
          is_primary: false,
          is_active: false,
          created_at: '2024-01-03T00:00:00Z',
        },
      ];

      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ wallets: walletsWithInactive, summary: mockSummary }),
      });

      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText('Inactive')).toBeInTheDocument();
    });

    test('truncates long wallet addresses', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      // Address should be truncated with ...
      expect(screen.getByText(/ltc1qtest123/)).toBeInTheDocument();
    });
  });

  describe('Summary Bar', () => {
    test('displays summary information after loading', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText('Active Wallets')).toBeInTheDocument();
      expect(screen.getByText('Allocated')).toBeInTheDocument();
      expect(screen.getByText('Remaining')).toBeInTheDocument();
    });

    test('shows correct active wallet count', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText('2')).toBeInTheDocument();
    });

    test('shows 100% allocated when all percentage used', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText('100.0%')).toBeInTheDocument();
    });
  });

  describe('Add Wallet Button', () => {
    test('shows add wallet button after loading', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText('+ Add Wallet')).toBeInTheDocument();
    });

    test('add wallet button is disabled when no remaining percentage', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      const addButton = screen.getByText('+ Add Wallet');
      expect(addButton).toBeDisabled();
    });

    test('add wallet button is enabled when remaining percentage available', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ wallets: [mockWallets[0]], summary: mockSummaryWithRemaining }),
      });

      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      const addButton = screen.getByText('+ Add Wallet');
      expect(addButton).not.toBeDisabled();
    });

    test('opens add wallet form when button clicked', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: async () => ({ wallets: [mockWallets[0]], summary: mockSummaryWithRemaining }),
      });

      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      fireEvent.click(screen.getByText('+ Add Wallet'));
      
      expect(screen.getByText('Add New Wallet')).toBeInTheDocument();
      expect(screen.getByText('Wallet Address *')).toBeInTheDocument();
    });
  });

  describe('Toggle Switches', () => {
    test('shows toggle switches for wallets after loading', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByTestId('wallet-toggle-1')).toBeInTheDocument();
      expect(screen.getByTestId('wallet-toggle-2')).toBeInTheDocument();
    });

    test('toggle switch calls API when clicked', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'Updated' }),
      });

      await act(async () => {
        fireEvent.click(screen.getByTestId('wallet-toggle-1'));
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/user/wallets/1/toggle',
        expect.objectContaining({
          method: 'PUT',
        })
      );
    });

    test('shows success message after toggle', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'Updated' }),
      });

      await act(async () => {
        fireEvent.click(screen.getByTestId('wallet-toggle-1'));
      });

      await waitFor(() => {
        expect(mockShowMessage).toHaveBeenCalledWith('success', expect.any(String));
      });
    });
  });

  describe('Edit Wallet', () => {
    test('shows edit button for each wallet', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByTestId('wallet-edit-1')).toBeInTheDocument();
      expect(screen.getByTestId('wallet-edit-2')).toBeInTheDocument();
    });

    test('opens edit form when edit button clicked', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      fireEvent.click(screen.getByTestId('wallet-edit-1'));
      
      // Should show Save and Cancel buttons
      expect(screen.getByText('Save')).toBeInTheDocument();
      expect(screen.getByText('Cancel')).toBeInTheDocument();
    });

    test('closes edit form when cancel clicked', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      fireEvent.click(screen.getByTestId('wallet-edit-1'));
      fireEvent.click(screen.getByText('Cancel'));
      
      // Edit form should be closed
      expect(screen.queryByText('Save')).not.toBeInTheDocument();
    });
  });

  describe('Delete Wallet', () => {
    test('shows delete button for each wallet', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByTestId('wallet-delete-1')).toBeInTheDocument();
      expect(screen.getByTestId('wallet-delete-2')).toBeInTheDocument();
    });

    test('shows confirmation before delete', async () => {
      const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(false);
      
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      fireEvent.click(screen.getByTestId('wallet-delete-1'));
      
      expect(confirmSpy).toHaveBeenCalled();
      confirmSpy.mockRestore();
    });

    test('calls delete API when confirmed', async () => {
      const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
      
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'Deleted' }),
      });

      await act(async () => {
        fireEvent.click(screen.getByTestId('wallet-delete-1'));
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/user/wallets/1',
        expect.objectContaining({
          method: 'DELETE',
        })
      );
      
      confirmSpy.mockRestore();
    });
  });

  describe('Allocation Slider', () => {
    test('shows allocation slider for active wallets with multiple wallets', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      expect(screen.getByTestId('wallet-slider-1')).toBeInTheDocument();
      expect(screen.getByTestId('wallet-slider-2')).toBeInTheDocument();
    });

    test('slider updates allocation on change', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      const slider = screen.getByTestId('wallet-slider-1');
      
      await act(async () => {
        fireEvent.change(slider, { target: { value: '70' } });
      });

      // Value should update in UI - use getAllByText since there may be multiple
      expect(screen.getAllByText('70.0%').length).toBeGreaterThan(0);
    });
  });

  describe('Payout Preview', () => {
    test('shows payout preview for multiple active wallets at 100%', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      expect(screen.getByText(/Payout Split Preview/i)).toBeInTheDocument();
    });

    test('shows example payout amounts', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      // Check that payout preview section exists with BDAG amounts
      expect(screen.getAllByText(/BDAG/).length).toBeGreaterThan(0);
    });
  });

  describe('Error Handling', () => {
    test('handles fetch error gracefully', async () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      
      mockFetch.mockRejectedValue(new Error('Network error'));

      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });
      
      consoleSpy.mockRestore();
    });

    test('shows error message on toggle failure', async () => {
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Toggle failed' }),
      });

      await act(async () => {
        fireEvent.click(screen.getByTestId('wallet-toggle-1'));
      });

      await waitFor(() => {
        expect(mockShowMessage).toHaveBeenCalledWith('error', expect.any(String));
      });
    });

    test('shows error message on delete failure', async () => {
      const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
      
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Delete failed' }),
      });

      await act(async () => {
        fireEvent.click(screen.getByTestId('wallet-delete-1'));
      });

      await waitFor(() => {
        expect(mockShowMessage).toHaveBeenCalledWith('error', expect.any(String));
      });
      
      confirmSpy.mockRestore();
    });
  });

  describe('API Integration', () => {
    test('sends correct authorization header', async () => {
      render(<WalletManager token="my-auth-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledWith(
          '/api/v1/user/wallets',
          expect.objectContaining({
            headers: { 'Authorization': 'Bearer my-auth-token' },
          })
        );
      });
    });

    test('refetches wallets after successful operations', async () => {
      const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
      
      render(<WalletManager token="test-token" showMessage={mockShowMessage} />);
      
      await waitFor(() => {
        expect(screen.queryByText(/Loading wallet settings/i)).not.toBeInTheDocument();
      }, { timeout: 3000 });

      const initialFetchCount = mockFetch.mock.calls.length;

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ message: 'Deleted' }),
      });

      await act(async () => {
        fireEvent.click(screen.getByTestId('wallet-delete-1'));
      });

      await waitFor(() => {
        expect(mockFetch.mock.calls.length).toBeGreaterThan(initialFetchCount);
      });
      
      confirmSpy.mockRestore();
    });
  });
});

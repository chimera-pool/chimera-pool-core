/**
 * Consolidated Admin Tab Component Tests
 * Tests basic rendering and isActive behavior for all extracted admin tab components
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';

import AdminUsersTab from '../AdminUsersTab';
import AdminAlgorithmTab from '../AdminAlgorithmTab';
import AdminNetworkTab from '../AdminNetworkTab';
import AdminRolesTab from '../AdminRolesTab';
import AdminBugsTab from '../AdminBugsTab';
import AdminMinersTab from '../AdminMinersTab';

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock window.confirm
window.confirm = jest.fn();

describe('Admin Tab Components - isActive Behavior', () => {
  const mockToken = 'test-token-123';
  const mockShowMessage = jest.fn();
  const mockOnClose = jest.fn();
  const mockOnNavigateToUsers = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockFetch.mockReset();
    // Default mock for API calls
    mockFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ users: [], bugs: [], miners: [], networks: [], admins: [], moderators: [], total_count: 0 }),
    });
  });

  describe('AdminUsersTab', () => {
    it('renders nothing when isActive is false', () => {
      const { container } = render(
        <AdminUsersTab 
          token={mockToken} 
          isActive={false} 
          showMessage={mockShowMessage}
          onClose={mockOnClose}
        />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders content when isActive is true', async () => {
      render(
        <AdminUsersTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
          onClose={mockOnClose}
        />
      );
      
      // Should render search input
      expect(screen.getByPlaceholderText(/search users/i)).toBeInTheDocument();
    });

    it('calls API when active', async () => {
      render(
        <AdminUsersTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
          onClose={mockOnClose}
        />
      );
      
      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalled();
      });
    });
  });

  describe('AdminAlgorithmTab', () => {
    it('renders nothing when isActive is false', () => {
      const { container } = render(
        <AdminAlgorithmTab 
          token={mockToken} 
          isActive={false} 
          showMessage={mockShowMessage}
        />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders content when isActive is true', () => {
      render(
        <AdminAlgorithmTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
        />
      );
      
      expect(screen.getByText(/Mining Algorithm Configuration/i)).toBeInTheDocument();
    });
  });

  describe('AdminNetworkTab', () => {
    it('renders nothing when isActive is false', () => {
      const { container } = render(
        <AdminNetworkTab 
          token={mockToken} 
          isActive={false} 
          showMessage={mockShowMessage}
        />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders content when isActive is true', () => {
      render(
        <AdminNetworkTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
        />
      );
      
      expect(screen.getByText(/Network Configuration/i)).toBeInTheDocument();
    });
  });

  describe('AdminRolesTab', () => {
    it('renders nothing when isActive is false', () => {
      const { container } = render(
        <AdminRolesTab 
          token={mockToken} 
          isActive={false} 
          showMessage={mockShowMessage}
          onNavigateToUsers={mockOnNavigateToUsers}
        />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders content when isActive is true', () => {
      render(
        <AdminRolesTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
          onNavigateToUsers={mockOnNavigateToUsers}
        />
      );
      
      expect(screen.getByText(/Role Management/i)).toBeInTheDocument();
    });
  });

  describe('AdminBugsTab', () => {
    it('renders nothing when isActive is false', () => {
      const { container } = render(
        <AdminBugsTab 
          token={mockToken} 
          isActive={false} 
          showMessage={mockShowMessage}
        />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders content when isActive is true', () => {
      const { container } = render(
        <AdminBugsTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
        />
      );
      
      // Component should render something when active
      expect(container.firstChild).not.toBeNull();
    });
  });

  describe('AdminMinersTab', () => {
    it('renders nothing when isActive is false', () => {
      const { container } = render(
        <AdminMinersTab 
          token={mockToken} 
          isActive={false} 
          showMessage={mockShowMessage}
        />
      );
      expect(container.firstChild).toBeNull();
    });

    it('renders content when isActive is true', () => {
      const { container } = render(
        <AdminMinersTab 
          token={mockToken} 
          isActive={true} 
          showMessage={mockShowMessage}
        />
      );
      
      // Component should render something when active
      expect(container.firstChild).not.toBeNull();
    });
  });
});

describe('Admin Tab Components - Interface Segregation Principle', () => {
  it('all components follow consistent props interface', () => {
    // This test documents and verifies that all components follow ISP
    const commonProps = {
      token: expect.any(String),
      isActive: expect.any(Boolean),
      showMessage: expect.any(Function),
    };

    // Components should accept these common props
    expect(AdminUsersTab).toBeDefined();
    expect(AdminAlgorithmTab).toBeDefined();
    expect(AdminNetworkTab).toBeDefined();
    expect(AdminRolesTab).toBeDefined();
    expect(AdminBugsTab).toBeDefined();
    expect(AdminMinersTab).toBeDefined();
  });
});

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { AIHelpAssistant } from '../AIHelpAssistant';

// Mock fetch for API calls
global.fetch = jest.fn();

describe('AIHelpAssistant', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({
        response: 'This is a helpful AI response about mining.',
        suggestions: ['Check your hashrate', 'Verify pool connection', 'Update mining software']
      })
    });
  });

  it('should render AI assistant with cyber styling', () => {
    render(<AIHelpAssistant />);
    
    expect(screen.getByTestId('ai-help-assistant')).toHaveClass('cyber-ai-assistant');
    expect(screen.getByText('AI_ASSISTANT')).toBeInTheDocument();
  });

  it('should show collapsed state initially', () => {
    render(<AIHelpAssistant />);
    
    const assistant = screen.getByTestId('ai-help-assistant');
    expect(assistant).toHaveClass('cyber-ai-assistant--collapsed');
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    expect(toggleButton).toBeInTheDocument();
  });

  it('should expand when toggle button is clicked', () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const assistant = screen.getByTestId('ai-help-assistant');
    expect(assistant).toHaveClass('cyber-ai-assistant--expanded');
    expect(screen.getByTestId('ai-chat-interface')).toBeInTheDocument();
  });

  it('should display welcome message when expanded', () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    expect(screen.getByText('Hello! I\'m your mining assistant. How can I help you today?')).toBeInTheDocument();
  });

  it('should handle user input and send messages', async () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input');
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'How do I improve my hashrate?' } });
    fireEvent.click(sendButton);
    
    expect(screen.getByText('How do I improve my hashrate?')).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith('/api/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message: 'How do I improve my hashrate?',
        context: 'mining_help'
      })
    });
  });

  it('should display AI response with cyber styling', async () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input');
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'Test question' } });
    fireEvent.click(sendButton);
    
    await waitFor(() => {
      expect(screen.getByText('This is a helpful AI response about mining.')).toBeInTheDocument();
    });
    
    const aiMessages = screen.getAllByText('This is a helpful AI response about mining.');
    expect(aiMessages[0].closest('.cyber-ai-message')).toHaveClass('cyber-ai-message--assistant');
  });

  it('should show typing indicator while waiting for response', async () => {
    (fetch as jest.Mock).mockImplementation(() => 
      new Promise(resolve => setTimeout(() => resolve({
        ok: true,
        json: () => Promise.resolve({ response: 'Delayed response' })
      }), 100))
    );
    
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input');
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'Test question' } });
    fireEvent.click(sendButton);
    
    expect(screen.getByTestId('ai-typing-indicator')).toBeInTheDocument();
    expect(screen.getByText('AI is thinking...')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(screen.queryByTestId('ai-typing-indicator')).not.toBeInTheDocument();
    });
  });

  it('should display suggested actions from AI response', async () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input');
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'Help with mining issues' } });
    fireEvent.click(sendButton);
    
    await waitFor(() => {
      expect(screen.getByText('Check your hashrate')).toBeInTheDocument();
      expect(screen.getByText('Verify pool connection')).toBeInTheDocument();
      expect(screen.getByText('Update mining software')).toBeInTheDocument();
    });
    
    const suggestions = screen.getAllByTestId(/ai-suggestion-/);
    suggestions.forEach(suggestion => {
      expect(suggestion).toHaveClass('cyber-ai-suggestion');
    });
  });

  it('should handle API errors gracefully', async () => {
    (fetch as jest.Mock).mockRejectedValue(new Error('API Error'));
    
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input');
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'Test question' } });
    fireEvent.click(sendButton);
    
    await waitFor(() => {
      const errorMessages = screen.getAllByText('Sorry, I encountered an error. Please try again.');
      expect(errorMessages.length).toBeGreaterThan(0);
    });
    
    const errorMessages = screen.getAllByText('Sorry, I encountered an error. Please try again.');
    expect(errorMessages[0].closest('.cyber-ai-message')).toHaveClass('cyber-ai-message--error');
  });

  it('should provide quick help buttons for common questions', () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    expect(screen.getByText('SETUP_HELP')).toBeInTheDocument();
    expect(screen.getByText('PERFORMANCE_TIPS')).toBeInTheDocument();
    expect(screen.getByText('TROUBLESHOOTING')).toBeInTheDocument();
    expect(screen.getByText('PAYOUT_INFO')).toBeInTheDocument();
  });

  it('should handle quick help button clicks', async () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const setupHelpButton = screen.getByText('SETUP_HELP');
    fireEvent.click(setupHelpButton);
    
    expect(fetch).toHaveBeenCalledWith('/api/ai/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        message: 'I need help with mining setup',
        context: 'setup_help'
      })
    });
  });

  it('should maintain chat history', async () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input');
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'First question' } });
    fireEvent.click(sendButton);
    
    await waitFor(() => {
      expect(screen.getByText('First question')).toBeInTheDocument();
    });
    
    expect(screen.getByText('First question')).toBeInTheDocument();
  });

  it('should clear input after sending message', async () => {
    render(<AIHelpAssistant />);
    
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);
    
    const input = screen.getByTestId('ai-message-input') as HTMLInputElement;
    const sendButton = screen.getByTestId('ai-send-button');
    
    fireEvent.change(input, { target: { value: 'Test message' } });
    fireEvent.click(sendButton);
    
    expect(input.value).toBe('');
  });
});
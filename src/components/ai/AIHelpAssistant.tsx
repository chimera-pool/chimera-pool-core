import React, { useState, useRef, useEffect } from 'react';
import './AIHelpAssistant.css';

export interface ChatMessage {
  id: string;
  type: 'user' | 'assistant' | 'error';
  content: string;
  timestamp: Date;
  suggestions?: string[];
}

export interface AIHelpAssistantProps {
  className?: string;
}

export const AIHelpAssistant: React.FC<AIHelpAssistantProps> = ({
  className = '',
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    if (messagesEndRef.current && messagesEndRef.current.scrollIntoView) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  useEffect(() => {
    if (isExpanded && messages.length === 0) {
      // Add welcome message when first expanded
      setMessages([{
        id: 'welcome',
        type: 'assistant',
        content: 'Hello! I\'m your mining assistant. How can I help you today?',
        timestamp: new Date(),
      }]);
    }
  }, [isExpanded, messages.length]);

  const sendMessage = async (content: string, context: string = 'mining_help') => {
    if (!content.trim()) return;

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      type: 'user',
      content,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInputValue('');
    setIsTyping(true);

    try {
      const response = await fetch('/api/ai/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message: content,
          context,
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to get AI response');
      }

      const assistantMessage: ChatMessage = {
        id: `assistant-${Date.now()}`,
        type: 'assistant',
        content: data.response,
        timestamp: new Date(),
        suggestions: data.suggestions,
      };

      setMessages(prev => [...prev, assistantMessage]);
    } catch (error) {
      const errorMessage: ChatMessage = {
        id: `error-${Date.now()}`,
        type: 'error',
        content: 'Sorry, I encountered an error. Please try again.',
        timestamp: new Date(),
      };

      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsTyping(false);
    }
  };

  const handleQuickHelp = (topic: string) => {
    const quickHelpMessages = {
      'SETUP_HELP': 'I need help with mining setup',
      'PERFORMANCE_TIPS': 'How can I improve my mining performance?',
      'TROUBLESHOOTING': 'I\'m having issues with my mining setup',
      'PAYOUT_INFO': 'How do payouts work in this pool?',
    };

    const message = quickHelpMessages[topic as keyof typeof quickHelpMessages];
    const context = topic.toLowerCase().replace('_', '_');
    
    if (message) {
      sendMessage(message, context);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    sendMessage(inputValue);
  };

  const assistantClasses = [
    'cyber-ai-assistant',
    isExpanded ? 'cyber-ai-assistant--expanded' : 'cyber-ai-assistant--collapsed',
    className,
  ].filter(Boolean).join(' ');

  return (
    <div className={assistantClasses} data-testid="ai-help-assistant">
      <button
        className="cyber-ai-assistant__toggle"
        onClick={() => setIsExpanded(!isExpanded)}
        data-testid="ai-assistant-toggle"
      >
        <span className="cyber-ai-assistant__toggle-icon">🤖</span>
        <span className="cyber-ai-assistant__toggle-text">
          {isExpanded ? 'MINIMIZE' : 'AI_ASSISTANT'}
        </span>
      </button>

      {isExpanded && (
        <div className="cyber-ai-assistant__content" data-testid="ai-chat-interface">
          <div className="cyber-ai-assistant__header">
            <h3 className="cyber-ai-assistant__title">AI_ASSISTANT</h3>
            <div className="cyber-ai-assistant__status">
              <div className="cyber-status-indicator cyber-pulse" />
              ONLINE
            </div>
          </div>

          <div className="cyber-ai-assistant__messages">
            {messages.map(message => (
              <div
                key={message.id}
                className={`cyber-ai-message cyber-ai-message--${message.type}`}
                data-testid={message.type === 'assistant' ? `ai-message-response-${message.id}` : message.type === 'error' ? 'ai-error-message' : undefined}
              >
                <div className="cyber-ai-message__content">
                  {message.content}
                </div>
                
                {message.suggestions && (
                  <div className="cyber-ai-message__suggestions">
                    {message.suggestions.map((suggestion, index) => (
                      <button
                        key={index}
                        className="cyber-ai-suggestion"
                        onClick={() => sendMessage(suggestion)}
                        data-testid={`ai-suggestion-${index}`}
                      >
                        {suggestion}
                      </button>
                    ))}
                  </div>
                )}
                
                <div className="cyber-ai-message__timestamp">
                  {message.timestamp.toLocaleTimeString()}
                </div>
              </div>
            ))}

            {isTyping && (
              <div className="cyber-ai-typing" data-testid="ai-typing-indicator">
                <div className="cyber-ai-typing__content">
                  <div className="cyber-ai-typing__dots">
                    <span></span>
                    <span></span>
                    <span></span>
                  </div>
                  <span className="cyber-ai-typing__text">AI is thinking...</span>
                </div>
              </div>
            )}



            <div ref={messagesEndRef} />
          </div>

          <div className="cyber-ai-assistant__quick-help">
            <div className="cyber-quick-help__title">QUICK_HELP:</div>
            <div className="cyber-quick-help__buttons">
              <button
                className="cyber-quick-help__button"
                onClick={() => handleQuickHelp('SETUP_HELP')}
              >
                SETUP_HELP
              </button>
              <button
                className="cyber-quick-help__button"
                onClick={() => handleQuickHelp('PERFORMANCE_TIPS')}
              >
                PERFORMANCE_TIPS
              </button>
              <button
                className="cyber-quick-help__button"
                onClick={() => handleQuickHelp('TROUBLESHOOTING')}
              >
                TROUBLESHOOTING
              </button>
              <button
                className="cyber-quick-help__button"
                onClick={() => handleQuickHelp('PAYOUT_INFO')}
              >
                PAYOUT_INFO
              </button>
            </div>
          </div>

          <form className="cyber-ai-assistant__input" onSubmit={handleSubmit}>
            <input
              type="text"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              placeholder="Ask me anything about mining..."
              className="cyber-ai-input"
              data-testid="ai-message-input"
            />
            <button
              type="submit"
              className="cyber-ai-send-button"
              disabled={!inputValue.trim() || isTyping}
              data-testid="ai-send-button"
            >
              SEND
            </button>
          </form>
        </div>
      )}
    </div>
  );
};
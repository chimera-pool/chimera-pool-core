// ============================================================================
// LOGGER UTILITY TESTS
// TDD tests for structured logging system
// ============================================================================

import { logger } from '../logger';

describe('Logger Utility', () => {
  let consoleSpy: {
    debug: jest.SpyInstance;
    info: jest.SpyInstance;
    warn: jest.SpyInstance;
    error: jest.SpyInstance;
  };

  beforeEach(() => {
    consoleSpy = {
      debug: jest.spyOn(console, 'debug').mockImplementation(),
      info: jest.spyOn(console, 'info').mockImplementation(),
      warn: jest.spyOn(console, 'warn').mockImplementation(),
      error: jest.spyOn(console, 'error').mockImplementation(),
    };
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Basic Logging', () => {
    it('should log info messages', () => {
      logger.info('Test info message');
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('INFO');
      expect(consoleSpy.info.mock.calls[0][0]).toContain('Test info message');
    });

    it('should log warn messages', () => {
      logger.warn('Test warning');
      expect(consoleSpy.warn).toHaveBeenCalled();
      expect(consoleSpy.warn.mock.calls[0][0]).toContain('WARN');
    });

    it('should log error messages', () => {
      logger.error('Test error');
      expect(consoleSpy.error).toHaveBeenCalled();
      expect(consoleSpy.error.mock.calls[0][0]).toContain('ERROR');
    });
  });

  describe('Context Logging', () => {
    it('should include context in log output', () => {
      logger.info('Action performed', { component: 'TestComponent', action: 'click' });
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('TestComponent');
    });

    it('should include userId in context', () => {
      logger.info('User action', { userId: 'user-123' });
      expect(consoleSpy.info.mock.calls[0][0]).toContain('user-123');
    });
  });

  describe('Domain-Specific Loggers', () => {
    it('should log API messages with component tag', () => {
      logger.api('API request made', { endpoint: '/test' });
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('API');
    });

    it('should log auth messages with component tag', () => {
      logger.auth('User logged in');
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('Auth');
    });

    it('should log mining messages with component tag', () => {
      logger.mining('New block found');
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('Mining');
    });

    it('should log blockchain messages with component tag', () => {
      logger.blockchain('Block synced');
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('Blockchain');
    });

    it('should log websocket messages with component tag', () => {
      logger.websocket('Connection established');
      expect(consoleSpy.info).toHaveBeenCalled();
      expect(consoleSpy.info.mock.calls[0][0]).toContain('WebSocket');
    });
  });

  describe('Error Logging', () => {
    it('should log errors with stack trace', () => {
      const testError = new Error('Test error');
      logger.logError(testError, { component: 'TestComponent' });
      expect(consoleSpy.error).toHaveBeenCalled();
      expect(consoleSpy.error.mock.calls[0][0]).toContain('Test error');
    });
  });

  describe('Log Level Configuration', () => {
    it('should respect minimum log level', () => {
      logger.setLevel('error');
      logger.info('This should not appear');
      logger.error('This should appear');
      
      expect(consoleSpy.info).not.toHaveBeenCalled();
      expect(consoleSpy.error).toHaveBeenCalled();
      
      // Reset to default
      logger.setLevel('info');
    });
  });

  describe('Timestamp Format', () => {
    it('should include ISO timestamp in log output', () => {
      logger.info('Timestamp test');
      const logOutput = consoleSpy.info.mock.calls[0][0];
      // ISO format: YYYY-MM-DDTHH:mm:ss.sssZ
      expect(logOutput).toMatch(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    });
  });
});

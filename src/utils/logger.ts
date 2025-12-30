// ============================================================================
// STRUCTURED LOGGER UTILITY
// Elite logging system following global coding standards
// Use this instead of console.log for all logging needs
// ============================================================================

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

interface LogContext {
  component?: string;
  action?: string;
  userId?: string;
  [key: string]: unknown;
}

interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: LogContext;
}

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

class Logger {
  private minLevel: LogLevel = 'info';
  private isDevelopment: boolean;

  constructor() {
    this.isDevelopment = process.env.NODE_ENV === 'development';
    this.minLevel = this.isDevelopment ? 'debug' : 'info';
  }

  private shouldLog(level: LogLevel): boolean {
    return LOG_LEVELS[level] >= LOG_LEVELS[this.minLevel];
  }

  private formatEntry(entry: LogEntry): string {
    const contextStr = entry.context 
      ? ` | ${JSON.stringify(entry.context)}`
      : '';
    return `[${entry.timestamp}] [${entry.level.toUpperCase()}] ${entry.message}${contextStr}`;
  }

  private log(level: LogLevel, message: string, context?: LogContext): void {
    if (!this.shouldLog(level)) return;

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message,
      context,
    };

    const formatted = this.formatEntry(entry);

    switch (level) {
      case 'debug':
        // eslint-disable-next-line no-console
        console.debug(formatted);
        break;
      case 'info':
        // eslint-disable-next-line no-console
        console.info(formatted);
        break;
      case 'warn':
        // eslint-disable-next-line no-console
        console.warn(formatted);
        break;
      case 'error':
        // eslint-disable-next-line no-console
        console.error(formatted);
        break;
    }
  }

  debug(message: string, context?: LogContext): void {
    this.log('debug', message, context);
  }

  info(message: string, context?: LogContext): void {
    this.log('info', message, context);
  }

  warn(message: string, context?: LogContext): void {
    this.log('warn', message, context);
  }

  error(message: string, context?: LogContext): void {
    this.log('error', message, context);
  }

  // Domain-specific loggers for easy identification
  api(message: string, context?: LogContext): void {
    this.log('info', message, { ...context, component: 'API' });
  }

  auth(message: string, context?: LogContext): void {
    this.log('info', message, { ...context, component: 'Auth' });
  }

  mining(message: string, context?: LogContext): void {
    this.log('info', message, { ...context, component: 'Mining' });
  }

  blockchain(message: string, context?: LogContext): void {
    this.log('info', message, { ...context, component: 'Blockchain' });
  }

  websocket(message: string, context?: LogContext): void {
    this.log('info', message, { ...context, component: 'WebSocket' });
  }

  ui(message: string, context?: LogContext): void {
    this.log('debug', message, { ...context, component: 'UI' });
  }

  performance(message: string, context?: LogContext): void {
    this.log('debug', message, { ...context, component: 'Performance' });
  }

  // Error logging with stack trace
  logError(error: Error, context?: LogContext): void {
    this.error(error.message, {
      ...context,
      stack: error.stack,
      name: error.name,
    });
  }

  // Set minimum log level
  setLevel(level: LogLevel): void {
    this.minLevel = level;
  }
}

// Singleton instance
export const logger = new Logger();

// Named exports for convenience
export const { debug, info, warn, error } = {
  debug: logger.debug.bind(logger),
  info: logger.info.bind(logger),
  warn: logger.warn.bind(logger),
  error: logger.error.bind(logger),
};

export default logger;

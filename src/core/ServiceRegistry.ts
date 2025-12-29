// ============================================================================
// SERVICE REGISTRY
// Dependency injection container for modular, testable architecture
// Following Interface Segregation Principle
// ============================================================================

type ServiceFactory<T> = () => T;
type ServiceInstance = any;

interface ServiceDescriptor<T> {
  factory: ServiceFactory<T>;
  singleton: boolean;
  instance?: T;
}

/**
 * Service Registry - Central dependency injection container
 * 
 * Benefits:
 * - Loose coupling between modules
 * - Easy mocking for tests
 * - Centralized service management
 * - Lazy initialization
 * - Singleton or transient services
 */
class ServiceRegistryClass {
  private services = new Map<string, ServiceDescriptor<any>>();
  private initialized = false;

  /**
   * Register a service factory
   * @param name - Unique service identifier
   * @param factory - Function that creates the service
   * @param singleton - If true, only one instance is created
   */
  register<T>(name: string, factory: ServiceFactory<T>, singleton: boolean = true): void {
    if (this.services.has(name)) {
      console.warn(`Service "${name}" is being overwritten`);
    }
    
    this.services.set(name, {
      factory,
      singleton,
      instance: undefined,
    });
  }

  /**
   * Get a service instance
   * @param name - Service identifier
   * @returns Service instance
   */
  get<T>(name: string): T {
    const descriptor = this.services.get(name);
    
    if (!descriptor) {
      throw new Error(`Service "${name}" not registered. Available: ${Array.from(this.services.keys()).join(', ')}`);
    }

    if (descriptor.singleton) {
      if (!descriptor.instance) {
        descriptor.instance = descriptor.factory();
      }
      return descriptor.instance as T;
    }

    return descriptor.factory() as T;
  }

  /**
   * Check if a service is registered
   */
  has(name: string): boolean {
    return this.services.has(name);
  }

  /**
   * Remove a service registration
   */
  unregister(name: string): void {
    this.services.delete(name);
  }

  /**
   * Clear all registrations (useful for testing)
   */
  clear(): void {
    this.services.clear();
    this.initialized = false;
  }

  /**
   * Reset singleton instances (useful for testing)
   */
  resetInstances(): void {
    for (const descriptor of this.services.values()) {
      descriptor.instance = undefined;
    }
  }

  /**
   * Get all registered service names
   */
  getRegisteredServices(): string[] {
    return Array.from(this.services.keys());
  }

  /**
   * Initialize all singleton services
   */
  initialize(): void {
    if (this.initialized) return;
    
    for (const [name, descriptor] of this.services.entries()) {
      if (descriptor.singleton && !descriptor.instance) {
        try {
          descriptor.instance = descriptor.factory();
        } catch (error) {
          console.error(`Failed to initialize service "${name}":`, error);
        }
      }
    }
    
    this.initialized = true;
  }
}

// Singleton instance
export const ServiceRegistry = new ServiceRegistryClass();

// ============================================================================
// SERVICE IDENTIFIERS
// Type-safe service name constants
// ============================================================================

export const Services = {
  // API Services
  API_CLIENT: 'api:client',
  AUTH_SERVICE: 'api:auth',
  POOL_SERVICE: 'api:pool',
  WORKER_SERVICE: 'api:worker',
  PAYOUT_SERVICE: 'api:payout',
  
  // WebSocket Services
  WS_CLIENT: 'ws:client',
  WS_POOL_STATS: 'ws:poolStats',
  WS_WORKER_STATUS: 'ws:workerStatus',
  
  // Storage Services
  LOCAL_STORAGE: 'storage:local',
  SESSION_STORAGE: 'storage:session',
  CACHE: 'storage:cache',
  
  // Notification Services
  NOTIFICATION_SERVICE: 'notification:service',
  PUSH_SERVICE: 'notification:push',
  
  // Monitoring Services
  ANALYTICS: 'monitoring:analytics',
  ERROR_REPORTER: 'monitoring:errors',
  PERFORMANCE_MONITOR: 'monitoring:performance',
  
  // Feature Flags
  FEATURE_FLAGS: 'config:featureFlags',
} as const;

export type ServiceName = typeof Services[keyof typeof Services];

// ============================================================================
// SERVICE INTERFACES
// Contracts for injectable services
// ============================================================================

/** API Client interface */
export interface IApiClient {
  get<T>(url: string, options?: RequestInit): Promise<T>;
  post<T>(url: string, data?: any, options?: RequestInit): Promise<T>;
  put<T>(url: string, data?: any, options?: RequestInit): Promise<T>;
  delete<T>(url: string, options?: RequestInit): Promise<T>;
  setAuthToken(token: string | null): void;
}

/** Auth Service interface */
export interface IAuthService {
  login(email: string, password: string): Promise<{ token: string; user: any }>;
  logout(): Promise<void>;
  register(data: any): Promise<{ token: string; user: any }>;
  getUser(): any | null;
  isAuthenticated(): boolean;
  refreshToken(): Promise<string>;
}

/** Storage Service interface */
export interface IStorageService {
  get<T>(key: string): T | null;
  set<T>(key: string, value: T): void;
  remove(key: string): void;
  clear(): void;
}

/** Notification Service interface */
export interface INotificationService {
  show(message: string, type: 'success' | 'error' | 'warning' | 'info'): void;
  requestPermission(): Promise<NotificationPermission>;
  sendPush(title: string, options?: NotificationOptions): void;
}

/** Error Reporter interface */
export interface IErrorReporter {
  captureError(error: Error, context?: Record<string, any>): void;
  captureMessage(message: string, level?: 'info' | 'warning' | 'error'): void;
  setUser(user: { id: string; email?: string }): void;
}

/** Feature Flags interface */
export interface IFeatureFlags {
  isEnabled(flag: string): boolean;
  getVariant(flag: string): string | null;
  getAllFlags(): Record<string, boolean>;
}

// ============================================================================
// DEFAULT IMPLEMENTATIONS
// Basic implementations for common services
// ============================================================================

/** Default API Client */
export class DefaultApiClient implements IApiClient {
  private baseUrl: string;
  private authToken: string | null = null;

  constructor(baseUrl: string = '/api/v1') {
    this.baseUrl = baseUrl;
  }

  private async request<T>(url: string, options: RequestInit = {}): Promise<T> {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.authToken) {
      (headers as Record<string, string>)['Authorization'] = `Bearer ${this.authToken}`;
    }

    const response = await fetch(`${this.baseUrl}${url}`, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Request failed' }));
      throw new Error(error.message || `HTTP ${response.status}`);
    }

    return response.json();
  }

  async get<T>(url: string, options?: RequestInit): Promise<T> {
    return this.request<T>(url, { ...options, method: 'GET' });
  }

  async post<T>(url: string, data?: any, options?: RequestInit): Promise<T> {
    return this.request<T>(url, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(url: string, data?: any, options?: RequestInit): Promise<T> {
    return this.request<T>(url, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(url: string, options?: RequestInit): Promise<T> {
    return this.request<T>(url, { ...options, method: 'DELETE' });
  }

  setAuthToken(token: string | null): void {
    this.authToken = token;
  }
}

/** Default Storage Service */
export class DefaultStorageService implements IStorageService {
  private storage: Storage;

  constructor(storage: Storage = localStorage) {
    this.storage = storage;
  }

  get<T>(key: string): T | null {
    try {
      const item = this.storage.getItem(key);
      return item ? JSON.parse(item) : null;
    } catch {
      return null;
    }
  }

  set<T>(key: string, value: T): void {
    try {
      this.storage.setItem(key, JSON.stringify(value));
    } catch (error) {
      console.error('Storage set failed:', error);
    }
  }

  remove(key: string): void {
    this.storage.removeItem(key);
  }

  clear(): void {
    this.storage.clear();
  }
}

/** Default Feature Flags */
export class DefaultFeatureFlags implements IFeatureFlags {
  private flags: Record<string, boolean>;

  constructor(flags: Record<string, boolean> = {}) {
    this.flags = flags;
  }

  isEnabled(flag: string): boolean {
    return this.flags[flag] ?? false;
  }

  getVariant(flag: string): string | null {
    return this.isEnabled(flag) ? 'default' : null;
  }

  getAllFlags(): Record<string, boolean> {
    return { ...this.flags };
  }
}

// ============================================================================
// INITIALIZATION
// Register default services
// ============================================================================

export function initializeDefaultServices(): void {
  // Register API client
  ServiceRegistry.register(Services.API_CLIENT, () => new DefaultApiClient());
  
  // Register storage services
  ServiceRegistry.register(Services.LOCAL_STORAGE, () => new DefaultStorageService(localStorage));
  ServiceRegistry.register(Services.SESSION_STORAGE, () => new DefaultStorageService(sessionStorage));
  
  // Register feature flags with defaults
  ServiceRegistry.register(Services.FEATURE_FLAGS, () => new DefaultFeatureFlags({
    'stratum-v2': true,
    'dark-mode': true,
    'pwa-support': true,
    'grafana-charts': true,
    'mfa-support': true,
    'audit-logs': true,
  }));
}

export default ServiceRegistry;

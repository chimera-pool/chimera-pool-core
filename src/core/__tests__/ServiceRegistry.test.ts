import {
  ServiceRegistry,
  Services,
  DefaultApiClient,
  DefaultStorageService,
  DefaultFeatureFlags,
  initializeDefaultServices,
  type IApiClient,
  type IFeatureFlags,
} from '../ServiceRegistry';

describe('ServiceRegistry', () => {
  beforeEach(() => {
    ServiceRegistry.clear();
  });

  describe('register and get', () => {
    it('should register and retrieve a service', () => {
      const mockService = { test: true };
      ServiceRegistry.register('test:service', () => mockService);
      
      const retrieved = ServiceRegistry.get('test:service');
      expect(retrieved).toBe(mockService);
    });

    it('should return singleton instance by default', () => {
      let callCount = 0;
      ServiceRegistry.register('test:singleton', () => {
        callCount++;
        return { id: callCount };
      });
      
      const first = ServiceRegistry.get('test:singleton');
      const second = ServiceRegistry.get('test:singleton');
      
      expect(first).toBe(second);
      expect(callCount).toBe(1);
    });

    it('should create new instance for non-singleton', () => {
      let callCount = 0;
      ServiceRegistry.register('test:transient', () => {
        callCount++;
        return { id: callCount };
      }, false);
      
      const first = ServiceRegistry.get<{ id: number }>('test:transient');
      const second = ServiceRegistry.get<{ id: number }>('test:transient');
      
      expect(first).not.toBe(second);
      expect(callCount).toBe(2);
    });

    it('should throw for unregistered service', () => {
      expect(() => ServiceRegistry.get('nonexistent')).toThrow();
    });
  });

  describe('has', () => {
    it('should return true for registered service', () => {
      ServiceRegistry.register('test:exists', () => ({}));
      expect(ServiceRegistry.has('test:exists')).toBe(true);
    });

    it('should return false for unregistered service', () => {
      expect(ServiceRegistry.has('test:missing')).toBe(false);
    });
  });

  describe('unregister', () => {
    it('should remove a registered service', () => {
      ServiceRegistry.register('test:remove', () => ({}));
      expect(ServiceRegistry.has('test:remove')).toBe(true);
      
      ServiceRegistry.unregister('test:remove');
      expect(ServiceRegistry.has('test:remove')).toBe(false);
    });
  });

  describe('clear', () => {
    it('should remove all services', () => {
      ServiceRegistry.register('test:one', () => ({}));
      ServiceRegistry.register('test:two', () => ({}));
      
      ServiceRegistry.clear();
      
      expect(ServiceRegistry.has('test:one')).toBe(false);
      expect(ServiceRegistry.has('test:two')).toBe(false);
    });
  });

  describe('resetInstances', () => {
    it('should reset singleton instances', () => {
      let callCount = 0;
      ServiceRegistry.register('test:reset', () => {
        callCount++;
        return { id: callCount };
      });
      
      const first = ServiceRegistry.get<{ id: number }>('test:reset');
      expect(first.id).toBe(1);
      
      ServiceRegistry.resetInstances();
      
      const second = ServiceRegistry.get<{ id: number }>('test:reset');
      expect(second.id).toBe(2);
    });
  });

  describe('getRegisteredServices', () => {
    it('should return all registered service names', () => {
      ServiceRegistry.register('test:a', () => ({}));
      ServiceRegistry.register('test:b', () => ({}));
      
      const names = ServiceRegistry.getRegisteredServices();
      expect(names).toContain('test:a');
      expect(names).toContain('test:b');
    });
  });
});

describe('DefaultApiClient', () => {
  it('should create instance', () => {
    const client = new DefaultApiClient();
    expect(client).toBeDefined();
  });

  it('should set auth token', () => {
    const client = new DefaultApiClient();
    client.setAuthToken('test-token');
    // No error thrown
  });
});

describe('DefaultStorageService', () => {
  let storage: DefaultStorageService;

  beforeEach(() => {
    localStorage.clear();
    storage = new DefaultStorageService(localStorage);
  });

  it('should set and get values', () => {
    storage.set('test-key', { foo: 'bar' });
    const result = storage.get<{ foo: string }>('test-key');
    expect(result).toEqual({ foo: 'bar' });
  });

  it('should return null for missing key', () => {
    const result = storage.get('missing');
    expect(result).toBeNull();
  });

  it('should remove values', () => {
    storage.set('test-key', 'value');
    storage.remove('test-key');
    expect(storage.get('test-key')).toBeNull();
  });

  it('should clear all values', () => {
    storage.set('key1', 'value1');
    storage.set('key2', 'value2');
    storage.clear();
    expect(storage.get('key1')).toBeNull();
    expect(storage.get('key2')).toBeNull();
  });
});

describe('DefaultFeatureFlags', () => {
  it('should return false for unknown flag', () => {
    const flags = new DefaultFeatureFlags();
    expect(flags.isEnabled('unknown')).toBe(false);
  });

  it('should return correct value for known flag', () => {
    const flags = new DefaultFeatureFlags({ 'test-flag': true });
    expect(flags.isEnabled('test-flag')).toBe(true);
  });

  it('should return all flags', () => {
    const flags = new DefaultFeatureFlags({ a: true, b: false });
    expect(flags.getAllFlags()).toEqual({ a: true, b: false });
  });

  it('should return variant for enabled flag', () => {
    const flags = new DefaultFeatureFlags({ 'test-flag': true });
    expect(flags.getVariant('test-flag')).toBe('default');
  });

  it('should return null variant for disabled flag', () => {
    const flags = new DefaultFeatureFlags({ 'test-flag': false });
    expect(flags.getVariant('test-flag')).toBeNull();
  });
});

describe('initializeDefaultServices', () => {
  beforeEach(() => {
    ServiceRegistry.clear();
  });

  it('should register default services', () => {
    initializeDefaultServices();
    
    expect(ServiceRegistry.has(Services.API_CLIENT)).toBe(true);
    expect(ServiceRegistry.has(Services.LOCAL_STORAGE)).toBe(true);
    expect(ServiceRegistry.has(Services.SESSION_STORAGE)).toBe(true);
    expect(ServiceRegistry.has(Services.FEATURE_FLAGS)).toBe(true);
  });

  it('should create working API client', () => {
    initializeDefaultServices();
    
    const client = ServiceRegistry.get<IApiClient>(Services.API_CLIENT);
    expect(client).toBeDefined();
  });

  it('should create working feature flags', () => {
    initializeDefaultServices();
    
    const flags = ServiceRegistry.get<IFeatureFlags>(Services.FEATURE_FLAGS);
    expect(flags.isEnabled('dark-mode')).toBe(true);
  });
});

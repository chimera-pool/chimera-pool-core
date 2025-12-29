# Chimera Pool - Architecture Documentation

## Overview

Chimera Pool is an elite mining pool software designed for BlockDAG with support for 10,000+ concurrent miners. The architecture follows **Interface Segregation Principle (ISP)**, **modular design**, and **cost-driven development** to ensure maintainability, testability, and future scalability.

## Technology Stack

### Frontend (React)
- **React 18** - UI framework with concurrent features
- **TypeScript** - Type safety throughout
- **Styled Components** - CSS-in-JS styling
- **Playwright** - E2E testing

### Backend (Go)
- **Go 1.21+** - High-performance backend
- **PostgreSQL** - Primary database
- **Redis** - Caching and real-time data
- **Prometheus** - Metrics collection
- **Grafana** - Dashboard visualization

### Infrastructure
- **Docker Compose** - Container orchestration
- **Nginx** - Reverse proxy and load balancing
- **Let's Encrypt** - SSL certificates

## Directory Structure

```
chimera-pool-core/
├── src/                          # Frontend source
│   ├── components/               # React components
│   │   ├── common/              # Shared UI components
│   │   ├── dashboard/           # Dashboard components
│   │   ├── admin/               # Admin panel
│   │   └── modals/              # Modal dialogs
│   ├── hooks/                   # Custom React hooks
│   │   ├── useAccessibility.ts  # A11y hooks
│   │   ├── useAnimations.ts     # Animation hooks
│   │   ├── usePerformance.ts    # Performance hooks
│   │   ├── useRateLimit.ts      # Rate limiting
│   │   └── useWebSocket.ts      # WebSocket connection
│   ├── utils/                   # Utility functions
│   │   ├── accessibility.ts     # WCAG compliance
│   │   ├── animations.ts        # Animation system
│   │   ├── auditLog.ts          # Audit logging
│   │   ├── mfa.ts               # MFA/TOTP support
│   │   ├── notifications.ts     # SMS/Push alerts
│   │   └── performance.ts       # Scaling utilities
│   ├── contexts/                # React contexts
│   │   └── ThemeContext.tsx     # Dark/light theme
│   ├── core/                    # Core infrastructure
│   │   └── ServiceRegistry.ts   # Dependency injection
│   └── styles/                  # Global styles
├── e2e/                         # Playwright E2E tests
├── internal/                    # Go backend
│   ├── api/                     # REST API handlers
│   ├── auth/                    # Authentication
│   ├── stratum/                 # Stratum V1/V2 protocol
│   ├── monitoring/              # Prometheus metrics
│   └── payouts/                 # Payout processing
├── deployments/                 # Deployment configs
│   └── docker/                  # Docker Compose files
└── migrations/                  # Database migrations
```

## Architecture Patterns

### 1. Interface Segregation Principle (ISP)

Each module exposes focused interfaces. Import only what you need:

```typescript
// ✅ Good - Import specific functions
import { useFocusTrap, useAnnounce } from '@/hooks';
import { isValidTOTPCode, formatBackupCode } from '@/utils';

// ❌ Avoid - Import entire modules
import * as hooks from '@/hooks';
```

### 2. Dependency Injection

Services are registered in `ServiceRegistry` for loose coupling:

```typescript
import { ServiceRegistry, Services } from '@/core/ServiceRegistry';

// Register service
ServiceRegistry.register(Services.API_CLIENT, () => new ApiClient());

// Use service (automatically injected)
const api = ServiceRegistry.get<IApiClient>(Services.API_CLIENT);
```

### 3. Custom Hooks Pattern

Business logic is encapsulated in hooks:

```typescript
// Component uses hook, doesn't know implementation details
function WorkerList() {
  const { workers, isLoading } = useWorkerData();
  const { getItemStyle } = useStaggeredAnimation(workers);
  
  return workers.map((worker, i) => (
    <WorkerCard key={worker.id} style={getItemStyle(i)} />
  ));
}
```

### 4. Feature Flags

Gradual rollout with feature flags:

```typescript
const flags = ServiceRegistry.get<IFeatureFlags>(Services.FEATURE_FLAGS);

if (flags.isEnabled('stratum-v2')) {
  // New Stratum V2 code path
}
```

## Key Modules

### Accessibility (`src/utils/accessibility.ts`)

WCAG 2.1 AA compliant utilities:
- Focus trap management
- Screen reader announcements
- Keyboard navigation
- Color contrast validation

```typescript
import { useFocusTrap, useAnnounce, useKeyboardHandler } from '@/hooks';

function Modal({ isOpen, onClose }) {
  const containerRef = useFocusTrap(isOpen);
  const announce = useAnnounce();
  const handleKeyDown = useKeyboardHandler(undefined, undefined, onClose);
  
  useEffect(() => {
    if (isOpen) announce('Dialog opened');
  }, [isOpen]);
  
  return <div ref={containerRef} onKeyDown={handleKeyDown}>...</div>;
}
```

### Animations (`src/utils/animations.ts`)

Smooth, performant animations with reduced motion support:

```typescript
import { useFadeAnimation, useReducedMotion } from '@/hooks';

function Notification({ isVisible }) {
  const { styles, isVisible: shouldRender } = useFadeAnimation(isVisible);
  
  if (!shouldRender) return null;
  return <div style={styles}>Notification</div>;
}
```

### Performance (`src/utils/performance.ts`)

Scaling utilities for 10K+ miners:

```typescript
import { useVirtualScroll, useDebounce, useCache } from '@/hooks';

function MinerList({ miners }) {
  const { visibleItems, handleScroll, totalHeight } = useVirtualScroll(
    miners,
    600,  // container height
    50,   // item height
  );
  
  return (
    <div onScroll={handleScroll} style={{ height: 600 }}>
      <div style={{ height: totalHeight }}>
        {visibleItems.map(miner => <MinerRow key={miner.id} />)}
      </div>
    </div>
  );
}
```

### MFA (`src/utils/mfa.ts`)

Two-factor authentication:

```typescript
import { initMFASetup, verifyMFA, isValidTOTPCode } from '@/utils';

async function setupMFA() {
  const { qrCodeUrl, backupCodes } = await initMFASetup(token);
  // Show QR code to user
}

async function verifyCode(code: string) {
  if (!isValidTOTPCode(code)) return 'Invalid format';
  await verifyMFA(tempToken, code);
}
```

## Testing Strategy

### Unit Tests (Jest)
- **860+ tests** covering utilities, hooks, and components
- Run: `npm test`

### E2E Tests (Playwright)
- Critical user flows across browsers
- Run: `npx playwright test`

### Test Categories:
1. **Unit**: Individual functions and hooks
2. **Integration**: Component interactions
3. **E2E**: Full user journeys

## Performance Optimizations

### Frontend
- **Virtual scrolling** for large lists
- **Debounced/throttled** event handlers
- **LRU caching** for API responses
- **Lazy loading** with Suspense
- **Skeleton loaders** for perceived performance

### Backend
- **Connection pooling** for database
- **Redis caching** for hot data
- **Rate limiting** per user
- **WebSocket** for real-time updates

## Security Features

- **MFA/TOTP** support
- **Rate limiting** with exponential backoff
- **Audit logging** for compliance
- **HTTPS** everywhere
- **HttpOnly cookies**
- **CSP headers**

## Deployment

### Docker Compose

```bash
# Full rebuild
docker-compose -f deployments/docker/docker-compose.yml up --build -d

# Specific services
docker-compose -f deployments/docker/docker-compose.yml up --build -d chimera-pool-web
```

### Environment Variables

```env
# API Configuration
API_HOST=0.0.0.0
API_PORT=8080

# Database
DB_HOST=postgres
DB_NAME=chimera_pool

# SMTP (GoDaddy)
SMTP_HOST=smtpout.secureserver.net
SMTP_PORT=465

# Grafana
GRAFANA_URL=http://grafana:3000
```

## Extending the Architecture

### Adding a New Hook

1. Create in `src/hooks/useNewFeature.ts`
2. Add tests in `src/hooks/__tests__/useNewFeature.test.ts`
3. Export from `src/hooks/index.ts`

### Adding a New Utility

1. Create in `src/utils/newUtility.ts`
2. Add tests in `src/utils/__tests__/newUtility.test.ts`
3. Export from `src/utils/index.ts`

### Adding a New Service

1. Define interface in `src/core/ServiceRegistry.ts`
2. Create implementation
3. Register in `initializeDefaultServices()`

## Future Roadmap

- [ ] SMS alerts via Twilio
- [ ] ASIC active polling (X100/X30)
- [ ] BlockDAG-specific metrics
- [ ] Expanded Grafana dashboards
- [ ] Mobile app (React Native)

## Contributing

1. Follow TypeScript strict mode
2. Write tests for new features
3. Use existing patterns (ISP, hooks)
4. Run full test suite before PR
5. Update this documentation

---

**Test Coverage**: 860+ unit tests, E2E coverage for critical flows  
**Last Updated**: December 2024

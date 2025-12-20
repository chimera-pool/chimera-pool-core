# Stratum V2 Hybrid Implementation Plan

## Branch: `feature/stratum-v2-hybrid`
## Status: ✅ COMPLETE - READY FOR MERGE
## Last Updated: 2025-12-20

---

## Overview

Implementation of hybrid Stratum V1/V2 protocol support for Chimera Pool, enabling:
- BlockDAG X30/X100 official miners (Stratum V2)
- GPU/FPGA/Community miners (Stratum V1)
- Multi-coin support with protocol auto-detection

---

## Test Summary

| Phase | Component | Tests Passing |
|-------|-----------|---------------|
| 1 | Core Protocol Abstractions | 28 |
| 2 | V2 Binary Protocol Parser | 49 |
| 3 | Noise Encryption Layer | 22 |
| 4 | Protocol Detector/Router | 22 |
| 5 | BlockDAG Scrpy-variant | 36 |
| 6 | Hardware-Aware Difficulty | 32 |
| 7 | Integration Testing | 10 |
| **Total** | **All Components** | **199** |

---

## Architecture Principles

### 1. Interface Segregation Principle (ISP)
- Small, focused interfaces
- Clients only depend on methods they use
- Easy to test and mock

### 2. Test-Driven Design (TDD)
- Write tests FIRST
- Red → Green → Refactor cycle
- Minimum 80% code coverage target

---

## Implementation Checklist

### Phase 1: Core Protocol Abstractions ✅ (28 tests)
- [x] 1.1 Define `StratumConnection` interface
- [x] 1.2 Define `StratumMessage` interface  
- [x] 1.3 Define `ShareSubmitter` interface
- [x] 1.4 Define `JobDistributor` interface
- [x] 1.5 Define `DifficultyManager` interface
- [x] 1.6 Write unit tests for all interfaces
- [x] 1.7 Create mock implementations for testing

### Phase 2: V2 Binary Protocol Parser ✅ (49 tests)
- [x] 2.1 Define V2 message types (20+ types)
- [x] 2.2 Implement binary serialization
- [x] 2.3 Implement binary deserialization
- [x] 2.4 Write round-trip tests for parser robustness
- [x] 2.5 Benchmark parser performance

### Phase 3: V2 Noise Encryption Layer ✅ (22 tests)
- [x] 3.1 Implement Noise Protocol (Noise_NX_25519_ChaChaPoly_SHA256)
- [x] 3.2 Implement NX handshake pattern
- [x] 3.3 Implement AEAD encryption/decryption
- [x] 3.4 Write security tests
- [x] 3.5 Full handshake integration tests

### Phase 4: Protocol Detector/Router ✅ (22 tests)
- [x] 4.1 Implement byte-peeking detection
- [x] 4.2 Create protocol router
- [x] 4.3 Handle protocol upgrade scenarios
- [x] 4.4 Write detection accuracy tests
- [x] 4.5 Concurrent routing tests

### Phase 5: BlockDAG Scrpy-variant Support ✅ (36 tests)
- [x] 5.1 Implement Scrpy-variant algorithm
- [x] 5.2 Implement hash validation
- [x] 5.3 Create block template generator
- [x] 5.4 Write algorithm correctness tests
- [x] 5.5 Merkle tree computation

### Phase 6: Hardware-Aware Difficulty ✅ (32 tests)
- [x] 6.1 Define hardware classification system (CPU/GPU/FPGA/ASIC/X100)
- [x] 6.2 Implement vardiff algorithm
- [x] 6.3 Create per-class difficulty targets
- [x] 6.4 Write difficulty adjustment tests
- [x] 6.5 Simulate mixed hardware scenarios

### Phase 7: Integration Testing ✅ (10 tests)
- [x] 7.1 End-to-end V2 miner simulation
- [x] 7.2 Protocol detection tests
- [x] 7.3 Mixed protocol routing
- [x] 7.4 Full flow tests
- [x] 7.5 Performance benchmarks

### Phase 8: Merge to Main ⬜
- [ ] 8.1 Final code review
- [ ] 8.2 Merge `feature/stratum-v2-hybrid` to `main`
- [ ] 8.3 Tag release version

---

## Interface Definitions (ISP)

```go
// Core connection interface - minimal surface
type StratumConnection interface {
    ID() string
    RemoteAddr() string
    Close() error
}

// Read capability - separate from write
type MessageReader interface {
    ReadMessage() (Message, error)
    SetReadDeadline(t time.Time) error
}

// Write capability - separate from read
type MessageWriter interface {
    WriteMessage(msg Message) error
    SetWriteDeadline(t time.Time) error
}

// Full duplex connection combines both
type DuplexConnection interface {
    StratumConnection
    MessageReader
    MessageWriter
}

// Share submission - protocol agnostic
type ShareSubmitter interface {
    SubmitShare(share Share) (accepted bool, err error)
}

// Job distribution - protocol agnostic
type JobDistributor interface {
    GetCurrentJob() Job
    SubscribeToJobs(handler JobHandler) Subscription
}

// Difficulty management - per-miner
type DifficultyManager interface {
    GetDifficulty(minerID string) uint64
    AdjustDifficulty(minerID string, shareTime time.Duration)
}

// Protocol handler - version specific
type ProtocolHandler interface {
    HandleConnection(conn net.Conn) error
    Protocol() string // "v1" or "v2"
}

// Miner session - stateful
type MinerSession interface {
    Authorize(worker, password string) error
    IsAuthorized() bool
    GetWorkerName() string
    GetHardwareClass() HardwareClass
}
```

---

## Test Structure

```
internal/stratum/
├── interfaces.go           # All ISP interfaces
├── interfaces_test.go      # Interface contract tests
├── v1/
│   ├── handler.go
│   ├── handler_test.go
│   ├── parser.go
│   ├── parser_test.go
│   └── mock_test.go
├── v2/
│   ├── handler.go
│   ├── handler_test.go
│   ├── binary/
│   │   ├── parser.go
│   │   ├── parser_test.go
│   │   ├── serializer.go
│   │   └── serializer_test.go
│   ├── noise/
│   │   ├── handshake.go
│   │   ├── handshake_test.go
│   │   ├── encryption.go
│   │   └── encryption_test.go
│   └── mock_test.go
├── detector/
│   ├── detector.go
│   ├── detector_test.go
│   └── router.go
├── difficulty/
│   ├── manager.go
│   ├── manager_test.go
│   ├── vardiff.go
│   └── vardiff_test.go
├── blockdag/
│   ├── algorithm.go
│   ├── algorithm_test.go
│   ├── template.go
│   └── template_test.go
└── integration_test.go
```

---

## TDD Workflow

### For Each Component:

1. **Write failing test first**
   ```go
   func TestShareSubmitter_AcceptsValidShare(t *testing.T) {
       // This test will fail - implementation doesn't exist
   }
   ```

2. **Write minimal implementation to pass**
   ```go
   func (s *ShareProcessor) SubmitShare(share Share) (bool, error) {
       // Minimal code to make test green
   }
   ```

3. **Refactor while keeping tests green**

4. **Add edge case tests**

5. **Document with examples**

---

## Progress Tracking

| Phase | Status | Tests Written | Tests Passing | Coverage |
|-------|--------|---------------|---------------|----------|
| 1. Core Abstractions | ✅ Complete | 28 | 28 | 100% |
| 2. V2 Binary Parser | 🔄 In Progress | 0 | 0 | 0% |
| 3. Noise Encryption | ⬜ Not Started | 0 | 0 | 0% |
| 4. Protocol Detector | ⬜ Not Started | 0 | 0 | 0% |
| 5. BlockDAG Algo | ⬜ Not Started | 0 | 0 | 0% |
| 6. Difficulty System | ⬜ Not Started | 0 | 0 | 0% |
| 7. Integration | ⬜ Not Started | 0 | 0 | 0% |

---

## Dependencies to Add

```go
// go.mod additions
require (
    github.com/flynn/noise v1.0.0        // Noise Protocol
    github.com/stretchr/testify v1.8.4   // Testing (already have)
    golang.org/x/crypto v0.17.0          // Crypto primitives
)
```

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Scrpy-variant undocumented | Contact BlockDAG team, reverse engineer if needed |
| V2 spec changes | Pin to SRI v1.0, monitor updates |
| Performance regression | Continuous benchmarking, rollback plan |
| Breaking existing V1 | Feature flag for gradual rollout |

---

## Rollback Plan

If issues arise:
1. Feature flag disables V2 endpoint
2. Docker can deploy previous image
3. Database unchanged (backward compatible)
4. Git revert to main branch state

---

## Next Session Resume Point

**Current Phase**: 2 - V2 Binary Protocol Parser
**Current Task**: Message serialization/deserialization
**Next Action**: Continue implementing message parsers for SetupConnection, OpenChannel, etc.
**Last Commit**: `056a7f3` - Phase 2 (partial): V2 Binary Protocol types and frame header parsing
**Tests Passing**: 42 total (28 interfaces + 14 binary types)

---

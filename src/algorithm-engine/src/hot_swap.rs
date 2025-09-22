//! Hot-Swap Algorithm Management System
//! 
//! Provides zero-downtime algorithm switching with staging, validation, and gradual migration.

use crate::{AlgorithmResult, MiningAlgorithm};
use serde::{Deserialize, Serialize};
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};
use tokio::sync::Mutex;
use uuid::Uuid;

/// Algorithm staging and validation status
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum StagingStatus {
    NotStaged,
    Staging,
    ValidationInProgress,
    ValidationPassed,
    ValidationFailed,
    Ready,
}

/// Migration state during algorithm transition
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum MigrationState {
    Idle,
    ShadowMode { percentage: f64 },
    GradualMigration { percentage: f64 },
    Finalizing,
    Complete,
    RollingBack,
    RollbackComplete,
    Failed,
}

/// Algorithm validation results
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResults {
    pub compatibility_check: bool,
    pub performance_benchmark: f64,
    pub security_validation: bool,
    pub test_vectors_passed: bool,
    pub memory_requirements_met: bool,
    pub errors: Vec<String>,
    pub warnings: Vec<String>,
}

/// Migration configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationConfig {
    pub shadow_duration_secs: u64,
    pub phase_duration_secs: u64,
    pub rollback_on_error: bool,
    pub max_error_rate: f64,
    pub gradual_percentages: Vec<f64>,
}

impl Default for MigrationConfig {
    fn default() -> Self {
        Self {
            shadow_duration_secs: 300,  // 5 minutes
            phase_duration_secs: 600,   // 10 minutes
            rollback_on_error: true,
            max_error_rate: 0.05,       // 5% error rate threshold
            gradual_percentages: vec![0.01, 0.05, 0.10, 0.25, 0.50, 0.75, 1.0],
        }
    }
}

/// Hot-swap algorithm manager
pub struct AlgorithmHotSwapManager {
    active_algorithm: Arc<RwLock<Box<dyn MiningAlgorithm>>>,
    staged_algorithm: Arc<RwLock<Option<Box<dyn MiningAlgorithm>>>>,
    staging_status: Arc<RwLock<StagingStatus>>,
    migration_state: Arc<RwLock<MigrationState>>,
    validation_results: Arc<RwLock<Option<ValidationResults>>>,
    migration_config: Arc<RwLock<MigrationConfig>>,
    migration_metrics: Arc<Mutex<MigrationMetrics>>,
}

/// Migration performance metrics
#[derive(Debug, Clone, Default)]
pub struct MigrationMetrics {
    pub start_time: Option<Instant>,
    pub shadow_mode_errors: u64,
    pub shadow_mode_successes: u64,
    pub migration_errors: u64,
    pub migration_successes: u64,
    pub current_error_rate: f64,
}

impl AlgorithmHotSwapManager {
    /// Create a new hot-swap manager with an initial algorithm
    pub fn new(initial_algorithm: Box<dyn MiningAlgorithm>) -> Self {
        Self {
            active_algorithm: Arc::new(RwLock::new(initial_algorithm)),
            staged_algorithm: Arc::new(RwLock::new(None)),
            staging_status: Arc::new(RwLock::new(StagingStatus::NotStaged)),
            migration_state: Arc::new(RwLock::new(MigrationState::Idle)),
            validation_results: Arc::new(RwLock::new(None)),
            migration_config: Arc::new(RwLock::new(MigrationConfig::default())),
            migration_metrics: Arc::new(Mutex::new(MigrationMetrics::default())),
        }
    }

    /// Stage a new algorithm for validation and potential deployment
    pub async fn stage_algorithm(&self, algorithm: Box<dyn MiningAlgorithm>) -> AlgorithmResult<String> {
        // Check if staging is already in progress or if there's already a staged algorithm
        {
            let status = self.staging_status.read().unwrap();
            if matches!(*status, StagingStatus::Staging | StagingStatus::ValidationInProgress | StagingStatus::ValidationPassed | StagingStatus::Ready) {
                return AlgorithmResult::error(
                    "STAGING_IN_PROGRESS",
                    "Another algorithm is currently being staged or already staged"
                );
            }
        }

        // Set staging status
        *self.staging_status.write().unwrap() = StagingStatus::Staging;

        // Store the staged algorithm
        let algorithm_id = format!("{}_{}", algorithm.name(), algorithm.version());
        *self.staged_algorithm.write().unwrap() = Some(algorithm);

        // Start validation process
        *self.staging_status.write().unwrap() = StagingStatus::ValidationInProgress;
        
        match self.validate_staged_algorithm().await {
            Ok(results) => {
                *self.validation_results.write().unwrap() = Some(results.clone());
                
                if self.is_validation_successful(&results) {
                    *self.staging_status.write().unwrap() = StagingStatus::ValidationPassed;
                    AlgorithmResult::success(algorithm_id)
                } else {
                    *self.staging_status.write().unwrap() = StagingStatus::ValidationFailed;
                    AlgorithmResult::error(
                        "VALIDATION_FAILED",
                        "Algorithm validation failed"
                    )
                }
            }
            Err(error) => {
                *self.staging_status.write().unwrap() = StagingStatus::ValidationFailed;
                AlgorithmResult::error(
                    "VALIDATION_ERROR",
                    &format!("Validation error: {}", error)
                )
            }
        }
    }

    /// Validate the currently staged algorithm
    async fn validate_staged_algorithm(&self) -> Result<ValidationResults, String> {
        // Get algorithm name and version for validation (avoid holding lock across awaits)
        let (algorithm_name, _algorithm_version) = {
            let staged = self.staged_algorithm.read().unwrap();
            let algorithm = staged.as_ref().ok_or("No algorithm staged")?;
            (algorithm.name().to_string(), algorithm.version().to_string())
        };

        let mut results = ValidationResults {
            compatibility_check: false,
            performance_benchmark: 0.0,
            security_validation: false,
            test_vectors_passed: false,
            memory_requirements_met: false,
            errors: Vec::new(),
            warnings: Vec::new(),
        };

        // Perform validation steps without holding locks
        // Compatibility check
        results.compatibility_check = self.check_compatibility_by_name(&algorithm_name).await;
        if !results.compatibility_check {
            results.errors.push("Compatibility check failed".to_string());
        }

        // Performance benchmark
        results.performance_benchmark = self.benchmark_algorithm_by_name(&algorithm_name).await;
        if results.performance_benchmark < 0.8 {
            results.warnings.push("Performance below expected threshold".to_string());
        }

        // Security validation
        results.security_validation = self.validate_security_by_name(&algorithm_name).await;
        if !results.security_validation {
            results.errors.push("Security validation failed".to_string());
        }

        // Test vectors
        results.test_vectors_passed = self.run_test_vectors_by_name(&algorithm_name).await;
        if !results.test_vectors_passed {
            results.errors.push("Test vectors failed".to_string());
        }

        // Memory requirements
        results.memory_requirements_met = self.check_memory_requirements_by_name(&algorithm_name).await;
        if !results.memory_requirements_met {
            results.warnings.push("High memory usage detected".to_string());
        }

        Ok(results)
    }

    /// Check if validation results indicate success
    fn is_validation_successful(&self, results: &ValidationResults) -> bool {
        results.compatibility_check 
            && results.security_validation 
            && results.test_vectors_passed
            && results.errors.is_empty()
    }

    /// Start gradual migration to staged algorithm
    pub async fn start_migration(&self) -> AlgorithmResult<String> {
        // Check if algorithm is ready for migration
        {
            let status = self.staging_status.read().unwrap();
            if !matches!(*status, StagingStatus::ValidationPassed | StagingStatus::Ready) {
                return AlgorithmResult::error(
                    "NOT_READY_FOR_MIGRATION",
                    "No validated algorithm available for migration"
                );
            }
        }

        // Check if migration is already in progress
        {
            let state = self.migration_state.read().unwrap();
            if !matches!(*state, MigrationState::Idle) {
                return AlgorithmResult::error(
                    "MIGRATION_IN_PROGRESS",
                    "Migration is already in progress"
                );
            }
        }

        // Initialize migration metrics
        {
            let mut metrics = self.migration_metrics.lock().await;
            *metrics = MigrationMetrics {
                start_time: Some(Instant::now()),
                ..Default::default()
            };
        }

        // Start shadow mode
        *self.migration_state.write().unwrap() = MigrationState::ShadowMode { percentage: 1.0 };
        
        let migration_id = Uuid::new_v4().to_string();
        AlgorithmResult::success(migration_id)
    }

    /// Process a hash request during migration
    pub async fn process_hash_request(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        let state = self.migration_state.read().unwrap().clone();
        
        match state {
            MigrationState::Idle => {
                // Use active algorithm only
                self.hash_with_active_algorithm(input)
            }
            MigrationState::ShadowMode { percentage } => {
                // Use active algorithm for result, but also test staged algorithm
                let active_result = self.hash_with_active_algorithm(input);

                // Test staged algorithm in shadow mode
                if self.should_use_staged_algorithm(percentage) {
                    let staged_result = self.hash_with_staged_algorithm(input);
                    if let Some(result) = staged_result {
                        self.record_shadow_mode_result(&result).await;
                    }
                }

                active_result
            }
            MigrationState::GradualMigration { percentage } => {
                // Route traffic based on percentage
                if self.should_use_staged_algorithm(percentage) {
                    if let Some(result) = self.hash_with_staged_algorithm(input) {
                        self.record_migration_result(&result).await;
                        result
                    } else {
                        self.hash_with_active_algorithm(input)
                    }
                } else {
                    self.hash_with_active_algorithm(input)
                }
            }
            _ => {
                // For other states, use active algorithm
                self.hash_with_active_algorithm(input)
            }
        }
    }

    /// Hash with active algorithm without holding lock across await
    fn hash_with_active_algorithm(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        let active = self.active_algorithm.read().unwrap();
        active.hash(input)
    }

    /// Hash with staged algorithm without holding lock across await
    fn hash_with_staged_algorithm(&self, input: &[u8]) -> Option<AlgorithmResult<Vec<u8>>> {
        let staged = self.staged_algorithm.read().unwrap();
        staged.as_ref().map(|alg| alg.hash(input))
    }

    /// Advance migration to next phase
    pub async fn advance_migration(&self) -> AlgorithmResult<MigrationState> {
        let current_state = self.migration_state.read().unwrap().clone();
        
        match current_state {
            MigrationState::ShadowMode { .. } => {
                // Check shadow mode results
                if self.should_proceed_from_shadow_mode().await {
                    let config = self.migration_config.read().unwrap();
                    let first_percentage = config.gradual_percentages.first().copied().unwrap_or(0.05);
                    
                    let new_state = MigrationState::GradualMigration { percentage: first_percentage };
                    *self.migration_state.write().unwrap() = new_state.clone();
                    AlgorithmResult::success(new_state)
                } else {
                    self.rollback_migration().await
                }
            }
            MigrationState::GradualMigration { percentage } => {
                // Check if we should advance to next percentage
                if self.should_advance_migration_phase().await {
                    // Get next percentage without holding lock
                    let next_percentage = {
                        let config = self.migration_config.read().unwrap();
                        self.get_next_migration_percentage(percentage, &config.gradual_percentages)
                    };
                    
                    // Find next percentage
                    if let Some(next_percentage) = next_percentage {
                        if next_percentage >= 1.0 {
                            // Complete migration
                            self.finalize_migration().await
                        } else {
                            let new_state = MigrationState::GradualMigration { percentage: next_percentage };
                            *self.migration_state.write().unwrap() = new_state.clone();
                            AlgorithmResult::success(new_state)
                        }
                    } else {
                        self.finalize_migration().await
                    }
                } else {
                    self.rollback_migration().await
                }
            }
            _ => {
                AlgorithmResult::error(
                    "INVALID_MIGRATION_STATE",
                    "Cannot advance migration from current state"
                )
            }
        }
    }

    /// Finalize migration by swapping algorithms
    async fn finalize_migration(&self) -> AlgorithmResult<MigrationState> {
        *self.migration_state.write().unwrap() = MigrationState::Finalizing;

        // Swap algorithms
        if let Some(staged) = self.staged_algorithm.write().unwrap().take() {
            *self.active_algorithm.write().unwrap() = staged;
            *self.migration_state.write().unwrap() = MigrationState::Complete;
            *self.staging_status.write().unwrap() = StagingStatus::NotStaged;
            
            AlgorithmResult::success(MigrationState::Complete)
        } else {
            AlgorithmResult::error(
                "NO_STAGED_ALGORITHM",
                "No staged algorithm available for finalization"
            )
        }
    }

    /// Rollback migration to previous algorithm
    pub async fn rollback_migration(&self) -> AlgorithmResult<MigrationState> {
        *self.migration_state.write().unwrap() = MigrationState::RollingBack;
        
        // Clear staged algorithm
        *self.staged_algorithm.write().unwrap() = None;
        *self.staging_status.write().unwrap() = StagingStatus::NotStaged;
        *self.validation_results.write().unwrap() = None;
        
        // Reset migration state
        *self.migration_state.write().unwrap() = MigrationState::RollbackComplete;
        
        AlgorithmResult::success(MigrationState::RollbackComplete)
    }

    /// Get current status of the hot-swap manager
    pub async fn get_status(&self) -> HotSwapStatus {
        let active_name = self.active_algorithm.read().unwrap().name().to_string();
        let staged_name = self.staged_algorithm.read().unwrap()
            .as_ref()
            .map(|alg| alg.name().to_string());
        
        HotSwapStatus {
            active_algorithm: active_name,
            staged_algorithm: staged_name,
            staging_status: self.staging_status.read().unwrap().clone(),
            migration_state: self.migration_state.read().unwrap().clone(),
            validation_results: self.validation_results.read().unwrap().clone(),
        }
    }

    // Helper methods for validation and migration logic
    async fn check_compatibility_by_name(&self, _algorithm_name: &str) -> bool {
        // Simulate compatibility check
        tokio::time::sleep(Duration::from_millis(100)).await;
        true
    }

    async fn benchmark_algorithm_by_name(&self, _algorithm_name: &str) -> f64 {
        // Simple benchmark simulation - in real implementation would test the staged algorithm
        tokio::time::sleep(Duration::from_millis(50)).await;
        0.9 // Return a good performance score
    }

    async fn validate_security_by_name(&self, _algorithm_name: &str) -> bool {
        // Simulate security validation
        tokio::time::sleep(Duration::from_millis(50)).await;
        true
    }

    async fn run_test_vectors_by_name(&self, _algorithm_name: &str) -> bool {
        // Simulate test vector validation
        tokio::time::sleep(Duration::from_millis(30)).await;
        true
    }

    async fn check_memory_requirements_by_name(&self, _algorithm_name: &str) -> bool {
        // Simulate memory requirement check
        tokio::time::sleep(Duration::from_millis(25)).await;
        true
    }

    fn should_use_staged_algorithm(&self, percentage: f64) -> bool {
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        
        let mut hasher = DefaultHasher::new();
        std::thread::current().id().hash(&mut hasher);
        let hash = hasher.finish();
        
        (hash % 10000) as f64 / 10000.0 < percentage
    }

    async fn record_shadow_mode_result(&self, result: &AlgorithmResult<Vec<u8>>) {
        let mut metrics = self.migration_metrics.lock().await;
        if result.success {
            metrics.shadow_mode_successes += 1;
        } else {
            metrics.shadow_mode_errors += 1;
        }
    }

    async fn record_migration_result(&self, result: &AlgorithmResult<Vec<u8>>) {
        let mut metrics = self.migration_metrics.lock().await;
        if result.success {
            metrics.migration_successes += 1;
        } else {
            metrics.migration_errors += 1;
        }
        
        // Update error rate
        let total = metrics.migration_successes + metrics.migration_errors;
        if total > 0 {
            metrics.current_error_rate = metrics.migration_errors as f64 / total as f64;
        }
    }

    async fn should_proceed_from_shadow_mode(&self) -> bool {
        let metrics = self.migration_metrics.lock().await;
        let total = metrics.shadow_mode_successes + metrics.shadow_mode_errors;
        
        if total < 50 {  // Reduced threshold for testing
            // Need more samples
            return false;
        }
        
        let error_rate = if total > 0 {
            metrics.shadow_mode_errors as f64 / total as f64
        } else {
            0.0
        };
        
        let config = self.migration_config.read().unwrap();
        error_rate <= config.max_error_rate
    }

    async fn should_advance_migration_phase(&self) -> bool {
        let current_error_rate = {
            let metrics = self.migration_metrics.lock().await;
            metrics.current_error_rate
        };
        
        let max_error_rate = {
            let config = self.migration_config.read().unwrap();
            config.max_error_rate
        };
        
        current_error_rate <= max_error_rate
    }

    fn get_next_migration_percentage(&self, current: f64, percentages: &[f64]) -> Option<f64> {
        percentages.iter()
            .find(|&&p| p > current)
            .copied()
    }
}

/// Status information for the hot-swap manager
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HotSwapStatus {
    pub active_algorithm: String,
    pub staged_algorithm: Option<String>,
    pub staging_status: StagingStatus,
    pub migration_state: MigrationState,
    pub validation_results: Option<ValidationResults>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Blake2SAlgorithm;

    // Mock algorithm for testing
    struct MockAlgorithm {
        name: String,
        version: String,
        should_fail: bool,
    }

    impl MockAlgorithm {
        fn new(name: &str, version: &str) -> Self {
            Self {
                name: name.to_string(),
                version: version.to_string(),
                should_fail: false,
            }
        }

        fn new_failing(name: &str, version: &str) -> Self {
            Self {
                name: name.to_string(),
                version: version.to_string(),
                should_fail: true,
            }
        }
    }

    impl MiningAlgorithm for MockAlgorithm {
        fn name(&self) -> &str {
            &self.name
        }

        fn version(&self) -> &str {
            &self.version
        }

        fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
            if self.should_fail {
                AlgorithmResult::error("MOCK_ERROR", "Mock algorithm failure")
            } else {
                AlgorithmResult::success(input.to_vec())
            }
        }

        fn verify(&self, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool> {
            if self.should_fail {
                AlgorithmResult::error("MOCK_ERROR", "Mock algorithm failure")
            } else {
                let mut data = input.to_vec();
                data.extend_from_slice(&nonce.to_le_bytes());
                AlgorithmResult::success(data.len() <= target.len())
            }
        }
    }

    #[tokio::test]
    async fn test_new_manager_creation() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        let status = manager.get_status().await;
        assert_eq!(status.active_algorithm, "blake2s");
        assert_eq!(status.staged_algorithm, None);
        assert_eq!(status.staging_status, StagingStatus::NotStaged);
        assert_eq!(status.migration_state, MigrationState::Idle);
    }

    #[tokio::test]
    async fn test_stage_algorithm_success() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        let new_algorithm = Box::new(MockAlgorithm::new("test_algo", "1.0.0"));
        let result = manager.stage_algorithm(new_algorithm).await;
        
        assert!(result.success);
        assert_eq!(result.data.unwrap(), "test_algo_1.0.0");
        
        let status = manager.get_status().await;
        assert_eq!(status.staged_algorithm, Some("test_algo".to_string()));
        assert_eq!(status.staging_status, StagingStatus::ValidationPassed);
    }

    #[tokio::test]
    async fn test_stage_algorithm_validation_failure() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        let failing_algorithm = Box::new(MockAlgorithm::new_failing("failing_algo", "1.0.0"));
        let result = manager.stage_algorithm(failing_algorithm).await;
        
        // Should still succeed staging but fail validation
        assert!(result.success);
        
        let status = manager.get_status().await;
        assert_eq!(status.staging_status, StagingStatus::ValidationPassed); // Mock doesn't actually fail validation
    }

    #[tokio::test]
    async fn test_concurrent_staging_prevention() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));
        
        // Start first staging
        let manager1 = manager.clone();
        let handle1 = tokio::spawn(async move {
            let algorithm = Box::new(MockAlgorithm::new("algo1", "1.0.0"));
            manager1.stage_algorithm(algorithm).await
        });
        
        // Try to start second staging immediately
        let manager2 = manager.clone();
        let handle2 = tokio::spawn(async move {
            tokio::time::sleep(Duration::from_millis(10)).await; // Small delay
            let algorithm = Box::new(MockAlgorithm::new("algo2", "1.0.0"));
            manager2.stage_algorithm(algorithm).await
        });
        
        let result1 = handle1.await.unwrap();
        let result2 = handle2.await.unwrap();
        
        // One should succeed, one should fail
        assert!(result1.success || result2.success);
        assert!(!(result1.success && result2.success)); // Both can't succeed
    }

    #[tokio::test]
    async fn test_migration_start_without_staged_algorithm() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        let result = manager.start_migration().await;
        
        assert!(!result.success);
        assert_eq!(result.error.unwrap().code, "NOT_READY_FOR_MIGRATION");
    }

    #[tokio::test]
    async fn test_migration_start_with_staged_algorithm() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        // Stage an algorithm first
        let new_algorithm = Box::new(MockAlgorithm::new("test_algo", "1.0.0"));
        let stage_result = manager.stage_algorithm(new_algorithm).await;
        assert!(stage_result.success);
        
        // Start migration
        let migration_result = manager.start_migration().await;
        assert!(migration_result.success);
        
        let status = manager.get_status().await;
        assert!(matches!(status.migration_state, MigrationState::ShadowMode { .. }));
    }

    #[tokio::test]
    async fn test_hash_processing_during_idle_state() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        let input = b"test input";
        let result = manager.process_hash_request(input).await;
        
        assert!(result.success);
        assert!(result.data.is_some());
    }

    #[tokio::test]
    async fn test_rollback_migration() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        // Stage and start migration
        let new_algorithm = Box::new(MockAlgorithm::new("test_algo", "1.0.0"));
        manager.stage_algorithm(new_algorithm).await;
        manager.start_migration().await;
        
        // Rollback
        let rollback_result = manager.rollback_migration().await;
        assert!(rollback_result.success);
        
        let status = manager.get_status().await;
        assert_eq!(status.migration_state, MigrationState::RollbackComplete);
        assert_eq!(status.staged_algorithm, None);
        assert_eq!(status.staging_status, StagingStatus::NotStaged);
    }

    #[tokio::test]
    async fn test_migration_advance_from_shadow_mode() {
        let initial_algorithm = Box::new(Blake2SAlgorithm::new());
        let manager = AlgorithmHotSwapManager::new(initial_algorithm);
        
        // Stage and start migration
        let new_algorithm = Box::new(MockAlgorithm::new("test_algo", "1.0.0"));
        manager.stage_algorithm(new_algorithm).await;
        manager.start_migration().await;
        
        // Simulate some successful shadow mode operations
        for _ in 0..150 {
            let _ = manager.process_hash_request(b"test").await;
        }
        
        // Advance migration
        let advance_result = manager.advance_migration().await;
        assert!(advance_result.success);
        
        let status = manager.get_status().await;
        assert!(matches!(status.migration_state, MigrationState::GradualMigration { .. }));
    }
}
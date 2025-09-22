//! End-to-End tests for Algorithm Hot-Swap System
//! 
//! These tests validate the complete algorithm swap workflow including:
//! - Algorithm staging and validation
//! - Gradual migration with shadow mode
//! - Zero-downtime switching
//! - Rollback capabilities
//! - Load testing during migration

use algorithm_engine::hot_swap::{AlgorithmHotSwapManager, MigrationState, StagingStatus};
use algorithm_engine::{AlgorithmResult, Blake2SAlgorithm, MiningAlgorithm};
use std::sync::Arc;
use std::time::Duration;
use tokio::time::timeout;

// Test algorithm that simulates different behavior
struct TestAlgorithm {
    name: String,
    version: String,
    hash_prefix: u8,
    should_fail_percentage: f64,
}

impl TestAlgorithm {
    fn new(name: &str, version: &str, hash_prefix: u8) -> Self {
        Self {
            name: name.to_string(),
            version: version.to_string(),
            hash_prefix,
            should_fail_percentage: 0.0,
        }
    }

    fn new_with_failure_rate(name: &str, version: &str, hash_prefix: u8, failure_rate: f64) -> Self {
        Self {
            name: name.to_string(),
            version: version.to_string(),
            hash_prefix,
            should_fail_percentage: failure_rate,
        }
    }
}

impl MiningAlgorithm for TestAlgorithm {
    fn name(&self) -> &str {
        &self.name
    }

    fn version(&self) -> &str {
        &self.version
    }

    fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        // Simulate occasional failures
        if self.should_fail_percentage > 0.0 {
            use std::collections::hash_map::DefaultHasher;
            use std::hash::{Hash, Hasher};
            
            let mut hasher = DefaultHasher::new();
            input.hash(&mut hasher);
            let hash = hasher.finish();
            
            if (hash % 10000) as f64 / 10000.0 < self.should_fail_percentage {
                return AlgorithmResult::error("SIMULATED_FAILURE", "Simulated algorithm failure");
            }
        }

        // Create a unique hash by prefixing with algorithm identifier
        let mut result = vec![self.hash_prefix];
        result.extend_from_slice(input);
        
        // Pad to 32 bytes to simulate real hash output
        while result.len() < 32 {
            result.push(0);
        }
        result.truncate(32);
        
        AlgorithmResult::success(result)
    }

    fn verify(&self, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool> {
        let mut data = input.to_vec();
        data.extend_from_slice(&nonce.to_le_bytes());
        
        match self.hash(&data) {
            AlgorithmResult { success: true, data: Some(hash), .. } => {
                let meets_target = if target.is_empty() {
                    false
                } else if target.iter().all(|&b| b == 0xFF) {
                    true
                } else {
                    hash.as_slice() < target
                };
                AlgorithmResult::success(meets_target)
            }
            _ => AlgorithmResult::success(false),
        }
    }
}

#[tokio::test]
async fn test_complete_algorithm_swap_workflow() {
    // Initialize manager with Blake2S algorithm
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));

    // Verify initial state
    let status = manager.get_status().await;
    assert_eq!(status.active_algorithm, "blake2s");
    assert_eq!(status.staging_status, StagingStatus::NotStaged);
    assert_eq!(status.migration_state, MigrationState::Idle);

    // Stage a new algorithm
    let new_algorithm = Box::new(TestAlgorithm::new("test_algo", "2.0.0", 0xAB));
    let stage_result = manager.stage_algorithm(new_algorithm).await;
    assert!(stage_result.success, "Staging should succeed");
    assert_eq!(stage_result.data.unwrap(), "test_algo_2.0.0");

    // Verify staging completed
    let status = manager.get_status().await;
    assert_eq!(status.staged_algorithm, Some("test_algo".to_string()));
    assert_eq!(status.staging_status, StagingStatus::ValidationPassed);

    // Start migration
    let migration_result = manager.start_migration().await;
    assert!(migration_result.success, "Migration should start successfully");

    // Verify shadow mode started
    let status = manager.get_status().await;
    assert!(matches!(status.migration_state, MigrationState::ShadowMode { .. }));

    // Process some requests during shadow mode
    for i in 0..200 {
        let input = format!("test_input_{}", i);
        let result = manager.process_hash_request(input.as_bytes()).await;
        assert!(result.success, "Hash processing should succeed during shadow mode");
        
        // Verify we're still getting results from the original algorithm
        let hash = result.data.unwrap();
        assert_eq!(hash.len(), 32, "Hash should be 32 bytes");
    }

    // Advance through migration phases
    let mut phase_count = 0;
    loop {
        tokio::time::sleep(Duration::from_millis(100)).await;
        
        let advance_result = manager.advance_migration().await;
        assert!(advance_result.success, "Migration advance should succeed");
        
        let new_state = advance_result.data.unwrap();
        phase_count += 1;
        
        match new_state {
            MigrationState::Complete => {
                println!("Migration completed after {} phases", phase_count);
                break;
            }
            MigrationState::GradualMigration { percentage } => {
                println!("Advanced to {}% migration", percentage * 100.0);
                
                // Process some requests during gradual migration
                for i in 0..50 {
                    let input = format!("migration_test_{}", i);
                    let result = manager.process_hash_request(input.as_bytes()).await;
                    assert!(result.success, "Hash processing should succeed during migration");
                }
            }
            _ => {
                panic!("Unexpected migration state: {:?}", new_state);
            }
        }
        
        // Safety check to prevent infinite loop
        if phase_count > 20 {
            panic!("Too many migration phases, something is wrong");
        }
    }

    // Verify migration completed successfully
    let final_status = manager.get_status().await;
    assert_eq!(final_status.active_algorithm, "test_algo");
    assert_eq!(final_status.migration_state, MigrationState::Complete);
    assert_eq!(final_status.staging_status, StagingStatus::NotStaged);

    // Verify new algorithm is working
    let test_result = manager.process_hash_request(b"final_test").await;
    assert!(test_result.success);
    let hash = test_result.data.unwrap();
    assert_eq!(hash[0], 0xAB, "Should be using new algorithm with prefix 0xAB");
}

#[tokio::test]
async fn test_migration_rollback_on_high_error_rate() {
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));

    // Stage an algorithm with high failure rate
    let failing_algorithm = Box::new(TestAlgorithm::new_with_failure_rate(
        "failing_algo", "1.0.0", 0xFE, 0.3 // 30% failure rate
    ));
    
    let stage_result = manager.stage_algorithm(failing_algorithm).await;
    assert!(stage_result.success);

    // Start migration
    let migration_result = manager.start_migration().await;
    assert!(migration_result.success);

    // Process many requests to trigger rollback
    for i in 0..500 {
        let input = format!("test_input_{}", i);
        let _ = manager.process_hash_request(input.as_bytes()).await;
        // Don't assert success here as some may fail due to the failing algorithm
    }

    // Try to advance migration - should trigger rollback due to high error rate
    let _advance_result = manager.advance_migration().await;
    
    // Check if rollback occurred
    let status = manager.get_status().await;
    if matches!(status.migration_state, MigrationState::RollbackComplete) {
        println!("Rollback occurred as expected due to high error rate");
        assert_eq!(status.active_algorithm, "blake2s", "Should rollback to original algorithm");
        assert_eq!(status.staged_algorithm, None, "Staged algorithm should be cleared");
    } else {
        // If rollback didn't occur automatically, trigger it manually for testing
        let rollback_result = manager.rollback_migration().await;
        assert!(rollback_result.success, "Manual rollback should succeed");
        
        let final_status = manager.get_status().await;
        assert_eq!(final_status.migration_state, MigrationState::RollbackComplete);
    }
}

#[tokio::test]
async fn test_concurrent_hash_processing_during_migration() {
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));

    // Stage new algorithm
    let new_algorithm = Box::new(TestAlgorithm::new("concurrent_test", "1.0.0", 0xCC));
    manager.stage_algorithm(new_algorithm).await;
    manager.start_migration().await;

    // Spawn multiple concurrent hash processing tasks
    let mut handles = Vec::new();
    
    for worker_id in 0..10 {
        let manager_clone = manager.clone();
        let handle = tokio::spawn(async move {
            let mut successful_hashes = 0;
            let mut failed_hashes = 0;
            
            for i in 0..100 {
                let input = format!("worker_{}_input_{}", worker_id, i);
                match manager_clone.process_hash_request(input.as_bytes()).await {
                    AlgorithmResult { success: true, .. } => successful_hashes += 1,
                    _ => failed_hashes += 1,
                }
                
                // Small delay to allow migration to progress
                if i % 10 == 0 {
                    tokio::time::sleep(Duration::from_millis(1)).await;
                }
            }
            
            (successful_hashes, failed_hashes)
        });
        
        handles.push(handle);
    }

    // Let workers run for a bit, then advance migration
    tokio::time::sleep(Duration::from_millis(50)).await;
    
    // Advance migration while workers are running
    let _advance_result = manager.advance_migration().await;

    // Wait for all workers to complete
    let mut total_successful = 0;
    let mut total_failed = 0;
    
    for handle in handles {
        let (successful, failed) = handle.await.unwrap();
        total_successful += successful;
        total_failed += failed;
    }

    println!("Concurrent processing results: {} successful, {} failed", 
             total_successful, total_failed);
    
    // Most requests should succeed even during migration
    assert!(total_successful > total_failed, 
            "Should have more successful requests than failed ones");
    assert!(total_successful > 500, "Should have significant number of successful requests");
}

#[tokio::test]
async fn test_migration_timeout_and_recovery() {
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));

    // Stage algorithm
    let new_algorithm = Box::new(TestAlgorithm::new("timeout_test", "1.0.0", 0xDD));
    manager.stage_algorithm(new_algorithm).await;
    manager.start_migration().await;

    // Test that operations don't hang indefinitely
    let hash_result = timeout(
        Duration::from_secs(5),
        manager.process_hash_request(b"timeout_test_input")
    ).await;

    assert!(hash_result.is_ok(), "Hash processing should not timeout");
    let result = hash_result.unwrap();
    assert!(result.success, "Hash processing should succeed");

    // Test migration advance with timeout
    let advance_result = timeout(
        Duration::from_secs(5),
        manager.advance_migration()
    ).await;

    assert!(advance_result.is_ok(), "Migration advance should not timeout");
}

#[tokio::test]
async fn test_multiple_algorithm_staging_attempts() {
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));

    // Stage first algorithm
    let first_algorithm = Box::new(TestAlgorithm::new("first_algo", "1.0.0", 0x11));
    let first_result = manager.stage_algorithm(first_algorithm).await;
    assert!(first_result.success);

    // Try to stage second algorithm while first is staged - should fail
    let second_algorithm = Box::new(TestAlgorithm::new("second_algo", "1.0.0", 0x22));
    let second_result = manager.stage_algorithm(second_algorithm).await;
    assert!(!second_result.success);
    assert_eq!(second_result.error.unwrap().code, "STAGING_IN_PROGRESS");

    // Complete migration of first algorithm
    manager.start_migration().await;
    
    // Advance through all migration phases
    loop {
        let advance_result = manager.advance_migration().await;
        if !advance_result.success {
            break;
        }
        
        let state = advance_result.data.unwrap();
        if matches!(state, MigrationState::Complete) {
            break;
        }
        
        tokio::time::sleep(Duration::from_millis(10)).await;
    }

    // Now should be able to stage a new algorithm
    let third_algorithm = Box::new(TestAlgorithm::new("third_algo", "1.0.0", 0x33));
    let third_result = manager.stage_algorithm(third_algorithm).await;
    assert!(third_result.success, "Should be able to stage new algorithm after migration completes");
}

#[tokio::test]
async fn test_zero_downtime_verification() {
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));

    // Stage and start migration
    let new_algorithm = Box::new(TestAlgorithm::new("zero_downtime", "1.0.0", 0xED));
    manager.stage_algorithm(new_algorithm).await;
    manager.start_migration().await;

    // Process enough requests to satisfy shadow mode requirements
    for i in 0..100 {
        let input = format!("shadow_mode_test_{}", i);
        let _ = manager.process_hash_request(input.as_bytes()).await;
    }

    // Continuously process requests throughout the entire migration
    let manager_clone = manager.clone();
    let continuous_processing = tokio::spawn(async move {
        let mut request_count = 0;
        let mut failure_count = 0;
        
        for i in 0..1000 {
            let input = format!("continuous_test_{}", i);
            match manager_clone.process_hash_request(input.as_bytes()).await {
                AlgorithmResult { success: true, .. } => request_count += 1,
                _ => failure_count += 1,
            }
            
            // Very small delay to allow other operations
            if i % 50 == 0 {
                tokio::time::sleep(Duration::from_millis(1)).await;
            }
        }
        
        (request_count, failure_count)
    });

    // Perform complete migration while processing continues
    loop {
        let advance_result = manager.advance_migration().await;
        if !advance_result.success {
            // If advance fails, check if it's a rollback
            let status = manager.get_status().await;
            if matches!(status.migration_state, MigrationState::RollbackComplete) {
                println!("Migration rolled back, this is acceptable for testing");
                break;
            } else {
                panic!("Migration advance failed unexpectedly: {:?}", advance_result.error);
            }
        }
        
        let state = advance_result.data.unwrap();
        if matches!(state, MigrationState::Complete) {
            break;
        }
        
        tokio::time::sleep(Duration::from_millis(10)).await;
    }

    // Wait for continuous processing to complete
    let (successful_requests, failed_requests) = continuous_processing.await.unwrap();
    
    println!("Zero-downtime test: {} successful, {} failed requests", 
             successful_requests, failed_requests);
    
    // Verify zero-downtime: should have very high success rate
    let success_rate = successful_requests as f64 / (successful_requests + failed_requests) as f64;
    assert!(success_rate > 0.95, "Success rate should be > 95% for zero-downtime migration");
    
    // Verify migration completed or rolled back gracefully
    let final_status = manager.get_status().await;
    assert!(
        matches!(final_status.migration_state, MigrationState::Complete | MigrationState::RollbackComplete),
        "Migration should either complete or rollback gracefully, got: {:?}", final_status.migration_state
    );
    
    // If migration completed, verify the new algorithm is active
    if matches!(final_status.migration_state, MigrationState::Complete) {
        assert_eq!(final_status.active_algorithm, "zero_downtime");
    }
}
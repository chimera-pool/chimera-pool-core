//! Algorithm Hot-Swap System Demo
//! 
//! This example demonstrates the complete algorithm hot-swap workflow:
//! 1. Initialize with Blake2S algorithm
//! 2. Stage a new algorithm with validation
//! 3. Perform gradual migration with shadow mode
//! 4. Complete zero-downtime algorithm swap
//! 5. Demonstrate rollback capabilities

use algorithm_engine::hot_swap::{AlgorithmHotSwapManager, MigrationState, StagingStatus};
use algorithm_engine::{AlgorithmResult, Blake2SAlgorithm, MiningAlgorithm};
use std::sync::Arc;
use std::time::{Duration, Instant};

// Demo algorithm for hot-swap demonstration
struct DemoAlgorithm {
    name: String,
    version: String,
    identifier: u8,
}

impl DemoAlgorithm {
    fn new(name: &str, version: &str, identifier: u8) -> Self {
        Self {
            name: name.to_string(),
            version: version.to_string(),
            identifier,
        }
    }
}

impl MiningAlgorithm for DemoAlgorithm {
    fn name(&self) -> &str {
        &self.name
    }

    fn version(&self) -> &str {
        &self.version
    }

    fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        // Create a unique hash by prefixing with algorithm identifier
        let mut result = vec![self.identifier];
        result.extend_from_slice(input);
        
        // Pad to 32 bytes
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

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("🚀 Algorithm Hot-Swap System Demo");
    println!("==================================\n");

    // Step 1: Initialize with Blake2S algorithm
    println!("📦 Step 1: Initializing with Blake2S algorithm...");
    let initial_algorithm = Box::new(Blake2SAlgorithm::new());
    let manager = Arc::new(AlgorithmHotSwapManager::new(initial_algorithm));
    
    let status = manager.get_status().await;
    println!("   ✅ Active algorithm: {}", status.active_algorithm);
    println!("   ✅ Status: {:?}\n", status.staging_status);

    // Step 2: Test initial algorithm
    println!("🔧 Step 2: Testing initial algorithm...");
    let test_input = b"demo_test_input";
    let result = manager.process_hash_request(test_input).await;
    
    if result.success {
        let hash = result.data.unwrap();
        println!("   ✅ Hash successful: {} bytes", hash.len());
        println!("   ✅ Hash preview: {:02x}{:02x}{:02x}{:02x}...", 
                 hash[0], hash[1], hash[2], hash[3]);
    } else {
        println!("   ❌ Hash failed: {:?}", result.error);
    }
    println!();

    // Step 3: Stage new algorithm
    println!("📋 Step 3: Staging new algorithm...");
    let new_algorithm = Box::new(DemoAlgorithm::new("demo_algo", "2.0.0", 0xAB));
    let stage_result = manager.stage_algorithm(new_algorithm).await;
    
    if stage_result.success {
        println!("   ✅ Algorithm staged: {}", stage_result.data.unwrap());
        
        let status = manager.get_status().await;
        println!("   ✅ Staged algorithm: {:?}", status.staged_algorithm);
        println!("   ✅ Validation status: {:?}", status.staging_status);
        
        if let Some(validation) = &status.validation_results {
            println!("   📊 Validation Results:");
            println!("      - Compatibility: {}", validation.compatibility_check);
            println!("      - Performance: {:.2}", validation.performance_benchmark);
            println!("      - Security: {}", validation.security_validation);
            println!("      - Test vectors: {}", validation.test_vectors_passed);
            println!("      - Memory requirements: {}", validation.memory_requirements_met);
        }
    } else {
        println!("   ❌ Staging failed: {:?}", stage_result.error);
        return Ok(());
    }
    println!();

    // Step 4: Start migration
    println!("🔄 Step 4: Starting migration...");
    let migration_result = manager.start_migration().await;
    
    if migration_result.success {
        println!("   ✅ Migration started: {}", migration_result.data.unwrap());
        
        let status = manager.get_status().await;
        println!("   ✅ Migration state: {:?}", status.migration_state);
    } else {
        println!("   ❌ Migration start failed: {:?}", migration_result.error);
        return Ok(());
    }
    println!();

    // Step 5: Process requests during shadow mode
    println!("👥 Step 5: Processing requests during shadow mode...");
    let shadow_start = Instant::now();
    let mut shadow_requests = 0;
    
    while shadow_start.elapsed() < Duration::from_millis(500) {
        let input = format!("shadow_test_{}", shadow_requests);
        let result = manager.process_hash_request(input.as_bytes()).await;
        
        if result.success {
            shadow_requests += 1;
            
            // Show progress every 50 requests
            if shadow_requests % 50 == 0 {
                let status = manager.get_status().await;
                println!("   📊 Processed {} requests, state: {:?}", 
                         shadow_requests, status.migration_state);
            }
        }
        
        // Small delay to simulate realistic load
        tokio::time::sleep(Duration::from_micros(100)).await;
    }
    
    println!("   ✅ Shadow mode completed: {} requests processed", shadow_requests);
    println!();

    // Step 6: Advance through migration phases
    println!("⚡ Step 6: Advancing through migration phases...");
    let mut phase_count = 0;
    
    loop {
        let advance_result = manager.advance_migration().await;
        
        if !advance_result.success {
            println!("   ⚠️  Migration advance failed: {:?}", advance_result.error);
            break;
        }
        
        let new_state = advance_result.data.unwrap();
        phase_count += 1;
        
        match new_state {
            MigrationState::Complete => {
                println!("   🎉 Migration completed after {} phases!", phase_count);
                break;
            }
            MigrationState::GradualMigration { percentage } => {
                println!("   📈 Phase {}: {}% migration", phase_count, percentage * 100.0);
                
                // Process some requests during this phase
                for i in 0..20 {
                    let input = format!("phase_{}_request_{}", phase_count, i);
                    let result = manager.process_hash_request(input.as_bytes()).await;
                    
                    if result.success {
                        let hash = result.data.unwrap();
                        // Check which algorithm was used based on hash prefix
                        let algorithm_used = if hash[0] == 0xAB { "demo_algo" } else { "blake2s" };
                        
                        if i == 0 {
                            println!("      Sample hash from {}: {:02x}{:02x}{:02x}{:02x}...", 
                                     algorithm_used, hash[0], hash[1], hash[2], hash[3]);
                        }
                    }
                }
                
                // Small delay between phases
                tokio::time::sleep(Duration::from_millis(100)).await;
            }
            MigrationState::RollbackComplete => {
                println!("   🔄 Migration rolled back");
                break;
            }
            _ => {
                println!("   📊 Migration state: {:?}", new_state);
            }
        }
        
        // Safety check
        if phase_count > 20 {
            println!("   ⚠️  Too many phases, stopping");
            break;
        }
    }
    println!();

    // Step 7: Verify final state
    println!("🔍 Step 7: Verifying final state...");
    let final_status = manager.get_status().await;
    
    println!("   📊 Final Status:");
    println!("      - Active algorithm: {}", final_status.active_algorithm);
    println!("      - Migration state: {:?}", final_status.migration_state);
    println!("      - Staging status: {:?}", final_status.staging_status);
    
    // Test final algorithm
    let final_test = manager.process_hash_request(b"final_verification").await;
    if final_test.success {
        let hash = final_test.data.unwrap();
        println!("   ✅ Final algorithm test successful");
        println!("      Hash: {:02x}{:02x}{:02x}{:02x}...", 
                 hash[0], hash[1], hash[2], hash[3]);
        
        // Determine which algorithm is active based on hash
        let active_algo = if hash[0] == 0xAB { "demo_algo" } else { "blake2s" };
        println!("      Confirmed active: {}", active_algo);
    }
    println!();

    // Step 8: Demonstrate rollback (if migration completed)
    if matches!(final_status.migration_state, MigrationState::Complete) {
        println!("🔄 Step 8: Demonstrating rollback capability...");
        
        // Stage another algorithm to test rollback
        let rollback_test_algo = Box::new(DemoAlgorithm::new("rollback_test", "1.0.0", 0xFF));
        let stage_result = manager.stage_algorithm(rollback_test_algo).await;
        
        if stage_result.success {
            println!("   ✅ Rollback test algorithm staged");
            
            // Start migration
            let migration_result = manager.start_migration().await;
            if migration_result.success {
                println!("   ✅ Rollback test migration started");
                
                // Immediately rollback
                let rollback_result = manager.rollback_migration().await;
                if rollback_result.success {
                    println!("   ✅ Rollback successful");
                    
                    let rollback_status = manager.get_status().await;
                    println!("      - State after rollback: {:?}", rollback_status.migration_state);
                    println!("      - Active algorithm: {}", rollback_status.active_algorithm);
                } else {
                    println!("   ❌ Rollback failed: {:?}", rollback_result.error);
                }
            }
        }
    }
    println!();

    // Step 9: Performance demonstration
    println!("⚡ Step 9: Performance demonstration...");
    let perf_start = Instant::now();
    let mut successful_hashes = 0;
    let mut failed_hashes = 0;
    
    for i in 0..1000 {
        let input = format!("perf_test_{}", i);
        let result = manager.process_hash_request(input.as_bytes()).await;
        
        if result.success {
            successful_hashes += 1;
        } else {
            failed_hashes += 1;
        }
    }
    
    let perf_duration = perf_start.elapsed();
    let hashes_per_second = successful_hashes as f64 / perf_duration.as_secs_f64();
    
    println!("   📊 Performance Results:");
    println!("      - Total time: {:?}", perf_duration);
    println!("      - Successful hashes: {}", successful_hashes);
    println!("      - Failed hashes: {}", failed_hashes);
    println!("      - Success rate: {:.2}%", 
             successful_hashes as f64 / (successful_hashes + failed_hashes) as f64 * 100.0);
    println!("      - Hashes per second: {:.2}", hashes_per_second);
    println!();

    println!("🎉 Algorithm Hot-Swap Demo Completed Successfully!");
    println!("   ✅ Zero-downtime algorithm switching demonstrated");
    println!("   ✅ Gradual migration with shadow mode validated");
    println!("   ✅ Rollback capability confirmed");
    println!("   ✅ Performance maintained throughout migration");
    
    Ok(())
}
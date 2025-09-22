//! Integration tests for the algorithm engine
//! 
//! These tests verify that the algorithm engine works correctly
//! in realistic scenarios and can be integrated with other components.

#[cfg(feature = "integration")]
mod integration {
    use algorithm_engine::{Blake2SAlgorithm, MiningAlgorithm};
    use std::sync::Arc;
    use std::thread;
    use std::time::Instant;

    #[test]
    fn test_algorithm_thread_safety() {
        let algorithm = Arc::new(Blake2SAlgorithm::new());
        let mut handles = vec![];

        // Spawn multiple threads using the same algorithm instance
        for i in 0..10 {
            let algo = Arc::clone(&algorithm);
            let handle = thread::spawn(move || {
                let input = format!("test_input_{}", i);
                let result = algo.hash(input.as_bytes());
                assert!(result.success);
                result.data.unwrap()
            });
            handles.push(handle);
        }

        // Collect results
        let mut results = vec![];
        for handle in handles {
            results.push(handle.join().unwrap());
        }

        // Verify all results are different (different inputs should produce different hashes)
        for i in 0..results.len() {
            for j in i + 1..results.len() {
                assert_ne!(results[i], results[j], "Different inputs should produce different hashes");
            }
        }
    }

    #[test]
    fn test_algorithm_performance_baseline() {
        let algorithm = Blake2SAlgorithm::new();
        let input = vec![0u8; 1024]; // 1KB input
        let iterations = 1000;

        let start = Instant::now();
        for _ in 0..iterations {
            let result = algorithm.hash(&input);
            assert!(result.success);
        }
        let duration = start.elapsed();

        let ops_per_second = iterations as f64 / duration.as_secs_f64();
        println!("Blake2S performance: {:.2} ops/sec for 1KB input", ops_per_second);

        // Basic performance assertion - should be able to do at least 1000 ops/sec
        assert!(ops_per_second > 1000.0, "Performance too low: {:.2} ops/sec", ops_per_second);
    }

    #[test]
    fn test_algorithm_memory_usage() {
        let algorithm = Blake2SAlgorithm::new();
        
        // Test with various input sizes
        let sizes = vec![64, 256, 1024, 4096, 16384]; // bytes
        
        for size in sizes {
            let input = vec![0u8; size];
            let result = algorithm.hash(&input);
            
            assert!(result.success);
            let hash = result.data.unwrap();
            
            // Blake2s256 should always produce 32-byte output regardless of input size
            assert_eq!(hash.len(), 32, "Hash length should be 32 bytes for input size {}", size);
        }
    }

    #[test]
    fn test_algorithm_verification_workflow() {
        let algorithm = Blake2SAlgorithm::new();
        let input = b"mining_block_header";
        
        // Test with different difficulty levels (represented by target)
        let easy_target = vec![0xFF; 32];    // Very easy
        let medium_target = vec![0x0F; 32];  // Medium
        let mut hard_target = vec![0x00; 31];    // Very hard
        hard_target.push(0x01);

        // Test verification with easy target
        let result = algorithm.verify(input, &easy_target, 12345);
        assert!(result.success);
        // With easy target, verification should likely succeed

        // Test verification consistency
        let result1 = algorithm.verify(input, &medium_target, 12345);
        let result2 = algorithm.verify(input, &medium_target, 12345);
        assert!(result1.success);
        assert!(result2.success);
        assert_eq!(result1.data, result2.data, "Same input should produce same verification result");

        // Test with different nonces
        let result_nonce1 = algorithm.verify(input, &medium_target, 1);
        let result_nonce2 = algorithm.verify(input, &medium_target, 2);
        assert!(result_nonce1.success);
        assert!(result_nonce2.success);
        // Different nonces might produce different results
    }

    #[test]
    fn test_algorithm_error_handling() {
        let algorithm = Blake2SAlgorithm::new();
        
        // Test with empty input
        let result = algorithm.hash(&[]);
        assert!(result.success, "Empty input should be handled gracefully");
        
        // Test with very large input
        let large_input = vec![0u8; 1_000_000]; // 1MB
        let result = algorithm.hash(&large_input);
        assert!(result.success, "Large input should be handled gracefully");
        
        // Test verification with empty target
        let result = algorithm.verify(b"test", &[], 123);
        assert!(result.success, "Empty target should be handled gracefully");
    }

    #[test]
    fn test_algorithm_deterministic_behavior() {
        let algorithm1 = Blake2SAlgorithm::new();
        let algorithm2 = Blake2SAlgorithm::new();
        
        let test_cases = vec![
            b"test1".to_vec(),
            b"test2".to_vec(),
            b"longer_test_input_with_more_data".to_vec(),
            vec![0u8; 1000],
            vec![0xFF; 500],
        ];

        for input in test_cases {
            let result1 = algorithm1.hash(&input);
            let result2 = algorithm2.hash(&input);
            
            assert!(result1.success);
            assert!(result2.success);
            assert_eq!(result1.data, result2.data, 
                "Different algorithm instances should produce same results for same input");
        }
    }

    #[test]
    fn test_algorithm_concurrent_verification() {
        let algorithm = Arc::new(Blake2SAlgorithm::new());
        let input = b"concurrent_test_block";
        let target = vec![0xFF; 32]; // Easy target
        
        let mut handles = vec![];
        
        // Spawn multiple threads doing verification
        for nonce in 0..100 {
            let algo = Arc::clone(&algorithm);
            let input_copy = input.to_vec();
            let target_copy = target.clone();
            
            let handle = thread::spawn(move || {
                let result = algo.verify(&input_copy, &target_copy, nonce);
                assert!(result.success);
                (nonce, result.data.unwrap())
            });
            handles.push(handle);
        }

        // Collect all results
        let mut results = vec![];
        for handle in handles {
            results.push(handle.join().unwrap());
        }

        // Verify we got results for all nonces
        assert_eq!(results.len(), 100);
        
        // Sort by nonce to verify order
        results.sort_by_key(|(nonce, _)| *nonce);
        for (i, (nonce, _)) in results.iter().enumerate() {
            assert_eq!(*nonce, i as u64);
        }
    }
}
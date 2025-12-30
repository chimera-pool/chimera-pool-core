//! Chimera Pool Algorithm Engine
//! 
//! Hot-swappable mining algorithm engine for universal cryptocurrency support.
//! 
//! Supported algorithms:
//! - Blake2S (BlockDAG)
//! - Scrypt (Litecoin, Dogecoin)
//! - SHA256d (Bitcoin, Bitcoin Cash)

pub mod hot_swap;
pub mod algorithms;

pub use algorithms::{ScryptAlgorithm, Sha256dAlgorithm, AlgorithmRegistry};

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Result type for algorithm operations with comprehensive error handling
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlgorithmResult<T> {
    pub success: bool,
    pub data: Option<T>,
    pub error: Option<AlgorithmError>,
}

/// Comprehensive error information for algorithm operations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlgorithmError {
    pub code: String,
    pub message: String,
    pub details: HashMap<String, String>,
}

impl<T> AlgorithmResult<T> {
    /// Create a successful result
    pub fn success(data: T) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
        }
    }
    
    /// Create an error result
    pub fn error(code: &str, message: &str) -> Self {
        Self {
            success: false,
            data: None,
            error: Some(AlgorithmError {
                code: code.to_string(),
                message: message.to_string(),
                details: HashMap::new(),
            }),
        }
    }
}

/// Core trait for mining algorithms
pub trait MiningAlgorithm: Send + Sync {
    /// Get algorithm name
    fn name(&self) -> &str;
    
    /// Get algorithm version
    fn version(&self) -> &str;
    
    /// Hash input data
    fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>>;
    
    /// Verify hash against target
    fn verify(&self, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool>;
}

/// Example Blake2S algorithm implementation for testing
pub struct Blake2SAlgorithm {
    name: String,
    version: String,
}

impl Blake2SAlgorithm {
    pub fn new() -> Self {
        Self {
            name: "blake2s".to_string(),
            version: "1.0.0".to_string(),
        }
    }
}

impl Default for Blake2SAlgorithm {
    fn default() -> Self {
        Self::new()
    }
}

impl MiningAlgorithm for Blake2SAlgorithm {
    fn name(&self) -> &str {
        &self.name
    }
    
    fn version(&self) -> &str {
        &self.version
    }
    
    fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        use blake2::{Blake2s256, Digest};
        
        // Create Blake2s hasher with default parameters (32-byte output)
        let mut hasher = Blake2s256::new();
        hasher.update(input);
        let result = hasher.finalize();
        
        AlgorithmResult::success(result.to_vec())
    }
    
    fn verify(&self, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool> {
        // Combine input with nonce (little-endian format)
        let mut data = input.to_vec();
        data.extend_from_slice(&nonce.to_le_bytes());
        
        match self.hash(&data) {
            AlgorithmResult { success: true, data: Some(hash), .. } => {
                // Check if hash meets target difficulty
                // In mining, we compare the hash as a big-endian number against the target
                let meets_target = if target.is_empty() {
                    // Empty target means impossible difficulty
                    false
                } else if target.iter().all(|&b| b == 0xFF) {
                    // All FF target means very easy (always passes)
                    true
                } else {
                    // Compare hash bytes against target bytes (big-endian comparison)
                    hash.len() == target.len() && hash.as_slice() < target
                };
                AlgorithmResult::success(meets_target)
            }
            _result => AlgorithmResult::error("HASH_FAILED", "Failed to hash input for verification"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_blake2s_algorithm_creation() {
        let algorithm = Blake2SAlgorithm::new();
        assert_eq!(algorithm.name(), "blake2s");
        assert_eq!(algorithm.version(), "1.0.0");
    }
    
    #[test]
    fn test_blake2s_hash() {
        let algorithm = Blake2SAlgorithm::new();
        let input = b"test input";
        
        let result = algorithm.hash(input);
        assert!(result.success);
        assert!(result.data.is_some());
        assert_eq!(result.data.unwrap().len(), 32); // Blake2s256 produces 32-byte hash
    }
    
    #[test]
    fn test_blake2s_hash_consistency() {
        let algorithm = Blake2SAlgorithm::new();
        let input = b"consistent input";
        
        let result1 = algorithm.hash(input);
        let result2 = algorithm.hash(input);
        
        assert!(result1.success);
        assert!(result2.success);
        assert_eq!(result1.data, result2.data);
    }
    
    #[test]
    fn test_blake2s_verify() {
        let algorithm = Blake2SAlgorithm::new();
        let input = b"verification test";
        let target = vec![0xFF; 32]; // Very easy target (all bits set)
        let nonce = 12345u64;
        
        let result = algorithm.verify(input, &target, nonce);
        assert!(result.success);
        // With an easy target, verification should succeed
        assert!(result.data.unwrap_or(false));
    }
    
    #[test]
    fn test_algorithm_result_success() {
        let result: AlgorithmResult<String> = AlgorithmResult::success("test".to_string());
        assert!(result.success);
        assert_eq!(result.data.unwrap(), "test");
        assert!(result.error.is_none());
    }
    
    #[test]
    fn test_algorithm_result_error() {
        let result: AlgorithmResult<String> = AlgorithmResult::error("TEST_ERROR", "Test error message");
        assert!(!result.success);
        assert!(result.data.is_none());
        assert!(result.error.is_some());
        
        let error = result.error.unwrap();
        assert_eq!(error.code, "TEST_ERROR");
        assert_eq!(error.message, "Test error message");
    }
}

// Property-based tests using proptest
#[cfg(all(test, feature = "proptest"))]
mod property_tests {
    use super::*;
    use proptest::prelude::*;
    
    proptest! {
        #[test]
        fn test_hash_deterministic(input in prop::collection::vec(any::<u8>(), 0..1000)) {
            let algorithm = Blake2SAlgorithm::new();
            
            let result1 = algorithm.hash(&input);
            let result2 = algorithm.hash(&input);
            
            prop_assert!(result1.success);
            prop_assert!(result2.success);
            prop_assert_eq!(result1.data, result2.data);
        }
        
        #[test]
        fn test_hash_different_inputs_different_outputs(
            input1 in prop::collection::vec(any::<u8>(), 1..100),
            input2 in prop::collection::vec(any::<u8>(), 1..100)
        ) {
            prop_assume!(input1 != input2);
            
            let algorithm = Blake2SAlgorithm::new();
            
            let result1 = algorithm.hash(&input1);
            let result2 = algorithm.hash(&input2);
            
            prop_assert!(result1.success);
            prop_assert!(result2.success);
            prop_assert_ne!(result1.data, result2.data);
        }
        
        #[test]
        fn test_verify_with_different_nonces(
            input in prop::collection::vec(any::<u8>(), 1..100),
            nonce1 in any::<u64>(),
            nonce2 in any::<u64>()
        ) {
            prop_assume!(nonce1 != nonce2);
            
            let algorithm = Blake2SAlgorithm::new();
            let target = vec![0xFF; 32]; // Easy target
            
            let result1 = algorithm.verify(&input, &target, nonce1);
            let result2 = algorithm.verify(&input, &target, nonce2);
            
            prop_assert!(result1.success);
            prop_assert!(result2.success);
            // Different nonces should potentially give different results
            // (though with an easy target, both might succeed)
        }
    }
}
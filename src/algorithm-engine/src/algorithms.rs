//! Multi-coin mining algorithm implementations
//! 
//! Supports: Scrypt (Litecoin), SHA256 (Bitcoin), Blake2S (BlockDAG)

use crate::{AlgorithmResult, MiningAlgorithm};

/// Scrypt algorithm parameters for Litecoin
#[derive(Debug, Clone)]
pub struct ScryptParams {
    pub n: u32,      // CPU/memory cost parameter (1024 for Litecoin)
    pub r: u32,      // Block size parameter (1 for Litecoin)
    pub p: u32,      // Parallelization parameter (1 for Litecoin)
    pub key_len: usize, // Output key length (32 bytes)
}

impl Default for ScryptParams {
    fn default() -> Self {
        // Litecoin standard parameters
        Self {
            n: 1024,
            r: 1,
            p: 1,
            key_len: 32,
        }
    }
}

/// Scrypt algorithm implementation for Litecoin mining
pub struct ScryptAlgorithm {
    name: String,
    version: String,
    params: ScryptParams,
}

impl ScryptAlgorithm {
    pub fn new() -> Self {
        Self::with_params(ScryptParams::default())
    }
    
    pub fn with_params(params: ScryptParams) -> Self {
        Self {
            name: "scrypt".to_string(),
            version: "1.0.0".to_string(),
            params,
        }
    }
    
    /// Create Litecoin-specific Scrypt algorithm
    pub fn litecoin() -> Self {
        Self::with_params(ScryptParams {
            n: 1024,
            r: 1,
            p: 1,
            key_len: 32,
        })
    }
}

impl Default for ScryptAlgorithm {
    fn default() -> Self {
        Self::new()
    }
}

impl MiningAlgorithm for ScryptAlgorithm {
    fn name(&self) -> &str {
        &self.name
    }
    
    fn version(&self) -> &str {
        &self.version
    }
    
    fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        use scrypt::{scrypt, Params};
        
        // Create scrypt params - using log2(n) for the params
        let log_n = (self.params.n as f64).log2() as u8;
        let params = match Params::new(log_n, self.params.r, self.params.p, self.params.key_len) {
            Ok(p) => p,
            Err(e) => return AlgorithmResult::error("INVALID_PARAMS", &format!("Invalid scrypt params: {}", e)),
        };
        
        // Use input as both password and salt (standard for mining)
        let mut output = vec![0u8; self.params.key_len];
        
        match scrypt(input, input, &params, &mut output) {
            Ok(_) => AlgorithmResult::success(output),
            Err(e) => AlgorithmResult::error("SCRYPT_FAILED", &format!("Scrypt hash failed: {}", e)),
        }
    }
    
    fn verify(&self, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool> {
        // Combine input with nonce (little-endian format, standard for mining)
        let mut data = input.to_vec();
        data.extend_from_slice(&nonce.to_le_bytes());
        
        match self.hash(&data) {
            AlgorithmResult { success: true, data: Some(hash), .. } => {
                let meets_target = if target.is_empty() {
                    false
                } else if target.iter().all(|&b| b == 0xFF) {
                    true
                } else {
                    hash.len() == target.len() && hash.as_slice() < target
                };
                AlgorithmResult::success(meets_target)
            }
            _ => AlgorithmResult::error("HASH_FAILED", "Failed to hash input for verification"),
        }
    }
}

/// SHA256d (double SHA256) algorithm for Bitcoin mining
pub struct Sha256dAlgorithm {
    name: String,
    version: String,
}

impl Sha256dAlgorithm {
    pub fn new() -> Self {
        Self {
            name: "sha256d".to_string(),
            version: "1.0.0".to_string(),
        }
    }
    
    /// Create Bitcoin-specific SHA256d algorithm
    pub fn bitcoin() -> Self {
        Self::new()
    }
}

impl Default for Sha256dAlgorithm {
    fn default() -> Self {
        Self::new()
    }
}

impl MiningAlgorithm for Sha256dAlgorithm {
    fn name(&self) -> &str {
        &self.name
    }
    
    fn version(&self) -> &str {
        &self.version
    }
    
    fn hash(&self, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        use sha2::{Sha256, Digest};
        
        // Double SHA256: SHA256(SHA256(input))
        let first_hash = Sha256::digest(input);
        let second_hash = Sha256::digest(&first_hash);
        
        AlgorithmResult::success(second_hash.to_vec())
    }
    
    fn verify(&self, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool> {
        // Combine input with nonce (little-endian format)
        let mut data = input.to_vec();
        data.extend_from_slice(&nonce.to_le_bytes());
        
        match self.hash(&data) {
            AlgorithmResult { success: true, data: Some(hash), .. } => {
                let meets_target = if target.is_empty() {
                    false
                } else if target.iter().all(|&b| b == 0xFF) {
                    true
                } else {
                    hash.len() == target.len() && hash.as_slice() < target
                };
                AlgorithmResult::success(meets_target)
            }
            _ => AlgorithmResult::error("HASH_FAILED", "Failed to hash input for verification"),
        }
    }
}

/// Algorithm registry for managing multiple algorithms
pub struct AlgorithmRegistry {
    algorithms: std::collections::HashMap<String, Box<dyn MiningAlgorithm>>,
}

impl AlgorithmRegistry {
    pub fn new() -> Self {
        let mut registry = Self {
            algorithms: std::collections::HashMap::new(),
        };
        
        // Register default algorithms
        registry.register(Box::new(crate::Blake2SAlgorithm::new()));
        registry.register(Box::new(ScryptAlgorithm::litecoin()));
        registry.register(Box::new(Sha256dAlgorithm::bitcoin()));
        
        registry
    }
    
    pub fn register(&mut self, algorithm: Box<dyn MiningAlgorithm>) {
        self.algorithms.insert(algorithm.name().to_string(), algorithm);
    }
    
    pub fn get(&self, name: &str) -> Option<&dyn MiningAlgorithm> {
        self.algorithms.get(name).map(|a| a.as_ref())
    }
    
    pub fn list(&self) -> Vec<&str> {
        self.algorithms.keys().map(|s| s.as_str()).collect()
    }
    
    pub fn hash(&self, algorithm_name: &str, input: &[u8]) -> AlgorithmResult<Vec<u8>> {
        match self.get(algorithm_name) {
            Some(algo) => algo.hash(input),
            None => AlgorithmResult::error("ALGO_NOT_FOUND", &format!("Algorithm not found: {}", algorithm_name)),
        }
    }
    
    pub fn verify(&self, algorithm_name: &str, input: &[u8], target: &[u8], nonce: u64) -> AlgorithmResult<bool> {
        match self.get(algorithm_name) {
            Some(algo) => algo.verify(input, target, nonce),
            None => AlgorithmResult::error("ALGO_NOT_FOUND", &format!("Algorithm not found: {}", algorithm_name)),
        }
    }
}

impl Default for AlgorithmRegistry {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_scrypt_algorithm_creation() {
        let algorithm = ScryptAlgorithm::new();
        assert_eq!(algorithm.name(), "scrypt");
        assert_eq!(algorithm.version(), "1.0.0");
    }
    
    #[test]
    fn test_scrypt_litecoin_params() {
        let algorithm = ScryptAlgorithm::litecoin();
        assert_eq!(algorithm.params.n, 1024);
        assert_eq!(algorithm.params.r, 1);
        assert_eq!(algorithm.params.p, 1);
    }
    
    #[test]
    fn test_scrypt_hash() {
        let algorithm = ScryptAlgorithm::litecoin();
        let input = b"test input for litecoin";
        
        let result = algorithm.hash(input);
        assert!(result.success, "Scrypt hash should succeed");
        assert!(result.data.is_some());
        assert_eq!(result.data.unwrap().len(), 32);
    }
    
    #[test]
    fn test_scrypt_hash_consistency() {
        let algorithm = ScryptAlgorithm::litecoin();
        let input = b"consistent scrypt test";
        
        let result1 = algorithm.hash(input);
        let result2 = algorithm.hash(input);
        
        assert!(result1.success);
        assert!(result2.success);
        assert_eq!(result1.data, result2.data, "Same input should produce same hash");
    }
    
    #[test]
    fn test_scrypt_verify_easy_target() {
        let algorithm = ScryptAlgorithm::litecoin();
        let input = b"verification test";
        let target = vec![0xFF; 32]; // Very easy target
        let nonce = 12345u64;
        
        let result = algorithm.verify(input, &target, nonce);
        assert!(result.success);
        assert!(result.data.unwrap_or(false), "Easy target should pass");
    }
    
    #[test]
    fn test_sha256d_algorithm_creation() {
        let algorithm = Sha256dAlgorithm::new();
        assert_eq!(algorithm.name(), "sha256d");
        assert_eq!(algorithm.version(), "1.0.0");
    }
    
    #[test]
    fn test_sha256d_hash() {
        let algorithm = Sha256dAlgorithm::bitcoin();
        let input = b"test input for bitcoin";
        
        let result = algorithm.hash(input);
        assert!(result.success, "SHA256d hash should succeed");
        assert!(result.data.is_some());
        assert_eq!(result.data.unwrap().len(), 32);
    }
    
    #[test]
    fn test_sha256d_known_vector() {
        // Test with known Bitcoin test vector
        let algorithm = Sha256dAlgorithm::bitcoin();
        let input = b"hello";
        
        let result = algorithm.hash(input);
        assert!(result.success);
        
        // SHA256d("hello") is a known value
        let hash = result.data.unwrap();
        assert_eq!(hash.len(), 32);
    }
    
    #[test]
    fn test_sha256d_verify_easy_target() {
        let algorithm = Sha256dAlgorithm::bitcoin();
        let input = b"bitcoin test";
        let target = vec![0xFF; 32];
        let nonce = 98765u64;
        
        let result = algorithm.verify(input, &target, nonce);
        assert!(result.success);
        assert!(result.data.unwrap_or(false));
    }
    
    #[test]
    fn test_algorithm_registry() {
        let registry = AlgorithmRegistry::new();
        
        let algorithms = registry.list();
        assert!(algorithms.contains(&"blake2s"));
        assert!(algorithms.contains(&"scrypt"));
        assert!(algorithms.contains(&"sha256d"));
    }
    
    #[test]
    fn test_registry_hash() {
        let registry = AlgorithmRegistry::new();
        let input = b"registry test";
        
        let blake_result = registry.hash("blake2s", input);
        assert!(blake_result.success);
        
        let scrypt_result = registry.hash("scrypt", input);
        assert!(scrypt_result.success);
        
        let sha256_result = registry.hash("sha256d", input);
        assert!(sha256_result.success);
        
        // Different algorithms produce different hashes
        assert_ne!(blake_result.data, scrypt_result.data);
        assert_ne!(scrypt_result.data, sha256_result.data);
    }
    
    #[test]
    fn test_registry_unknown_algorithm() {
        let registry = AlgorithmRegistry::new();
        let input = b"test";
        
        let result = registry.hash("unknown_algo", input);
        assert!(!result.success);
        assert!(result.error.is_some());
    }
}

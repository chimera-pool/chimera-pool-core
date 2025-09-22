use criterion::{black_box, criterion_group, criterion_main, Criterion};
use algorithm_engine::{Blake2SAlgorithm, MiningAlgorithm};

fn benchmark_blake2s_hash(c: &mut Criterion) {
    let algorithm = Blake2SAlgorithm::new();
    let input = vec![0u8; 1024]; // 1KB input
    
    c.bench_function("blake2s_hash_1kb", |b| {
        b.iter(|| {
            algorithm.hash(black_box(&input))
        })
    });
}

fn benchmark_blake2s_verify(c: &mut Criterion) {
    let algorithm = Blake2SAlgorithm::new();
    let input = vec![0u8; 80]; // Typical block header size
    let target = vec![0x00, 0x00, 0x0F, 0xFF]; // Moderate difficulty
    let nonce = 12345u64;
    
    c.bench_function("blake2s_verify", |b| {
        b.iter(|| {
            algorithm.verify(black_box(&input), black_box(&target), black_box(nonce))
        })
    });
}

criterion_group!(benches, benchmark_blake2s_hash, benchmark_blake2s_verify);
criterion_main!(benches);
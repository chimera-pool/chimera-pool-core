# One-Click Installation System

This package implements a comprehensive one-click installation system for the Chimera Mining Pool, providing automated deployment for both pool operators and miners.

## Features

### Pool Operator Installation
- **Auto-Configuration**: Automatically detects system specifications and generates optimal configuration
- **Docker Compose Generation**: Creates production-ready Docker Compose files with all services
- **System Detection**: Detects CPU, memory, storage, and network capabilities
- **Intelligent Optimization**: Configures database connections, worker threads, and resource limits based on hardware
- **SSL Support**: Automated SSL certificate setup and configuration
- **Management Scripts**: Generates start/stop scripts for easy pool management

### Miner Installation
- **Hardware Auto-Detection**: Automatically detects CPUs, GPUs, and their capabilities
- **Optimal Configuration**: Generates mining configuration optimized for detected hardware
- **Wizard Interface**: Step-by-step setup wizard for non-technical users
- **Driver Management**: Detects and helps install missing GPU drivers
- **Cross-Platform**: Supports Windows, macOS, and Linux
- **Auto-Updates**: Built-in update checking and installation

### Cloud Deployment
- **Multi-Cloud Support**: AWS, Google Cloud Platform, and Microsoft Azure
- **Infrastructure as Code**: Generates CloudFormation, Deployment Manager, and ARM templates
- **Cost Estimation**: Provides accurate cost estimates before deployment
- **Security Configuration**: Includes WAF, DDoS protection, and security groups
- **Auto-Scaling**: Configures auto-scaling based on expected load

### Network Discovery
- **mDNS Advertisement**: Automatically advertises pool on local network
- **Pool Discovery**: Discovers other pools on the local network
- **Service Browsing**: Browse for Chimera pool services
- **Conflict Resolution**: Handles name conflicts automatically

## Usage

### Pool Installation

```go
installer := NewPoolInstaller()

// Detect system specifications
specs, err := installer.DetectSystemSpecs()
if err != nil {
    log.Fatal(err)
}

// Generate optimal configuration
config, err := installer.GenerateAutoConfiguration(specs)
if err != nil {
    log.Fatal(err)
}

// One-click installation
installConfig := InstallConfig{
    InstallPath:    "/opt/chimera-pool",
    AutoStart:      true,
    EnableSSL:      true,
    DomainName:     "pool.example.com",
    AdminEmail:     "admin@example.com",
    WalletAddress:  "your_wallet_address",
}

result, err := installer.OneClickInstall(context.Background(), installConfig)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Installation completed: %s\n", result.Status)
```

### Miner Installation

```go
installer := NewMinerInstaller()

// Detect hardware
hardware, err := installer.DetectHardware()
if err != nil {
    log.Fatal(err)
}

// One-click installation
installConfig := MinerInstallConfig{
    InstallPath:   "/opt/chimera-miner",
    PoolURL:       "stratum+tcp://pool.example.com:4444",
    WalletAddress: "your_wallet_address",
    MinerName:     "my-miner",
    AutoStart:     true,
    AutoDetect:    true,
}

result, err := installer.OneClickInstall(context.Background(), installConfig)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Miner installed: %s\n", result.MinerExecutable)
```

### Cloud Deployment

```go
deployer := NewCloudDeployer()

config := CloudDeploymentConfig{
    Provider:     CloudProviderAWS,
    Region:       "us-west-2",
    InstanceType: "t3.medium",
    StorageSize:  100,
    DatabaseConfig: DatabaseConfig{
        InstanceType: "db.t3.micro",
        Storage:      20,
    },
    SecurityConfig: SecurityConfig{
        EnableSSL:     true,
        EnableWAF:     true,
        EnableDDoSProtection: true,
    },
}

// Generate cost estimate
estimate, err := deployer.EstimateCost(config)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Estimated monthly cost: $%.2f\n", estimate.MonthlyTotal)

// Deploy to cloud
result, err := deployer.Deploy(context.Background(), config)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Deployment ID: %s\n", result.DeploymentID)
```

### mDNS Discovery

```go
discovery := NewMDNSDiscovery()

// Advertise pool
poolInfo := PoolAdvertisement{
    Name:        "My Mining Pool",
    ServiceType: "_chimera-pool._tcp",
    Port:        4444,
    Domain:      "local",
    TXTRecords: map[string]string{
        "algorithm": "blake2s",
        "fee":       "1.0",
        "location":  "us-west",
    },
}

err := discovery.AdvertisePool(context.Background(), poolInfo)
if err != nil {
    log.Fatal(err)
}

// Discover pools
pools, err := discovery.DiscoverPools(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, pool := range pools {
    fmt.Printf("Found pool: %s at %s:%d\n", pool.Name, pool.Address, pool.Port)
}
```

## Architecture

### Components

1. **PoolInstaller**: Main installer for pool operators
2. **MinerInstaller**: Installer for miners
3. **SystemDetector**: Hardware and system detection
4. **HardwareDetector**: Detailed hardware detection for miners
5. **CloudDeployer**: Cloud deployment management
6. **MDNSDiscovery**: Network discovery and advertisement
7. **DockerComposer**: Docker Compose file generation
8. **ConfigGenerator**: Configuration file generation

### System Detection

The system detection automatically identifies:
- CPU cores, threads, and architecture
- Memory total and available
- Storage capacity and type
- Network bandwidth and type
- Container runtime support (Docker/Podman)
- GPU information (NVIDIA, AMD, Intel)

### Auto-Configuration

Based on detected hardware, the system automatically configures:
- Maximum concurrent miners
- Database connection pools
- Redis connection limits
- Stratum server worker threads
- Security settings (MFA, rate limiting)
- Monitoring and logging levels

### Cloud Templates

Generates infrastructure templates for:
- **AWS**: CloudFormation templates with EC2, RDS, ElastiCache, VPC, Security Groups
- **GCP**: Deployment Manager templates with Compute Engine, Cloud SQL, Memorystore
- **Azure**: ARM templates with Virtual Machines, SQL Database, Redis Cache

## Testing

The implementation includes comprehensive tests:

```bash
go test ./internal/installer/... -v
```

### Test Coverage

- Pool installer auto-configuration
- System detection across platforms
- Hardware detection for mining
- Docker Compose generation
- Cloud template generation
- mDNS discovery and advertisement
- Cost estimation
- Error handling and validation

## Requirements Satisfied

This implementation satisfies the following requirements:

- **9.1-9.6**: One-click deployment and zero-config setup
- **12.1-12.5**: Intelligent auto-configuration
- **13.1-13.5**: Plug-and-play miner integration
- **24.1-24.6**: One-click miner installation
- **25.1-25.6**: Wizard-driven setup experience
- **26.1-26.6**: Intelligent auto-detection and configuration
- **27.1-27.6**: Universal compatibility and fallback
- **28.1-28.6**: Post-installation support and monitoring

## Security Considerations

- Input validation for all configuration parameters
- Secure default configurations
- SSL/TLS encryption support
- Network security group configuration
- WAF and DDoS protection for cloud deployments
- Secure credential handling

## Platform Support

- **Linux**: Full support with package manager integration
- **macOS**: Development and testing support
- **Windows**: Full support with Windows-specific optimizations
- **Docker**: Containerized deployment support
- **Cloud**: AWS, GCP, Azure support

## Future Enhancements

- Kubernetes deployment support
- Advanced monitoring integration
- Automated backup configuration
- Multi-region deployment
- Load balancer configuration
- Advanced security scanning
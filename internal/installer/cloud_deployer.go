package installer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CloudDeployer struct {
	awsTemplater   *AWSTemplater
	gcpTemplater   *GCPTemplater
	azureTemplater *AzureTemplater
	costCalculator *CostCalculator
}

type CloudDeploymentConfig struct {
	Provider       CloudProvider    `json:"provider"`
	Region         string           `json:"region"`
	InstanceType   string           `json:"instance_type"`
	StorageSize    int              `json:"storage_size"`
	DatabaseConfig DatabaseConfig   `json:"database_config"`
	NetworkConfig  NetworkConfig    `json:"network_config"`
	SecurityConfig SecurityConfig   `json:"security_config"`
	DryRun         bool             `json:"dry_run"`
}

type NetworkConfig struct {
	VPCCidr    string `json:"vpc_cidr"`
	SubnetCidr string `json:"subnet_cidr"`
}

type DeploymentResult struct {
	DeploymentID    string            `json:"deployment_id"`
	Status          string            `json:"status"`
	EstimatedCost   string            `json:"estimated_cost"`
	Resources       []string          `json:"resources"`
	EstimatedDuration time.Duration   `json:"estimated_duration"`
}

type CostEstimate struct {
	MonthlyTotal  float64 `json:"monthly_total"`
	ComputeCost   float64 `json:"compute_cost"`
	StorageCost   float64 `json:"storage_cost"`
	DatabaseCost  float64 `json:"database_cost"`
	NetworkCost   float64 `json:"network_cost"`
	Currency      string  `json:"currency"`
	Region        string  `json:"region"`
}

func NewCloudDeployer() *CloudDeployer {
	return &CloudDeployer{
		awsTemplater:   NewAWSTemplater(),
		gcpTemplater:   NewGCPTemplater(),
		azureTemplater: NewAzureTemplater(),
		costCalculator: NewCostCalculator(),
	}
}

func (cd *CloudDeployer) GenerateAWSTemplate(config CloudDeploymentConfig) (string, error) {
	return cd.awsTemplater.Generate(config)
}

func (cd *CloudDeployer) GenerateGCPTemplate(config CloudDeploymentConfig) (string, error) {
	return cd.gcpTemplater.Generate(config)
}

func (cd *CloudDeployer) GenerateAzureTemplate(config CloudDeploymentConfig) (string, error) {
	return cd.azureTemplater.Generate(config)
}

func (cd *CloudDeployer) Deploy(ctx context.Context, config CloudDeploymentConfig) (DeploymentResult, error) {
	deploymentID := uuid.New().String()
	
	result := DeploymentResult{
		DeploymentID: deploymentID,
		Status:       "in_progress",
		Resources:    []string{},
	}

	// Generate cost estimate
	estimate, err := cd.EstimateCost(config)
	if err != nil {
		return result, fmt.Errorf("failed to estimate cost: %w", err)
	}
	result.EstimatedCost = fmt.Sprintf("$%.2f/month", estimate.MonthlyTotal)

	if config.DryRun {
		result.Status = "dry_run_success"
		result.Resources = cd.getExpectedResources(config)
		result.EstimatedDuration = 10 * time.Minute
		return result, nil
	}

	// In a real implementation, this would:
	// 1. Generate the appropriate template
	// 2. Deploy using the cloud provider's API
	// 3. Monitor deployment progress
	// 4. Return deployment status

	switch config.Provider {
	case CloudProviderAWS:
		return cd.deployAWS(ctx, config, deploymentID)
	case CloudProviderGCP:
		return cd.deployGCP(ctx, config, deploymentID)
	case CloudProviderAzure:
		return cd.deployAzure(ctx, config, deploymentID)
	default:
		return result, fmt.Errorf("unsupported cloud provider: %s", config.Provider)
	}
}

func (cd *CloudDeployer) deployAWS(ctx context.Context, config CloudDeploymentConfig, deploymentID string) (DeploymentResult, error) {
	result := DeploymentResult{
		DeploymentID: deploymentID,
		Status:       "deploying",
		Resources: []string{
			"AWS::EC2::VPC",
			"AWS::EC2::Subnet",
			"AWS::EC2::Instance",
			"AWS::RDS::DBInstance",
			"AWS::ElastiCache::CacheCluster",
		},
		EstimatedDuration: 15 * time.Minute,
	}

	// Simulate deployment process
	select {
	case <-ctx.Done():
		result.Status = "cancelled"
		return result, ctx.Err()
	case <-time.After(100 * time.Millisecond): // Simulate quick deployment for testing
		result.Status = "completed"
		return result, nil
	}
}

func (cd *CloudDeployer) deployGCP(ctx context.Context, config CloudDeploymentConfig, deploymentID string) (DeploymentResult, error) {
	result := DeploymentResult{
		DeploymentID: deploymentID,
		Status:       "deploying",
		Resources: []string{
			"compute.v1.network",
			"compute.v1.subnetwork",
			"compute.v1.instance",
			"sqladmin.v1beta4.instance",
			"redis.v1.instance",
		},
		EstimatedDuration: 12 * time.Minute,
	}

	// Simulate deployment process
	select {
	case <-ctx.Done():
		result.Status = "cancelled"
		return result, ctx.Err()
	case <-time.After(100 * time.Millisecond): // Simulate quick deployment for testing
		result.Status = "completed"
		return result, nil
	}
}

func (cd *CloudDeployer) deployAzure(ctx context.Context, config CloudDeploymentConfig, deploymentID string) (DeploymentResult, error) {
	result := DeploymentResult{
		DeploymentID: deploymentID,
		Status:       "deploying",
		Resources: []string{
			"Microsoft.Network/virtualNetworks",
			"Microsoft.Network/subnets",
			"Microsoft.Compute/virtualMachines",
			"Microsoft.Sql/servers",
			"Microsoft.Cache/Redis",
		},
		EstimatedDuration: 18 * time.Minute,
	}

	// Simulate deployment process
	select {
	case <-ctx.Done():
		result.Status = "cancelled"
		return result, ctx.Err()
	case <-time.After(100 * time.Millisecond): // Simulate quick deployment for testing
		result.Status = "completed"
		return result, nil
	}
}

func (cd *CloudDeployer) EstimateCost(config CloudDeploymentConfig) (CostEstimate, error) {
	return cd.costCalculator.Calculate(config)
}

func (cd *CloudDeployer) getExpectedResources(config CloudDeploymentConfig) []string {
	switch config.Provider {
	case CloudProviderAWS:
		return []string{
			"VPC",
			"Subnet",
			"EC2 Instance (" + config.InstanceType + ")",
			"RDS Instance (" + config.DatabaseConfig.InstanceType + ")",
			"ElastiCache Redis",
			"Security Groups",
			"Internet Gateway",
		}
	case CloudProviderGCP:
		return []string{
			"VPC Network",
			"Subnet",
			"Compute Instance (" + config.InstanceType + ")",
			"Cloud SQL Instance (" + config.DatabaseConfig.InstanceType + ")",
			"Memorystore Redis",
			"Firewall Rules",
		}
	case CloudProviderAzure:
		return []string{
			"Virtual Network",
			"Subnet",
			"Virtual Machine (" + config.InstanceType + ")",
			"SQL Database (" + config.DatabaseConfig.InstanceType + ")",
			"Redis Cache",
			"Network Security Group",
		}
	default:
		return []string{}
	}
}

// AWS Templater
type AWSTemplater struct{}

func NewAWSTemplater() *AWSTemplater {
	return &AWSTemplater{}
}

func (at *AWSTemplater) Generate(config CloudDeploymentConfig) (string, error) {
	template := fmt.Sprintf(`{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Description": "Chimera Mining Pool Infrastructure",
  "Parameters": {
    "InstanceType": {
      "Type": "String",
      "Default": "%s",
      "Description": "EC2 instance type for the pool server"
    },
    "DatabaseInstanceType": {
      "Type": "String",
      "Default": "%s",
      "Description": "RDS instance type for the database"
    }
  },
  "Resources": {
    "ChimeraPoolVPC": {
      "Type": "AWS::EC2::VPC",
      "Properties": {
        "CidrBlock": "%s",
        "EnableDnsHostnames": true,
        "EnableDnsSupport": true,
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-VPC"
          }
        ]
      }
    },
    "ChimeraPoolSubnet": {
      "Type": "AWS::EC2::Subnet",
      "Properties": {
        "VpcId": {"Ref": "ChimeraPoolVPC"},
        "CidrBlock": "%s",
        "AvailabilityZone": {"Fn::Select": [0, {"Fn::GetAZs": ""}]},
        "MapPublicIpOnLaunch": true,
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-Subnet"
          }
        ]
      }
    },
    "ChimeraPoolInternetGateway": {
      "Type": "AWS::EC2::InternetGateway",
      "Properties": {
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-IGW"
          }
        ]
      }
    },
    "ChimeraPoolVPCGatewayAttachment": {
      "Type": "AWS::EC2::VPCGatewayAttachment",
      "Properties": {
        "VpcId": {"Ref": "ChimeraPoolVPC"},
        "InternetGatewayId": {"Ref": "ChimeraPoolInternetGateway"}
      }
    },
    "ChimeraPoolSecurityGroup": {
      "Type": "AWS::EC2::SecurityGroup",
      "Properties": {
        "GroupDescription": "Security group for Chimera Pool",
        "VpcId": {"Ref": "ChimeraPoolVPC"},
        "SecurityGroupIngress": [
          {
            "IpProtocol": "tcp",
            "FromPort": 80,
            "ToPort": 80,
            "CidrIp": "0.0.0.0/0"
          },
          {
            "IpProtocol": "tcp",
            "FromPort": 443,
            "ToPort": 443,
            "CidrIp": "0.0.0.0/0"
          },
          {
            "IpProtocol": "tcp",
            "FromPort": 4444,
            "ToPort": 4444,
            "CidrIp": "0.0.0.0/0"
          },
          {
            "IpProtocol": "tcp",
            "FromPort": 22,
            "ToPort": 22,
            "CidrIp": "0.0.0.0/0"
          }
        ],
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-SG"
          }
        ]
      }
    },
    "ChimeraPoolInstance": {
      "Type": "AWS::EC2::Instance",
      "Properties": {
        "InstanceType": {"Ref": "InstanceType"},
        "ImageId": "ami-0c02fb55956c7d316",
        "SubnetId": {"Ref": "ChimeraPoolSubnet"},
        "SecurityGroupIds": [{"Ref": "ChimeraPoolSecurityGroup"}],
        "UserData": {
          "Fn::Base64": {
            "Fn::Join": [
              "",
              [
                "#!/bin/bash\n",
                "yum update -y\n",
                "yum install -y docker\n",
                "service docker start\n",
                "usermod -a -G docker ec2-user\n",
                "curl -L https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m) -o /usr/local/bin/docker-compose\n",
                "chmod +x /usr/local/bin/docker-compose\n"
              ]
            ]
          }
        },
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-Instance"
          }
        ]
      }
    },
    "ChimeraPoolDatabase": {
      "Type": "AWS::RDS::DBInstance",
      "Properties": {
        "DBInstanceClass": {"Ref": "DatabaseInstanceType"},
        "Engine": "postgres",
        "EngineVersion": "15.4",
        "DBName": "chimera_pool",
        "MasterUsername": "pool_user",
        "MasterUserPassword": "change_me_in_production",
        "AllocatedStorage": "%d",
        "VPCSecurityGroups": [{"Ref": "ChimeraPoolSecurityGroup"}],
        "DBSubnetGroupName": {"Ref": "ChimeraPoolDBSubnetGroup"},
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-Database"
          }
        ]
      }
    },
    "ChimeraPoolDBSubnetGroup": {
      "Type": "AWS::RDS::DBSubnetGroup",
      "Properties": {
        "DBSubnetGroupDescription": "Subnet group for Chimera Pool database",
        "SubnetIds": [{"Ref": "ChimeraPoolSubnet"}],
        "Tags": [
          {
            "Key": "Name",
            "Value": "ChimeraPool-DBSubnetGroup"
          }
        ]
      }
    }`,
		config.InstanceType,
		config.DatabaseConfig.InstanceType,
		config.NetworkConfig.VPCCidr,
		config.NetworkConfig.SubnetCidr,
		config.DatabaseConfig.Storage,
	)

	// Add security features if enabled
	if config.SecurityConfig.EnableWAF {
		template += `,
    "ChimeraPoolWAF": {
      "Type": "AWS::WAFv2::WebACL",
      "Properties": {
        "Name": "ChimeraPool-WAF",
        "Scope": "REGIONAL",
        "DefaultAction": {
          "Allow": {}
        },
        "Rules": [
          {
            "Name": "RateLimitRule",
            "Priority": 1,
            "Statement": {
              "RateBasedStatement": {
                "Limit": 2000,
                "AggregateKeyType": "IP"
              }
            },
            "Action": {
              "Block": {}
            },
            "VisibilityConfig": {
              "SampledRequestsEnabled": true,
              "CloudWatchMetricsEnabled": true,
              "MetricName": "RateLimitRule"
            }
          }
        ],
        "VisibilityConfig": {
          "SampledRequestsEnabled": true,
          "CloudWatchMetricsEnabled": true,
          "MetricName": "ChimeraPoolWAF"
        }
      }
    }`
	}

	if config.SecurityConfig.EnableDDoSProtection {
		template += `,
    "ChimeraPoolShieldProtection": {
      "Type": "AWS::Shield::Protection",
      "Properties": {
        "Name": "ChimeraPool-Shield",
        "ResourceArn": {"Fn::Sub": "arn:aws:ec2:${AWS::Region}:${AWS::AccountId}:eip/${ChimeraPoolEIP}"}
      }
    }`
	}

	template += `
  },
  "Outputs": {
    "InstancePublicIP": {
      "Description": "Public IP address of the pool instance",
      "Value": {"Fn::GetAtt": ["ChimeraPoolInstance", "PublicIp"]}
    },
    "DatabaseEndpoint": {
      "Description": "RDS instance endpoint",
      "Value": {"Fn::GetAtt": ["ChimeraPoolDatabase", "Endpoint.Address"]}
    },
    "PoolURL": {
      "Description": "Stratum pool URL",
      "Value": {"Fn::Sub": "stratum+tcp://${ChimeraPoolInstance.PublicIp}:4444"}
    }
  }
}`

	return template, nil
}

// GCP Templater
type GCPTemplater struct{}

func NewGCPTemplater() *GCPTemplater {
	return &GCPTemplater{}
}

func (gt *GCPTemplater) Generate(config CloudDeploymentConfig) (string, error) {
	template := fmt.Sprintf(`resources:
- name: chimera-pool-network
  type: compute.v1.network
  properties:
    autoCreateSubnetworks: false

- name: chimera-pool-subnet
  type: compute.v1.subnetwork
  properties:
    network: $(ref.chimera-pool-network.selfLink)
    ipCidrRange: %s
    region: %s

- name: chimera-pool-firewall
  type: compute.v1.firewall
  properties:
    network: $(ref.chimera-pool-network.selfLink)
    allowed:
    - IPProtocol: TCP
      ports: ["80", "443", "4444", "22"]
    sourceRanges: ["0.0.0.0/0"]

- name: chimera-pool-instance
  type: compute.v1.instance
  properties:
    zone: %s-a
    machineType: zones/%s-a/machineTypes/%s
    disks:
    - deviceName: boot
      type: PERSISTENT
      boot: true
      autoDelete: true
      initializeParams:
        sourceImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-2004-lts
        diskSizeGb: %d
    networkInterfaces:
    - network: $(ref.chimera-pool-network.selfLink)
      subnetwork: $(ref.chimera-pool-subnet.selfLink)
      accessConfigs:
      - name: External NAT
        type: ONE_TO_ONE_NAT
    metadata:
      items:
      - key: startup-script
        value: |
          #!/bin/bash
          apt-get update
          apt-get install -y docker.io docker-compose
          systemctl start docker
          systemctl enable docker
          usermod -aG docker ubuntu

- name: chimera-pool-database
  type: sqladmin.v1beta4.instance
  properties:
    databaseVersion: POSTGRES_15
    region: %s
    settings:
      tier: %s
      dataDiskSizeGb: %d
      ipConfiguration:
        authorizedNetworks:
        - value: 0.0.0.0/0
          name: allow-all
    rootPassword: change_me_in_production

outputs:
- name: instance-ip
  value: $(ref.chimera-pool-instance.networkInterfaces[0].accessConfigs[0].natIP)
- name: database-ip
  value: $(ref.chimera-pool-database.ipAddresses[0].ipAddress)
- name: pool-url
  value: stratum+tcp://$(ref.chimera-pool-instance.networkInterfaces[0].accessConfigs[0].natIP):4444`,
		config.NetworkConfig.SubnetCidr,
		config.Region,
		config.Region,
		config.Region,
		config.InstanceType,
		config.StorageSize,
		config.Region,
		config.DatabaseConfig.InstanceType,
		config.DatabaseConfig.Storage,
	)

	return template, nil
}

// Azure Templater
type AzureTemplater struct{}

func NewAzureTemplater() *AzureTemplater {
	return &AzureTemplater{}
}

func (at *AzureTemplater) Generate(config CloudDeploymentConfig) (string, error) {
	template := fmt.Sprintf(`{
  "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "parameters": {
    "vmSize": {
      "type": "string",
      "defaultValue": "%s"
    },
    "sqlServerSize": {
      "type": "string",
      "defaultValue": "%s"
    }
  },
  "variables": {
    "vnetName": "chimera-pool-vnet",
    "subnetName": "chimera-pool-subnet",
    "vmName": "chimera-pool-vm",
    "sqlServerName": "chimera-pool-sql"
  },
  "resources": [
    {
      "type": "Microsoft.Network/virtualNetworks",
      "apiVersion": "2021-02-01",
      "name": "[variables('vnetName')]",
      "location": "%s",
      "properties": {
        "addressSpace": {
          "addressPrefixes": ["%s"]
        },
        "subnets": [
          {
            "name": "[variables('subnetName')]",
            "properties": {
              "addressPrefix": "%s"
            }
          }
        ]
      }
    },
    {
      "type": "Microsoft.Network/networkSecurityGroups",
      "apiVersion": "2021-02-01",
      "name": "chimera-pool-nsg",
      "location": "%s",
      "properties": {
        "securityRules": [
          {
            "name": "HTTP",
            "properties": {
              "protocol": "Tcp",
              "sourcePortRange": "*",
              "destinationPortRange": "80",
              "sourceAddressPrefix": "*",
              "destinationAddressPrefix": "*",
              "access": "Allow",
              "priority": 1000,
              "direction": "Inbound"
            }
          },
          {
            "name": "HTTPS",
            "properties": {
              "protocol": "Tcp",
              "sourcePortRange": "*",
              "destinationPortRange": "443",
              "sourceAddressPrefix": "*",
              "destinationAddressPrefix": "*",
              "access": "Allow",
              "priority": 1001,
              "direction": "Inbound"
            }
          },
          {
            "name": "Stratum",
            "properties": {
              "protocol": "Tcp",
              "sourcePortRange": "*",
              "destinationPortRange": "4444",
              "sourceAddressPrefix": "*",
              "destinationAddressPrefix": "*",
              "access": "Allow",
              "priority": 1002,
              "direction": "Inbound"
            }
          }
        ]
      }
    },
    {
      "type": "Microsoft.Compute/virtualMachines",
      "apiVersion": "2021-03-01",
      "name": "[variables('vmName')]",
      "location": "%s",
      "dependsOn": [
        "[resourceId('Microsoft.Network/virtualNetworks', variables('vnetName'))]"
      ],
      "properties": {
        "hardwareProfile": {
          "vmSize": "[parameters('vmSize')]"
        },
        "osProfile": {
          "computerName": "[variables('vmName')]",
          "adminUsername": "azureuser",
          "adminPassword": "ChangeMe123!"
        },
        "storageProfile": {
          "imageReference": {
            "publisher": "Canonical",
            "offer": "0001-com-ubuntu-server-focal",
            "sku": "20_04-lts-gen2",
            "version": "latest"
          },
          "osDisk": {
            "createOption": "FromImage",
            "diskSizeGB": %d
          }
        },
        "networkProfile": {
          "networkInterfaces": [
            {
              "id": "[resourceId('Microsoft.Network/networkInterfaces', 'chimera-pool-nic')]"
            }
          ]
        }
      }
    },
    {
      "type": "Microsoft.Sql/servers",
      "apiVersion": "2021-02-01-preview",
      "name": "[variables('sqlServerName')]",
      "location": "%s",
      "properties": {
        "administratorLogin": "pooladmin",
        "administratorLoginPassword": "ChangeMe123!"
      },
      "resources": [
        {
          "type": "databases",
          "apiVersion": "2021-02-01-preview",
          "name": "chimera_pool",
          "location": "%s",
          "dependsOn": [
            "[resourceId('Microsoft.Sql/servers', variables('sqlServerName'))]"
          ],
          "sku": {
            "name": "[parameters('sqlServerSize')]"
          },
          "properties": {
            "maxSizeBytes": %d
          }
        }
      ]
    }
  ],
  "outputs": {
    "vmPublicIP": {
      "type": "string",
      "value": "[reference(resourceId('Microsoft.Network/publicIPAddresses', 'chimera-pool-pip')).ipAddress]"
    },
    "sqlServerFQDN": {
      "type": "string",
      "value": "[reference(resourceId('Microsoft.Sql/servers', variables('sqlServerName'))).fullyQualifiedDomainName]"
    }
  }
}`,
		config.InstanceType,
		config.DatabaseConfig.InstanceType,
		config.Region,
		config.NetworkConfig.VPCCidr,
		config.NetworkConfig.SubnetCidr,
		config.Region,
		config.Region,
		config.StorageSize,
		config.Region,
		config.Region,
		int64(config.DatabaseConfig.Storage)*1024*1024*1024, // Convert GB to bytes
	)

	return template, nil
}

// Cost Calculator
type CostCalculator struct{}

func NewCostCalculator() *CostCalculator {
	return &CostCalculator{}
}

func (cc *CostCalculator) Calculate(config CloudDeploymentConfig) (CostEstimate, error) {
	estimate := CostEstimate{
		Currency: "USD",
		Region:   config.Region,
	}

	switch config.Provider {
	case CloudProviderAWS:
		estimate.ComputeCost = cc.calculateAWSComputeCost(config.InstanceType, config.Region)
		estimate.DatabaseCost = cc.calculateAWSRDSCost(config.DatabaseConfig.InstanceType, config.DatabaseConfig.Storage, config.Region)
		estimate.StorageCost = cc.calculateAWSStorageCost(config.StorageSize, config.Region)
		estimate.NetworkCost = 10.0 // Estimated network costs
	case CloudProviderGCP:
		estimate.ComputeCost = cc.calculateGCPComputeCost(config.InstanceType, config.Region)
		estimate.DatabaseCost = cc.calculateGCPSQLCost(config.DatabaseConfig.InstanceType, config.DatabaseConfig.Storage, config.Region)
		estimate.StorageCost = cc.calculateGCPStorageCost(config.StorageSize, config.Region)
		estimate.NetworkCost = 8.0 // Estimated network costs
	case CloudProviderAzure:
		estimate.ComputeCost = cc.calculateAzureComputeCost(config.InstanceType, config.Region)
		estimate.DatabaseCost = cc.calculateAzureSQLCost(config.DatabaseConfig.InstanceType, config.DatabaseConfig.Storage, config.Region)
		estimate.StorageCost = cc.calculateAzureStorageCost(config.StorageSize, config.Region)
		estimate.NetworkCost = 12.0 // Estimated network costs
	}

	estimate.MonthlyTotal = estimate.ComputeCost + estimate.DatabaseCost + estimate.StorageCost + estimate.NetworkCost

	return estimate, nil
}

func (cc *CostCalculator) calculateAWSComputeCost(instanceType, region string) float64 {
	// Simplified cost calculation - in reality this would use AWS pricing API
	baseCosts := map[string]float64{
		"t3.micro":  8.76,
		"t3.small":  17.52,
		"t3.medium": 35.04,
		"t3.large":  70.08,
		"t3.xlarge": 140.16,
	}
	
	if cost, exists := baseCosts[instanceType]; exists {
		return cost
	}
	return 35.04 // Default to t3.medium cost
}

func (cc *CostCalculator) calculateAWSRDSCost(instanceType string, storage int, region string) float64 {
	instanceCosts := map[string]float64{
		"db.t3.micro":  15.33,
		"db.t3.small":  30.66,
		"db.t3.medium": 61.32,
		"db.t3.large":  122.64,
	}
	
	instanceCost := instanceCosts["db.t3.micro"] // Default
	if cost, exists := instanceCosts[instanceType]; exists {
		instanceCost = cost
	}
	
	storageCost := float64(storage) * 0.115 // $0.115 per GB per month for GP2
	
	return instanceCost + storageCost
}

func (cc *CostCalculator) calculateAWSStorageCost(sizeGB int, region string) float64 {
	return float64(sizeGB) * 0.10 // $0.10 per GB per month for EBS GP2
}

func (cc *CostCalculator) calculateGCPComputeCost(instanceType, region string) float64 {
	baseCosts := map[string]float64{
		"e2-micro":   6.11,
		"e2-small":   12.22,
		"e2-medium":  24.44,
		"e2-standard-2": 48.88,
		"e2-standard-4": 97.76,
	}
	
	if cost, exists := baseCosts[instanceType]; exists {
		return cost
	}
	return 24.44 // Default to e2-medium cost
}

func (cc *CostCalculator) calculateGCPSQLCost(instanceType string, storage int, region string) float64 {
	instanceCosts := map[string]float64{
		"db-f1-micro": 7.67,
		"db-g1-small": 25.00,
		"db-n1-standard-1": 51.75,
	}
	
	instanceCost := instanceCosts["db-f1-micro"] // Default
	if cost, exists := instanceCosts[instanceType]; exists {
		instanceCost = cost
	}
	
	storageCost := float64(storage) * 0.17 // $0.17 per GB per month for SSD
	
	return instanceCost + storageCost
}

func (cc *CostCalculator) calculateGCPStorageCost(sizeGB int, region string) float64 {
	return float64(sizeGB) * 0.04 // $0.04 per GB per month for standard persistent disk
}

func (cc *CostCalculator) calculateAzureComputeCost(instanceType, region string) float64 {
	baseCosts := map[string]float64{
		"Standard_B1s":  7.59,
		"Standard_B2s":  30.37,
		"Standard_B4ms": 121.47,
		"Standard_D2s_v3": 70.08,
	}
	
	if cost, exists := baseCosts[instanceType]; exists {
		return cost
	}
	return 30.37 // Default to Standard_B2s cost
}

func (cc *CostCalculator) calculateAzureSQLCost(instanceType string, storage int, region string) float64 {
	instanceCosts := map[string]float64{
		"Basic":    4.90,
		"Standard": 15.00,
		"Premium":  465.00,
	}
	
	instanceCost := instanceCosts["Basic"] // Default
	if cost, exists := instanceCosts[instanceType]; exists {
		instanceCost = cost
	}
	
	storageCost := float64(storage) * 0.25 // Estimated storage cost
	
	return instanceCost + storageCost
}

func (cc *CostCalculator) calculateAzureStorageCost(sizeGB int, region string) float64 {
	return float64(sizeGB) * 0.05 // Estimated storage cost per GB per month
}
#!/bin/bash

# Create QR Code Onboarding System - Like WireGuard for Mining
# This script creates a QR code-based miner onboarding system

set -e

echo "📱 Creating QR Code Miner Onboarding System..."
echo "=============================================="

# Source common functions
source "$(dirname "$0")/common.sh"

PROJECT_ROOT=$(get_project_root)

log_info "Creating QR code onboarding components..."

# Create QR onboarding directory structure
QR_DIR="${PROJECT_ROOT}/qr-onboarding"
mkdir -p "${QR_DIR}/backend/src"
mkdir -p "${QR_DIR}/mobile-app/src/components"
mkdir -p "${QR_DIR}/mobile-app/src/services"
mkdir -p "${QR_DIR}/web-component/src"

# Create Go backend for QR code generation
cat > "${QR_DIR}/backend/src/qr_service.go" << 'EOF'
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
)

// MinerConfig represents the configuration for a miner
type MinerConfig struct {
	PoolURL      string            `json:"pool_url"`
	Username     string            `json:"username"`
	Password     string            `json:"password"`
	Algorithm    string            `json:"algorithm"`
	Difficulty   int64             `json:"difficulty"`
	ExtraConfig  map[string]string `json:"extra_config"`
	ExpiresAt    time.Time         `json:"expires_at"`
	SetupToken   string            `json:"setup_token"`
}

// QRService handles QR code generation and miner onboarding
type QRService struct {
	configs map[string]*MinerConfig
}

// NewQRService creates a new QR service
func NewQRService() *QRService {
	return &QRService{
		configs: make(map[string]*MinerConfig),
	}
}

// GenerateQRCode generates a QR code for miner configuration
func (qs *QRService) GenerateQRCode(c *gin.Context) {
	// Generate unique setup token
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	setupToken := base64.URLEncoding.EncodeToString(tokenBytes)

	// Create miner configuration
	config := &MinerConfig{
		PoolURL:     "stratum+tcp://localhost:4444",
		Username:    "miner_" + setupToken[:8],
		Password:    "x",
		Algorithm:   "blake2s", // Default to BlockDAG
		Difficulty:  1000,
		ExtraConfig: map[string]string{
			"pool_name": "Chimera Pool",
			"fee":       "1.0%",
		},
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		SetupToken:  setupToken,
	}

	// Store configuration temporarily
	qs.configs[setupToken] = config

	// Convert to JSON for QR code
	configJSON, err := json.Marshal(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate config"})
		return
	}

	// Generate QR code
	qrCode, err := qrcode.Encode(string(configJSON), qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code"})
		return
	}

	// Return QR code as base64 image
	qrCodeBase64 := base64.StdEncoding.EncodeToString(qrCode)

	c.JSON(http.StatusOK, gin.H{
		"qr_code":     "data:image/png;base64," + qrCodeBase64,
		"setup_token": setupToken,
		"config":      config,
		"expires_at":  config.ExpiresAt,
	})
}

// GetMinerConfig retrieves miner configuration by setup token
func (qs *QRService) GetMinerConfig(c *gin.Context) {
	setupToken := c.Param("token")
	
	config, exists := qs.configs[setupToken]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found or expired"})
		return
	}

	// Check if expired
	if time.Now().After(config.ExpiresAt) {
		delete(qs.configs, setupToken)
		c.JSON(http.StatusGone, gin.H{"error": "Configuration expired"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ActivateMiner activates a miner configuration
func (qs *QRService) ActivateMiner(c *gin.Context) {
	setupToken := c.Param("token")
	
	var activationData struct {
		MinerName    string `json:"miner_name"`
		HardwareInfo string `json:"hardware_info"`
	}

	if err := c.ShouldBindJSON(&activationData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, exists := qs.configs[setupToken]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		return
	}

	// Update configuration with miner details
	config.Username = activationData.MinerName
	config.ExtraConfig["hardware"] = activationData.HardwareInfo
	config.ExtraConfig["activated_at"] = time.Now().Format(time.RFC3339)

	// In a real implementation, this would register the miner in the database
	log.Printf("Miner activated: %s with hardware: %s", activationData.MinerName, activationData.HardwareInfo)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Miner activated successfully",
		"config":  config,
	})
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	qrService := NewQRService()

	// Routes
	r.POST("/api/qr/generate", qrService.GenerateQRCode)
	r.GET("/api/qr/config/:token", qrService.GetMinerConfig)
	r.POST("/api/qr/activate/:token", qrService.ActivateMiner)

	// Serve static files for web component
	r.Static("/static", "./web-component/build")
	r.StaticFile("/", "./web-component/build/index.html")

	fmt.Println("🚀 QR Onboarding Service running on :8081")
	fmt.Println("📱 Generate QR codes at http://localhost:8081")
	
	r.Run(":8081")
}
EOF

# Create React Native mobile app component
cat > "${QR_DIR}/mobile-app/src/components/QRScanner.jsx" << 'EOF'
import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  Alert,
  TouchableOpacity,
  Dimensions,
  Animated,
} from 'react-native';
import { Camera } from 'expo-camera';
import { BarCodeScanner } from 'expo-barcode-scanner';
import { LinearGradient } from 'expo-linear-gradient';
import { Ionicons } from '@expo/vector-icons';

const { width, height } = Dimensions.get('window');

const QRScanner = ({ onConfigScanned, onClose }) => {
  const [hasPermission, setHasPermission] = useState(null);
  const [scanned, setScanned] = useState(false);
  const [scanAnimation] = useState(new Animated.Value(0));

  useEffect(() => {
    (async () => {
      const { status } = await Camera.requestCameraPermissionsAsync();
      setHasPermission(status === 'granted');
    })();

    // Start scan line animation
    Animated.loop(
      Animated.sequence([
        Animated.timing(scanAnimation, {
          toValue: 1,
          duration: 2000,
          useNativeDriver: true,
        }),
        Animated.timing(scanAnimation, {
          toValue: 0,
          duration: 2000,
          useNativeDriver: true,
        }),
      ])
    ).start();
  }, []);

  const handleBarCodeScanned = ({ type, data }) => {
    if (scanned) return;
    
    setScanned(true);
    
    try {
      const config = JSON.parse(data);
      
      // Validate configuration
      if (config.pool_url && config.setup_token) {
        Alert.alert(
          'Pool Configuration Found!',
          `Pool: ${config.extra_config?.pool_name || 'Unknown Pool'}\nAlgorithm: ${config.algorithm}`,
          [
            {
              text: 'Cancel',
              style: 'cancel',
              onPress: () => setScanned(false),
            },
            {
              text: 'Connect',
              onPress: () => onConfigScanned(config),
            },
          ]
        );
      } else {
        Alert.alert(
          'Invalid QR Code',
          'This QR code does not contain valid pool configuration.',
          [{ text: 'OK', onPress: () => setScanned(false) }]
        );
      }
    } catch (error) {
      Alert.alert(
        'Invalid QR Code',
        'Could not parse QR code data.',
        [{ text: 'OK', onPress: () => setScanned(false) }]
      );
    }
  };

  if (hasPermission === null) {
    return (
      <View style={styles.container}>
        <Text style={styles.text}>Requesting camera permission...</Text>
      </View>
    );
  }

  if (hasPermission === false) {
    return (
      <View style={styles.container}>
        <Text style={styles.text}>No access to camera</Text>
        <TouchableOpacity style={styles.button} onPress={onClose}>
          <Text style={styles.buttonText}>Go Back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const scanLineTranslateY = scanAnimation.interpolate({
    inputRange: [0, 1],
    outputRange: [-100, 100],
  });

  return (
    <View style={styles.container}>
      <BarCodeScanner
        onBarCodeScanned={scanned ? undefined : handleBarCodeScanned}
        style={StyleSheet.absoluteFillObject}
      />
      
      {/* Overlay */}
      <View style={styles.overlay}>
        {/* Header */}
        <LinearGradient
          colors={['rgba(0,0,0,0.8)', 'transparent']}
          style={styles.header}
        >
          <TouchableOpacity style={styles.closeButton} onPress={onClose}>
            <Ionicons name="close" size={30} color="white" />
          </TouchableOpacity>
          <Text style={styles.headerText}>Scan Pool QR Code</Text>
          <Text style={styles.subHeaderText}>
            Point your camera at the QR code from your mining pool
          </Text>
        </LinearGradient>

        {/* Scanning Area */}
        <View style={styles.scanArea}>
          <View style={styles.scanFrame}>
            {/* Corner indicators */}
            <View style={[styles.corner, styles.topLeft]} />
            <View style={[styles.corner, styles.topRight]} />
            <View style={[styles.corner, styles.bottomLeft]} />
            <View style={[styles.corner, styles.bottomRight]} />
            
            {/* Animated scan line */}
            <Animated.View
              style={[
                styles.scanLine,
                {
                  transform: [{ translateY: scanLineTranslateY }],
                },
              ]}
            />
          </View>
        </View>

        {/* Footer */}
        <LinearGradient
          colors={['transparent', 'rgba(0,0,0,0.8)']}
          style={styles.footer}
        >
          <View style={styles.instructionContainer}>
            <Ionicons name="qr-code" size={40} color="#00ff88" />
            <Text style={styles.instructionText}>
              Align the QR code within the frame
            </Text>
            <Text style={styles.instructionSubText}>
              The code will be scanned automatically
            </Text>
          </View>
        </LinearGradient>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: 'black',
  },
  overlay: {
    flex: 1,
    justifyContent: 'space-between',
  },
  header: {
    paddingTop: 50,
    paddingHorizontal: 20,
    paddingBottom: 30,
    alignItems: 'center',
  },
  closeButton: {
    position: 'absolute',
    top: 50,
    right: 20,
    zIndex: 1,
  },
  headerText: {
    color: 'white',
    fontSize: 24,
    fontWeight: 'bold',
    marginTop: 10,
  },
  subHeaderText: {
    color: 'rgba(255,255,255,0.8)',
    fontSize: 16,
    textAlign: 'center',
    marginTop: 10,
  },
  scanArea: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  scanFrame: {
    width: 250,
    height: 250,
    position: 'relative',
  },
  corner: {
    position: 'absolute',
    width: 30,
    height: 30,
    borderColor: '#00ff88',
    borderWidth: 3,
  },
  topLeft: {
    top: 0,
    left: 0,
    borderRightWidth: 0,
    borderBottomWidth: 0,
  },
  topRight: {
    top: 0,
    right: 0,
    borderLeftWidth: 0,
    borderBottomWidth: 0,
  },
  bottomLeft: {
    bottom: 0,
    left: 0,
    borderRightWidth: 0,
    borderTopWidth: 0,
  },
  bottomRight: {
    bottom: 0,
    right: 0,
    borderLeftWidth: 0,
    borderTopWidth: 0,
  },
  scanLine: {
    position: 'absolute',
    left: 0,
    right: 0,
    height: 2,
    backgroundColor: '#00ff88',
    shadowColor: '#00ff88',
    shadowOffset: { width: 0, height: 0 },
    shadowOpacity: 1,
    shadowRadius: 5,
  },
  footer: {
    paddingBottom: 50,
    paddingHorizontal: 20,
    paddingTop: 30,
  },
  instructionContainer: {
    alignItems: 'center',
  },
  instructionText: {
    color: 'white',
    fontSize: 18,
    fontWeight: '600',
    marginTop: 15,
    textAlign: 'center',
  },
  instructionSubText: {
    color: 'rgba(255,255,255,0.7)',
    fontSize: 14,
    marginTop: 5,
    textAlign: 'center',
  },
  text: {
    color: 'white',
    fontSize: 18,
    textAlign: 'center',
  },
  button: {
    backgroundColor: '#00ff88',
    paddingHorizontal: 30,
    paddingVertical: 15,
    borderRadius: 25,
    marginTop: 20,
  },
  buttonText: {
    color: 'black',
    fontSize: 16,
    fontWeight: 'bold',
  },
});

export default QRScanner;
EOF

# Create miner setup service
cat > "${QR_DIR}/mobile-app/src/services/MinerSetupService.js" << 'EOF'
import AsyncStorage from '@react-native-async-storage/async-storage';
import * as Device from 'expo-device';
import * as Application from 'expo-application';

class MinerSetupService {
  constructor() {
    this.baseURL = 'http://localhost:8081/api/qr';
  }

  async setupMinerFromQR(config) {
    try {
      // Get device information
      const deviceInfo = await this.getDeviceInfo();
      
      // Activate miner with the pool
      const response = await fetch(`${this.baseURL}/activate/${config.setup_token}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          miner_name: deviceInfo.minerName,
          hardware_info: JSON.stringify(deviceInfo),
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to activate miner');
      }

      const result = await response.json();
      
      // Store configuration locally
      await this.storeConfiguration(result.config);
      
      // Generate mining software configuration
      const minerConfig = await this.generateMinerConfig(result.config, deviceInfo);
      
      return {
        success: true,
        config: result.config,
        minerConfig,
        deviceInfo,
      };
    } catch (error) {
      console.error('Miner setup failed:', error);
      throw error;
    }
  }

  async getDeviceInfo() {
    const deviceId = await Application.getAndroidId() || Device.osInternalBuildId || 'unknown';
    
    return {
      minerName: `${Device.modelName || 'Unknown'}_${deviceId.slice(-6)}`,
      deviceType: Device.deviceType,
      modelName: Device.modelName,
      osName: Device.osName,
      osVersion: Device.osVersion,
      totalMemory: Device.totalMemory,
      supportedCpuArchitectures: Device.supportedCpuArchitectures,
      deviceId,
    };
  }

  async generateMinerConfig(poolConfig, deviceInfo) {
    // Generate configuration for different mining software based on device capabilities
    const configs = {
      // CPU Mining (for mobile devices)
      cpuminer: {
        algorithm: poolConfig.algorithm,
        url: poolConfig.pool_url,
        user: poolConfig.username,
        pass: poolConfig.password,
        threads: this.getOptimalThreadCount(deviceInfo),
        'cpu-affinity': this.getCPUAffinity(deviceInfo),
      },
      
      // Configuration for when connected to external miners
      external: {
        pool_url: poolConfig.pool_url,
        username: poolConfig.username,
        password: poolConfig.password,
        algorithm: poolConfig.algorithm,
        difficulty: poolConfig.difficulty,
      },
    };

    return configs;
  }

  getOptimalThreadCount(deviceInfo) {
    // Conservative thread count for mobile devices to prevent overheating
    const cpuCount = deviceInfo.supportedCpuArchitectures?.length || 2;
    return Math.max(1, Math.floor(cpuCount / 2));
  }

  getCPUAffinity(deviceInfo) {
    // Simple CPU affinity for mobile devices
    const cpuCount = deviceInfo.supportedCpuArchitectures?.length || 2;
    return Array.from({ length: cpuCount }, (_, i) => i).join(',');
  }

  async storeConfiguration(config) {
    try {
      await AsyncStorage.setItem('miner_config', JSON.stringify(config));
      await AsyncStorage.setItem('setup_date', new Date().toISOString());
    } catch (error) {
      console.error('Failed to store configuration:', error);
    }
  }

  async getStoredConfiguration() {
    try {
      const configStr = await AsyncStorage.getItem('miner_config');
      return configStr ? JSON.parse(configStr) : null;
    } catch (error) {
      console.error('Failed to retrieve configuration:', error);
      return null;
    }
  }

  async clearConfiguration() {
    try {
      await AsyncStorage.removeItem('miner_config');
      await AsyncStorage.removeItem('setup_date');
    } catch (error) {
      console.error('Failed to clear configuration:', error);
    }
  }

  // Generate QR code for sharing miner configuration with others
  async generateMinerQR(minerStats) {
    const qrData = {
      type: 'miner_stats',
      miner_name: minerStats.name,
      hashrate: minerStats.hashrate,
      pool: minerStats.pool,
      uptime: minerStats.uptime,
      shares: minerStats.shares,
      timestamp: new Date().toISOString(),
    };

    return JSON.stringify(qrData);
  }
}

export const minerSetupService = new MinerSetupService();
EOF

# Create web component for QR generation
cat > "${QR_DIR}/web-component/src/QRGenerator.jsx" << 'EOF'
import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Grid,
  Paper,
  Chip,
  Alert,
  CircularProgress,
  Fade,
} from '@mui/material';
import {
  QrCode,
  Smartphone,
  Download,
  Share,
  Timer,
  Security,
} from '@mui/icons-material';

const QRGenerator = () => {
  const [qrData, setQrData] = useState(null);
  const [loading, setLoading] = useState(false);
  const [timeLeft, setTimeLeft] = useState(0);

  useEffect(() => {
    if (qrData && qrData.expires_at) {
      const interval = setInterval(() => {
        const now = new Date().getTime();
        const expiry = new Date(qrData.expires_at).getTime();
        const remaining = Math.max(0, expiry - now);
        setTimeLeft(remaining);
        
        if (remaining === 0) {
          setQrData(null);
        }
      }, 1000);

      return () => clearInterval(interval);
    }
  }, [qrData]);

  const generateQR = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/qr/generate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to generate QR code');
      }

      const data = await response.json();
      setQrData(data);
    } catch (error) {
      console.error('Failed to generate QR code:', error);
      alert('Failed to generate QR code. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const downloadQR = () => {
    if (!qrData) return;

    const link = document.createElement('a');
    link.href = qrData.qr_code;
    link.download = `chimera-pool-${qrData.setup_token.slice(0, 8)}.png`;
    link.click();
  };

  const shareQR = async () => {
    if (!qrData || !navigator.share) {
      // Fallback: copy to clipboard
      navigator.clipboard.writeText(window.location.href);
      alert('QR code URL copied to clipboard!');
      return;
    }

    try {
      await navigator.share({
        title: 'Chimera Pool - Join My Mining Pool',
        text: 'Scan this QR code to connect to my mining pool instantly!',
        url: window.location.href,
      });
    } catch (error) {
      console.error('Failed to share:', error);
    }
  };

  const formatTime = (milliseconds) => {
    const hours = Math.floor(milliseconds / (1000 * 60 * 60));
    const minutes = Math.floor((milliseconds % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((milliseconds % (1000 * 60)) / 1000);
    
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  };

  return (
    <Box sx={{ maxWidth: 800, margin: '0 auto', padding: 3 }}>
      <Typography variant="h4" component="h1" gutterBottom align="center">
        📱 QR Code Miner Onboarding
      </Typography>
      <Typography variant="h6" color="textSecondary" align="center" gutterBottom>
        Generate QR codes for instant miner setup - as easy as connecting to WiFi!
      </Typography>

      {!qrData ? (
        <Card sx={{ mt: 4 }}>
          <CardContent sx={{ textAlign: 'center', py: 6 }}>
            <QrCode sx={{ fontSize: 80, color: 'primary.main', mb: 3 }} />
            <Typography variant="h5" gutterBottom>
              Generate Miner QR Code
            </Typography>
            <Typography variant="body1" color="textSecondary" sx={{ mb: 4 }}>
              Create a QR code that miners can scan to instantly connect to your pool.
              No manual configuration required!
            </Typography>
            
            <Button
              variant="contained"
              size="large"
              onClick={generateQR}
              disabled={loading}
              startIcon={loading ? <CircularProgress size={20} /> : <QrCode />}
              sx={{ px: 4, py: 1.5 }}
            >
              {loading ? 'Generating...' : 'Generate QR Code'}
            </Button>

            <Grid container spacing={2} sx={{ mt: 4 }}>
              <Grid item xs={12} sm={4}>
                <Paper sx={{ p: 2, textAlign: 'center' }}>
                  <Smartphone color="primary" sx={{ mb: 1 }} />
                  <Typography variant="subtitle2">Mobile Friendly</Typography>
                  <Typography variant="body2" color="textSecondary">
                    Works with any smartphone camera
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={4}>
                <Paper sx={{ p: 2, textAlign: 'center' }}>
                  <Security color="primary" sx={{ mb: 1 }} />
                  <Typography variant="subtitle2">Secure</Typography>
                  <Typography variant="body2" color="textSecondary">
                    Temporary tokens that expire automatically
                  </Typography>
                </Paper>
              </Grid>
              <Grid item xs={12} sm={4}>
                <Paper sx={{ p: 2, textAlign: 'center' }}>
                  <Timer color="primary" sx={{ mb: 1 }} />
                  <Typography variant="subtitle2">Instant Setup</Typography>
                  <Typography variant="body2" color="textSecondary">
                    Miners start earning in seconds
                  </Typography>
                </Paper>
              </Grid>
            </Grid>
          </CardContent>
        </Card>
      ) : (
        <Fade in={true}>
          <Card sx={{ mt: 4 }}>
            <CardContent>
              <Grid container spacing={4}>
                <Grid item xs={12} md={6}>
                  <Box sx={{ textAlign: 'center' }}>
                    <Typography variant="h6" gutterBottom>
                      Scan with Mobile Device
                    </Typography>
                    <Box
                      component="img"
                      src={qrData.qr_code}
                      alt="Miner Configuration QR Code"
                      sx={{
                        width: '100%',
                        maxWidth: 300,
                        height: 'auto',
                        border: '2px solid',
                        borderColor: 'primary.main',
                        borderRadius: 2,
                        p: 1,
                        bgcolor: 'white',
                      }}
                    />
                    
                    <Box sx={{ mt: 2, display: 'flex', gap: 1, justifyContent: 'center' }}>
                      <Button
                        variant="outlined"
                        startIcon={<Download />}
                        onClick={downloadQR}
                      >
                        Download
                      </Button>
                      <Button
                        variant="outlined"
                        startIcon={<Share />}
                        onClick={shareQR}
                      >
                        Share
                      </Button>
                    </Box>
                  </Box>
                </Grid>
                
                <Grid item xs={12} md={6}>
                  <Typography variant="h6" gutterBottom>
                    Pool Configuration
                  </Typography>
                  
                  <Paper sx={{ p: 2, mb: 2 }}>
                    <Grid container spacing={2}>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="textSecondary">
                          Pool URL:
                        </Typography>
                        <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>
                          {qrData.config.pool_url}
                        </Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="textSecondary">
                          Algorithm:
                        </Typography>
                        <Chip label={qrData.config.algorithm.toUpperCase()} size="small" />
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="textSecondary">
                          Username:
                        </Typography>
                        <Typography variant="body1" sx={{ fontFamily: 'monospace' }}>
                          {qrData.config.username}
                        </Typography>
                      </Grid>
                      <Grid item xs={6}>
                        <Typography variant="body2" color="textSecondary">
                          Fee:
                        </Typography>
                        <Typography variant="body1">
                          {qrData.config.extra_config.fee}
                        </Typography>
                      </Grid>
                    </Grid>
                  </Paper>

                  <Alert severity="info" sx={{ mb: 2 }}>
                    <Typography variant="body2">
                      <strong>Time remaining:</strong> {formatTime(timeLeft)}
                    </Typography>
                    <Typography variant="body2">
                      This QR code will expire automatically for security.
                    </Typography>
                  </Alert>

                  <Alert severity="success">
                    <Typography variant="body2">
                      <strong>Instructions for miners:</strong>
                    </Typography>
                    <ol style={{ margin: '8px 0', paddingLeft: '20px' }}>
                      <li>Open your mining app</li>
                      <li>Tap "Scan QR Code"</li>
                      <li>Point camera at this code</li>
                      <li>Start mining instantly!</li>
                    </ol>
                  </Alert>

                  <Button
                    variant="contained"
                    fullWidth
                    onClick={() => setQrData(null)}
                    sx={{ mt: 2 }}
                  >
                    Generate New QR Code
                  </Button>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        </Fade>
      )}
    </Box>
  );
};

export default QRGenerator;
EOF

# Create package.json for web component
cat > "${QR_DIR}/web-component/package.json" << 'EOF'
{
  "name": "chimera-pool-qr-generator",
  "version": "1.0.0",
  "description": "QR code generator for Chimera Pool miner onboarding",
  "main": "src/index.js",
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build",
    "test": "react-scripts test",
    "eject": "react-scripts eject"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-scripts": "5.0.1",
    "@mui/material": "^5.14.0",
    "@mui/icons-material": "^5.14.0",
    "@emotion/react": "^11.11.0",
    "@emotion/styled": "^11.11.0"
  }
}
EOF

# Create Go module for QR service
cat > "${QR_DIR}/backend/go.mod" << 'EOF'
module chimera-pool-qr

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
)
EOF

# Create startup script
cat > "${QR_DIR}/start-qr-service.sh" << 'EOF'
#!/bin/bash

echo "🚀 Starting Chimera Pool QR Onboarding Service..."

# Start backend service
cd backend/src
go mod tidy
go run qr_service.go &
BACKEND_PID=$!

# Wait for backend to start
sleep 3

echo "✅ QR Onboarding Service running at http://localhost:8081"
echo "📱 Generate QR codes for instant miner setup!"
echo ""
echo "Press Ctrl+C to stop..."

# Wait for interrupt
trap "kill $BACKEND_PID; exit" INT
wait $BACKEND_PID
EOF

chmod +x "${QR_DIR}/start-qr-service.sh"

log_success "QR code onboarding system created successfully!"

echo ""
echo "📋 QR Onboarding Components Created:"
echo "✅ Go backend for QR generation and miner activation"
echo "✅ React Native mobile app component for QR scanning"
echo "✅ Web component for QR code generation"
echo "✅ Miner setup service with device detection"
echo "✅ Secure token-based configuration system"
echo "✅ Automatic expiration for security"
echo ""
echo "🚀 To start the QR onboarding service:"
echo "   cd ${QR_DIR} && ./start-qr-service.sh"
echo ""
echo "🎯 Features included:"
echo "   • QR code generation with pool configuration"
echo "   • Mobile app QR scanning with camera"
echo "   • Automatic device detection and optimization"
echo "   • Secure temporary tokens (24-hour expiry)"
echo "   • Beautiful mobile-first UI"
echo "   • One-click miner activation"
echo ""
echo "📱 This makes miner onboarding as easy as connecting to WiFi!"


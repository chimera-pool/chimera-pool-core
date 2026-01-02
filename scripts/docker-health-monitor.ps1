<#
.SYNOPSIS
    Docker Health Monitor for Chimeria Pool
.DESCRIPTION
    Monitors Docker containers and automatically restarts unhealthy ones.
    Runs as a background service to ensure pool uptime.
.NOTES
    Following ISP: Single responsibility - monitor and recover Docker containers
#>

param(
    [int]$CheckIntervalSeconds = 30,
    [int]$MaxRestartAttempts = 3,
    [string]$LogFile = "docker-health-monitor.log"
)

$ErrorActionPreference = "Continue"
$ScriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$LogPath = Join-Path $ScriptPath $LogFile

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logEntry = "[$timestamp] [$Level] $Message"
    Add-Content -Path $LogPath -Value $logEntry
    Write-Host $logEntry
}

function Test-DockerEngine {
    try {
        $result = docker info 2>&1
        return $LASTEXITCODE -eq 0
    } catch {
        return $false
    }
}

function Start-DockerDesktop {
    Write-Log "Docker engine not responding, attempting to start Docker Desktop..." "WARN"
    try {
        Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe" -WindowStyle Hidden
        Start-Sleep -Seconds 30
        
        # Wait for Docker to be ready (max 2 minutes)
        $maxWait = 120
        $waited = 0
        while (-not (Test-DockerEngine) -and $waited -lt $maxWait) {
            Start-Sleep -Seconds 5
            $waited += 5
            Write-Log "Waiting for Docker engine... ($waited/$maxWait seconds)"
        }
        
        if (Test-DockerEngine) {
            Write-Log "Docker Desktop started successfully" "INFO"
            return $true
        } else {
            Write-Log "Failed to start Docker Desktop after $maxWait seconds" "ERROR"
            return $false
        }
    } catch {
        Write-Log "Error starting Docker Desktop: $_" "ERROR"
        return $false
    }
}

function Get-ContainerHealth {
    param([string]$ContainerName)
    
    try {
        $status = docker inspect --format='{{.State.Status}}' $ContainerName 2>&1
        $health = docker inspect --format='{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}' $ContainerName 2>&1
        
        return @{
            Name = $ContainerName
            Status = $status
            Health = $health
            IsHealthy = ($status -eq "running") -and ($health -in @("healthy", "no-healthcheck"))
        }
    } catch {
        return @{
            Name = $ContainerName
            Status = "error"
            Health = "unknown"
            IsHealthy = $false
        }
    }
}

function Restart-Container {
    param([string]$ContainerName, [int]$Attempt = 1)
    
    Write-Log "Restarting container $ContainerName (attempt $Attempt/$MaxRestartAttempts)" "WARN"
    
    try {
        docker restart $ContainerName 2>&1 | Out-Null
        Start-Sleep -Seconds 10
        
        $health = Get-ContainerHealth -ContainerName $ContainerName
        if ($health.IsHealthy) {
            Write-Log "Container $ContainerName restarted successfully" "INFO"
            return $true
        } else {
            Write-Log "Container $ContainerName still unhealthy after restart" "WARN"
            return $false
        }
    } catch {
        Write-Log "Error restarting container $ContainerName: $_" "ERROR"
        return $false
    }
}

function Start-AllContainers {
    Write-Log "Starting all Chimeria Pool containers..." "INFO"
    
    try {
        $composePath = Join-Path (Split-Path -Parent $ScriptPath) "deployments\docker\docker-compose.yml"
        if (Test-Path $composePath) {
            docker-compose -f $composePath up -d 2>&1 | Out-Null
            Write-Log "All containers started via docker-compose" "INFO"
            return $true
        } else {
            Write-Log "docker-compose.yml not found at $composePath" "ERROR"
            return $false
        }
    } catch {
        Write-Log "Error starting containers: $_" "ERROR"
        return $false
    }
}

# Critical containers to monitor (in priority order)
$CriticalContainers = @(
    "docker-postgres-1",
    "docker-redis-1",
    "docker-litecoind-1",
    "docker-chimera-pool-api-1",
    "docker-chimera-pool-stratum-1",
    "docker-chimera-pool-web-1",
    "docker-nginx-1",
    "docker-grafana-1",
    "docker-prometheus-1"
)

# Track restart attempts per container
$RestartAttempts = @{}
foreach ($container in $CriticalContainers) {
    $RestartAttempts[$container] = 0
}

Write-Log "=== Chimeria Pool Docker Health Monitor Started ===" "INFO"
Write-Log "Monitoring $($CriticalContainers.Count) containers every $CheckIntervalSeconds seconds" "INFO"

# Main monitoring loop
while ($true) {
    try {
        # Check if Docker engine is running
        if (-not (Test-DockerEngine)) {
            Write-Log "Docker engine not responding!" "ERROR"
            if (Start-DockerDesktop) {
                Start-AllContainers
            }
            Start-Sleep -Seconds $CheckIntervalSeconds
            continue
        }
        
        # Check each container
        foreach ($containerName in $CriticalContainers) {
            $health = Get-ContainerHealth -ContainerName $containerName
            
            if (-not $health.IsHealthy) {
                Write-Log "Container $containerName is unhealthy (Status: $($health.Status), Health: $($health.Health))" "WARN"
                
                if ($RestartAttempts[$containerName] -lt $MaxRestartAttempts) {
                    $RestartAttempts[$containerName]++
                    $success = Restart-Container -ContainerName $containerName -Attempt $RestartAttempts[$containerName]
                    
                    if ($success) {
                        $RestartAttempts[$containerName] = 0
                    }
                } else {
                    Write-Log "Container $containerName has exceeded max restart attempts ($MaxRestartAttempts)" "ERROR"
                    # Reset after 5 minutes to allow retry
                    if ((Get-Date).Minute % 5 -eq 0) {
                        $RestartAttempts[$containerName] = 0
                    }
                }
            } else {
                # Reset restart counter on healthy container
                if ($RestartAttempts[$containerName] -gt 0) {
                    Write-Log "Container $containerName is now healthy, resetting restart counter" "INFO"
                    $RestartAttempts[$containerName] = 0
                }
            }
        }
        
    } catch {
        Write-Log "Error in monitoring loop: $_" "ERROR"
    }
    
    Start-Sleep -Seconds $CheckIntervalSeconds
}

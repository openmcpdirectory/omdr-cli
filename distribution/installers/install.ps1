# OMDR CLI installer script for Windows

$ErrorActionPreference = "Stop"

$REPO = "openmcpdirectory/omdr-cli"
$BINARY_NAME = "omdr.exe"

function Get-LatestRelease {
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
    return $response.tag_name
}

function Detect-Platform {
    $arch = [System.Environment]::Is64BitOperatingSystem
    if ($arch) {
        return "x86_64"
    } else {
        Write-Error "32-bit Windows is not supported"
        exit 1
    }
}

function Main {
    param([string]$Version)
    
    $archName = Detect-Platform
    
    if (-not $Version) {
        $Version = Get-LatestRelease
    }
    
    $Version = $Version.TrimStart('v')
    
    $filename = "omdr_Windows_${archName}.zip"
    $url = "https://github.com/$REPO/releases/download/v${Version}/${filename}"
    
    Write-Host "Downloading OMDR CLI v${Version} for Windows ${archName}..."
    
    $tmpDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    $zipPath = Join-Path $tmpDir $filename
    
    try {
        Invoke-WebRequest -Uri $url -OutFile $zipPath
        Expand-Archive -Path $zipPath -DestinationPath $tmpDir
        
        $installDir = "$env:LOCALAPPDATA\Programs\omdr"
        New-Item -ItemType Directory -Force -Path $installDir | Out-Null
        
        Copy-Item -Path (Join-Path $tmpDir $BINARY_NAME) -Destination $installDir -Force
        
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$installDir*") {
            [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
            Write-Host ""
            Write-Host "Added $installDir to PATH"
            Write-Host "Please restart your terminal for PATH changes to take effect"
        }
        
        Write-Host ""
        Write-Host "OMDR CLI v${Version} installed successfully!"
        Write-Host "Run 'omdr --help' to get started (restart terminal first)"
    }
    finally {
        Remove-Item -Recurse -Force $tmpDir
    }
}

Main -Version $args[0]

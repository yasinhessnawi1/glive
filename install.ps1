# GLive Installer for Windows
# Usage: irm https://raw.githubusercontent.com/yasinhessnawi1/glive/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "yasinhessnawi1/glive"
$BinaryName = "glive.exe"
$InstallDir = "$env:LOCALAPPDATA\glive\bin"

Write-Host "Installing GLive..." -ForegroundColor Green

# Create install directory
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# Detect architecture
$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

Write-Host "Detected: windows/$Arch"

# Get latest release
Write-Host "Fetching latest release..."
try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
    $LatestVersion = $Release.tag_name
    Write-Host "Latest version: $LatestVersion"

    $VersionNoV = $LatestVersion.TrimStart('v')
    $DownloadUrl = "https://github.com/$Repo/releases/download/$LatestVersion/glive_${VersionNoV}_windows_${Arch}.zip"

    Write-Host "Downloading from: $DownloadUrl"

    $TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    $ZipPath = Join-Path $TempDir "glive.zip"

    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing

    # Extract
    Expand-Archive -Path $ZipPath -DestinationPath $TempDir -Force

    # Move binary
    $ExtractedBinary = Get-ChildItem -Path $TempDir -Recurse -Filter "glive.exe" | Select-Object -First 1
    if ($ExtractedBinary) {
        Move-Item -Path $ExtractedBinary.FullName -Destination (Join-Path $InstallDir $BinaryName) -Force
    } else {
        throw "Binary not found in archive"
    }

    # Cleanup
    Remove-Item -Path $TempDir -Recurse -Force
}
catch {
    Write-Host "No releases found or download failed. Building from source..." -ForegroundColor Yellow

    # Check if Go is installed
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "Go is not installed. Please install Go from https://go.dev/dl/" -ForegroundColor Red
        exit 1
    }

    # Clone and build
    $TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    Push-Location $TempDir

    git clone --depth 1 "https://github.com/$Repo.git"
    Set-Location "glive\cmd\glive"
    go build -o $BinaryName

    Move-Item -Path $BinaryName -Destination (Join-Path $InstallDir $BinaryName) -Force

    Pop-Location
    Remove-Item -Path $TempDir -Recurse -Force
}

# Add to PATH if not already there
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
$PathUpdated = $false
if ($CurrentPath -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to PATH..." -ForegroundColor Yellow
    try {
        [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
        $env:Path = "$env:Path;$InstallDir"
        $PathUpdated = $true
        Write-Host "PATH updated successfully!" -ForegroundColor Green
    }
    catch {
        Write-Host "Failed to update PATH automatically." -ForegroundColor Red
        $PathUpdated = $false
    }
}

# Verify installation
if (Test-Path (Join-Path $InstallDir $BinaryName)) {
    Write-Host ""
    Write-Host "GLive installed successfully!" -ForegroundColor Green
    Write-Host "Location: $InstallDir\$BinaryName"
    Write-Host ""

    if ($PathUpdated) {
        Write-Host "PATH has been updated. Please restart your terminal for changes to take effect." -ForegroundColor Cyan
    } else {
        # Check if it's actually in PATH now
        $VerifyPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($VerifyPath -notlike "*$InstallDir*") {
            Write-Host "To use 'glive' from anywhere, add this to your PATH:" -ForegroundColor Yellow
            Write-Host ""
            Write-Host "  $InstallDir" -ForegroundColor White
            Write-Host ""
            Write-Host "Run this command to add it now:" -ForegroundColor Yellow
            Write-Host '  [Environment]::SetEnvironmentVariable("Path", $env:Path + ";' + $InstallDir + '", "User")' -ForegroundColor White
            Write-Host ""
        }
    }
    Write-Host ""
    Write-Host "Usage:" -ForegroundColor Cyan
    Write-Host "  glive <github-url>        Clone and set up a GitHub project"
    Write-Host "  glive user/repo           Short format (e.g., glive facebook/react)"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Cyan
    Write-Host "  glive https://github.com/user/repo"
    Write-Host "  glive user/repo"
    Write-Host "  glive --help"
    Write-Host ""
} else {
    Write-Host "Installation may have failed. Please check manually." -ForegroundColor Red
}

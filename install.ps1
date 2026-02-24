# Markdown in the Middle - Install Script for Windows
# Downloads the latest release binary.
#
# Usage (PowerShell):
#   iwr -useb https://raw.githubusercontent.com/rickcrawford/markdowninthemiddle/main/install.ps1 | iex
#
# Environment variables:
#   INSTALL_DIR  - Directory to install to (default: %USERPROFILE%\bin)
#   VERSION      - Specific version to install (default: latest)

$ErrorActionPreference = "Stop"

$Repo = "rickcrawford/markdowninthemiddle"
$BinaryName = "markdowninthemiddle"

function Get-LatestVersion {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    return $release.tag_name
}

function Get-Arch {
    if ([Environment]::Is64BitOperatingSystem) {
        if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
            return "arm64"
        }
        return "amd64"
    }
    Write-Error "Unsupported architecture"
    exit 1
}

$Arch = Get-Arch

$Version = if ($env:VERSION) { $env:VERSION } else { Get-LatestVersion }
if (-not $Version) {
    Write-Error "Could not determine latest version"
    exit 1
}

$VersionNum = $Version -replace "^v", ""
$Filename = "${BinaryName}_${VersionNum}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$Filename"

$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:USERPROFILE "bin" }

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Write-Host "Downloading $BinaryName $Version for windows/$Arch..."
    $ZipPath = Join-Path $TmpDir $Filename
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipPath -UseBasicParsing

    Write-Host "Extracting..."
    Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

    Write-Host "Installing to $InstallDir\$BinaryName.exe..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $TmpDir "$BinaryName.exe") -Destination (Join-Path $InstallDir "$BinaryName.exe") -Force

    # Add to PATH if not already there
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        Write-Host "Adding $InstallDir to user PATH..."
        [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
        $env:PATH = "$env:PATH;$InstallDir"
    }

    Write-Host ""
    Write-Host "$BinaryName $Version installed to $InstallDir\$BinaryName.exe"
    Write-Host ""
    Write-Host "Run '$BinaryName --help' to get started."
}
finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

param(
    [string]$Repo = "ZHANGSHUNLIN/TTL-CLI",
    [string]$DownloadUrl = $env:TTL_DOWNLOAD_URL
)

$ErrorActionPreference = "Stop"
$BinaryName = "ttl"
$InstallDir = Join-Path $env:USERPROFILE ".ttl"

function Write-Host-Color {
    param([string]$Message, [string]$Color = "White")
    Write-Host "[$BinaryName] " -NoNewline
    Write-Host $Message -ForegroundColor $Color
}

function Info  { param($msg) Write-Host-Color $msg "Cyan" }
function Success { param($msg) Write-Host-Color $msg "Green" }
function Warn   { param($msg) Write-Host-Color $msg "Yellow" }
function Error  { param($msg) Write-Host-Color $msg "Red"; exit 1 }

function Get-OsArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64"   { return "amd64" }
        "ARM64"   { return "arm64" }
        default    { Error "Unsupported CPU architecture: $arch" }
    }
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        return $response.tag_name
    } catch {
        Error "Failed to get latest version, please check repo: $Repo"
    }
}

function Download-Binary {
    param([string]$Url, [string]$OutputPath)

    Info "Downloading $Url..."

    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing
    } catch {
        Error "Download failed: $($_.Exception.Message)`nPossible causes:`n  - Network issue`n  - GitHub is blocked`n  - Try: `$env:TTL_DOWNLOAD_URL = 'your-mirror-url'; ./install.ps1"
    }
}

function Add-ToPath {
    param([string]$Dir)

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -split ';' -contains $Dir) {
        return
    }

    Warn "$Dir is not in PATH, adding..."

    try {
        $newPath = if ($currentPath) { "$currentPath;$Dir" } else { $Dir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Success "Added $Dir to user PATH"
        Warn "Please reopen terminal for changes to take effect"
    } catch {
        Warn "Could not add to PATH automatically, please add manually"
        Warn ""
        Warn "  1. Open System Properties -> Environment Variables"
        Warn "  2. Find Path in User Variables and click Edit"
        Warn "  3. Add: $Dir"
        Warn ""
    }
}

$arch = Get-OsArch

if ($DownloadUrl) {
    $filename = [System.IO.Path]::GetFileName($DownloadUrl)
    $downloadUrl = $DownloadUrl
    Info "Using custom download URL"
    Info "Download file: $filename"
} else {
    $version = Get-LatestVersion
    $filename = "ttl-cli-${version}-windows-${arch}.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$version/$filename"
    Info "OS: Windows / Arch: $arch"
    Info "Latest version: $version"
}
Info "Install target: $InstallDir\$BinaryName.exe"

if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Info "Created install directory: $InstallDir"
}

$tempPath = Join-Path $env:TEMP $filename
$extractDir = Join-Path $env:TEMP "ttl-cli-extract"
Download-Binary -Url $downloadUrl -OutputPath $tempPath

Info "Extracting..."
Remove-Item -Path $extractDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
Expand-Archive -Path $tempPath -DestinationPath $extractDir -Force

$extractedExe = Get-ChildItem -Path $extractDir -Filter "*.exe" | Select-Object -First 1

if ($extractedExe) {
    $destPath = Join-Path $InstallDir "$BinaryName.exe"
    Move-Item -Path $extractedExe.FullName -Destination $destPath -Force

    Remove-Item -Path $tempPath -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $extractDir -Recurse -Force -ErrorAction SilentlyContinue
} else {
    Error "Extraction failed, no executable found"
}

Success "Installation successful!"
Success "Installed to: $destPath"

Add-ToPath -Dir $InstallDir

Info ""
Info "Installation complete! You can now run:"
Info "  ttl --help"
Info ""
Warn "If command not found, please reopen terminal"

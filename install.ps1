# =============================================================================
# ttl Windows 安装脚本
# 用法：irm https://your-host/install.ps1 | iex
# 从 GitHub Release 下载预编译二进制
# =============================================================================

param(
    [string]$Repo = "your-org/great-tool-go"  # ← 替换为实际仓库：owner/repo
)

$ErrorActionPreference = "Stop"
$BinaryName = "ttl"
$InstallDir = Join-Path $env:USERPROFILE "bin"

# ── 输出函数 ──────────────────────────────────────────────────────────────────
function Write-Host-Color {
    param([string]$Message, [string]$Color = "White")
    Write-Host "[$BinaryName] " -NoNewline
    Write-Host $Message -ForegroundColor $Color
}

function Info  { Write-Host-Color $_[1] "Cyan" }
function Success { Write-Host-Color $_[1] "Green" }
function Warn   { Write-Host-Color $_[1] "Yellow" }
function Error  { Write-Host-Color $_[1] "Red"; exit 1 }

# ── 环境检测 ──────────────────────────────────────────────────────────────────
function Get-OsArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64"   { return "amd64" }
        "ARM64"   { return "arm64" }
        default    { Error "不支持的 CPU 架构：$arch" }
    }
}

# ── GitHub API 获取最新版本 ───────────────────────────────────────────────────
function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        return $response.tag_name
    } catch {
        Error "无法获取最新版本信息，请检查仓库地址：$Repo"
    }
}

# ── 下载二进制文件 ───────────────────────────────────────────────────────────
function Download-Binary {
    param([string]$Url, [string]$OutputPath)

    Info "正在下载 $Url..."

    try {
        # 使用 TLS 1.2
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $Url -OutFile $OutputPath -UseBasicParsing
    } catch {
        Error "下载失败：$($_.Exception.Message)"
    }
}

# ── 添加到 PATH ───────────────────────────────────────────────────────────────
function Add-ToPath {
    param([string]$Dir)

    # 检查是否已在 PATH 中
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -split ';' -contains $Dir) {
        return
    }

    Warn "$Dir 不在 PATH 中，正在添加..."

    try {
        $newPath = if ($currentPath) { "$currentPath;$Dir" } else { $Dir }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Success "已将 $Dir 添加到用户 PATH"
        Warn "请重新打开终端以使更改生效"
    } catch {
        Warn "无法自动添加到 PATH，请手动添加"
        Warn ""
        Warn "  1. 打开 系统属性 -> 环境变量"
        Warn "  2. 在 用户变量 中找到 Path，点击编辑"
        Warn "  3. 添加: $Dir"
        Warn ""
    }
}

# ── 主流程 ────────────────────────────────────────────────────────────────────
$arch = Get-OsArch
$version = Get-LatestVersion
$filename = "$BinaryName-windows-$arch.exe"
$downloadUrl = "https://github.com/$Repo/releases/download/$version/$filename"

Info "操作系统：Windows / 架构：$arch"
Info "最新版本：$version"
Info "安装目标：$InstallDir\$BinaryName.exe"

# 创建安装目录
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Info "创建安装目录：$InstallDir"
}

# 下载到临时文件
$tempPath = Join-Path $env:TEMP $filename
Download-Binary -Url $downloadUrl -OutputPath $tempPath

# 安装
$destPath = Join-Path $InstallDir "$BinaryName.exe"
Move-Item -Path $tempPath -Destination $destPath -Force

Success "安装成功！"
Success "安装位置：$destPath"

# 添加到 PATH
Add-ToPath -Dir $InstallDir

# 验证
Info ""
Info "安装完成！现在可以运行："
Info "  ttl --help"
Info ""
Warn "如果提示找不到命令，请重新打开终端"

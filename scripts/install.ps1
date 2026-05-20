param(
    [switch]$SkipLaunch,
    [switch]$NoBootstrap
)

$ErrorActionPreference = "Stop"

$AppName = "weazlchat"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$AppRoot = if ($env:WEAZLCHAT_HOME) { $env:WEAZLCHAT_HOME } else { Join-Path $env:APPDATA $AppName }
$BinDir = Join-Path $AppRoot "bin"
$BinPath = Join-Path $BinDir "$AppName.exe"
$SetupPath = Join-Path $BinDir "$AppName-setup.exe"
$GoCache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $AppRoot "go-cache" }
$GoModCache = if ($env:GOMODCACHE) { $env:GOMODCACHE } else { Join-Path $AppRoot "go-mod-cache" }
$MSYSRoot = if ($env:MSYS2_ROOT) { $env:MSYS2_ROOT } else { "C:\msys64" }
$UCRTBin = Join-Path $MSYSRoot "ucrt64\bin"
$MSYSBash = Join-Path $MSYSRoot "usr\bin\bash.exe"

function Get-RequiredGoVersion {
    $goLine = Get-Content (Join-Path $RepoRoot "go.mod") | Where-Object { $_ -match "^go\s+" } | Select-Object -First 1
    if (-not $goLine) {
        throw "Could not find required Go version in go.mod."
    }
    return ($goLine -replace "^go\s+", "").Split(".")[0..1] -join "."
}

function Get-GoVersion {
    $versionText = & go version 2>$null
    if ($LASTEXITCODE -ne 0 -or -not $versionText) {
        return $null
    }
    if ($versionText -match "go(\d+\.\d+)") {
        return [version]$Matches[1]
    }
    return $null
}

function Test-GoVersion {
    param([string]$Required)
    $current = Get-GoVersion
    return $current -and ($current -ge [version]$Required)
}

function Add-PathSegment {
    param([string]$PathSegment)
    if (-not (Test-Path $PathSegment)) {
        return
    }
    $segments = $env:Path -split ";" | Where-Object { $_ }
    if ($segments -notcontains $PathSegment) {
        $env:Path = "$PathSegment;$env:Path"
    }

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $userSegments = $userPath -split ";" | Where-Object { $_ }
    if ($userSegments -notcontains $PathSegment) {
        $newUserPath = if ($userPath) { "$userPath;$PathSegment" } else { $PathSegment }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Host "Added $PathSegment to your user PATH."
    }
}

function Install-WithWinget {
    param(
        [string]$PackageId,
        [string]$Name
    )
    if ($NoBootstrap) {
        throw "$Name is required, but automatic dependency install is disabled."
    }
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw "$Name is required and winget was not found. Install $Name manually, then rerun scripts\install.ps1."
    }
    Write-Host "Installing $Name with winget..."
    winget install --id $PackageId --exact --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "winget failed to install $Name."
    }
}

function Ensure-Go {
    $required = Get-RequiredGoVersion
    if (Test-GoVersion $required) {
        return
    }
    Install-WithWinget -PackageId "GoLang.Go" -Name "Go $required or newer"
    Add-PathSegment "C:\Program Files\Go\bin"
    if (-not (Test-GoVersion $required)) {
        throw "Go $required or newer is still not available on PATH after install."
    }
}

function Ensure-MSYS2 {
    if (-not (Test-Path $MSYSBash)) {
        Install-WithWinget -PackageId "MSYS2.MSYS2" -Name "MSYS2"
    }
    if (-not (Test-Path $MSYSBash)) {
        throw "MSYS2 bash was not found at $MSYSBash. Set MSYS2_ROOT if MSYS2 is installed elsewhere."
    }

    Write-Host "Installing MSYS2 UCRT64 compiler toolchain..."
    & $MSYSBash -lc "pacman -Sy --noconfirm --needed mingw-w64-ucrt-x86_64-gcc"
    if ($LASTEXITCODE -ne 0) {
        throw "pacman failed to install the UCRT64 gcc toolchain."
    }
    Add-PathSegment $UCRTBin
}

New-Item -ItemType Directory -Force -Path $BinDir, $GoCache, $GoModCache | Out-Null

Ensure-Go
Ensure-MSYS2

$env:CGO_ENABLED = "1"
$env:GOCACHE = $GoCache
$env:GOMODCACHE = $GoModCache
Add-PathSegment $BinDir
Add-PathSegment $UCRTBin

Write-Host "Building $AppName..."
Push-Location $RepoRoot
try {
    go build -buildvcs=false -o $BinPath ./cmd/weazlchat
    if ($LASTEXITCODE -ne 0) {
        throw "failed to build $AppName"
    }
    go build -buildvcs=false -o $SetupPath ./cmd/weazlchat-setup
    if ($LASTEXITCODE -ne 0) {
        throw "failed to build $AppName-setup"
    }
}
finally {
    Pop-Location
}

Write-Host "Installed $AppName to $BinPath"
Write-Host "Config, history, and workspace vaults live under $AppRoot"
Write-Host ""
Write-Host "Configuring provider and optional tools..."
& $SetupPath
if ($LASTEXITCODE -ne 0) {
    throw "$AppName setup failed."
}

if ($SkipLaunch -or $env:WEAZLCHAT_SKIP_LAUNCH -eq "1") {
    Write-Host "Skipping first launch."
} else {
    Write-Host ""
    Write-Host "Launching $AppName..."
    & $BinPath
}

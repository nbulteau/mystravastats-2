param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BackDir = Join-Path $RootDir "back-kotlin"
$NativeBinary = Join-Path $BackDir "build\native\nativeCompile\mystravastats-kotlin.exe"
$OutputBinaryName = if ($env:OUTPUT_BINARY_NAME) { $env:OUTPUT_BINARY_NAME } else { "mystravastats-kotlin-windows.exe" }
$OutputBinary = Join-Path $RootDir $OutputBinaryName
$GradleWrapper = Join-Path $BackDir "gradlew.bat"
$StravaCacheDir = Join-Path $RootDir "strava-cache"
$StravaFilePath = Join-Path $StravaCacheDir ".strava"
$EnvFilePath = Join-Path $RootDir ".env"
$NativeImageOptionsValue = if ($env:NATIVE_IMAGE_OPTIONS) { $env:NATIVE_IMAGE_OPTIONS } else { "--parallelism=4 -J-Xms1g -J-Xmx6g" }
$GradleWorkersMax = if ($env:GRADLE_WORKERS_MAX) { $env:GRADLE_WORKERS_MAX } else { "2" }
$GradleOptsValue = "-Dorg.gradle.vfs.watch=false -Dorg.gradle.workers.max=$GradleWorkersMax"
$GradleUserHome = if ($env:GRADLE_USER_HOME_OVERRIDE) { $env:GRADLE_USER_HOME_OVERRIDE } else { Join-Path $RootDir ".gradle-home-windows" }
$SkipFrontBuild = if ($env:SKIP_FRONT_BUILD) { $env:SKIP_FRONT_BUILD } else { "0" }

$StartTime = Get-Date
Write-Output "[START] Starting build process..."
Write-Output "Native image options: $NativeImageOptionsValue"
Write-Output "Gradle workers max: $GradleWorkersMax"
Write-Output "Gradle user home: $GradleUserHome"
if ($SkipFrontBuild -eq "1") {
    Write-Output "[INFO] Front build: disabled (SKIP_FRONT_BUILD=1)"
} else {
    Write-Output "[INFO] Front build: enabled (front-vue -> public\)"
}

if ($env:OS -ne "Windows_NT") {
    throw "This script targets Windows and must be run on Windows."
}

if (-not (Test-Path $GradleWrapper)) {
    throw "back-kotlin\gradlew.bat not found."
}

if ($SkipFrontBuild -ne "1") {
    $dockerCommand = Get-Command "docker" -ErrorAction SilentlyContinue
    if (-not $dockerCommand) {
        throw "Docker is required to build front-vue in this script. (Or set SKIP_FRONT_BUILD=1 if public\ is already ready.)"
    }

    Write-Output "[FRONT] Building front-vue project..."
    if ($Verbose) {
        & docker run --rm -v "${RootDir}:/app" -w /app/front-vue node:latest sh -c "npm install -g npm@11.6.2 && npm install && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build"
    } else {
        & docker run --rm -v "${RootDir}:/app" -w /app/front-vue node:latest sh -c "npm install -g npm@11.6.2 >/dev/null 2>&1 && npm install >/dev/null 2>&1 && VITE_CJS_TRACE=false NODE_OPTIONS='--no-deprecation' npm run build >/dev/null 2>&1" *> $null
    }
    if ($LASTEXITCODE -ne 0) {
        throw "front-vue build failed."
    }

    $FrontDist = Join-Path $RootDir "front-vue\dist"
    if (-not (Test-Path $FrontDist)) {
        throw "front-vue build failed: dist\ not found."
    }

    # Kotlin backend serves static files from file:public/
    Write-Output "[FRONT] Copying UI build to public\..."
    $PublicDir = Join-Path $RootDir "public"
    if (Test-Path $PublicDir) {
        Remove-Item -Path $PublicDir -Recurse -Force
    }
    New-Item -Path $PublicDir -ItemType Directory | Out-Null
    Copy-Item -Path (Join-Path $FrontDist "*") -Destination $PublicDir -Recurse -Force
} else {
    Write-Output "[SKIP] Skipping front-vue build and copy because SKIP_FRONT_BUILD=1."
}

$nativeImageCommand = Get-Command "native-image" -ErrorAction SilentlyContinue
if (-not $nativeImageCommand) {
    Write-Output "[INFO] 'native-image' was not found in PATH."
    Write-Output "[INFO] Gradle will try to auto-provision a local GraalVM toolchain (Java 25)."
}

Write-Output "[BUILD] Building Windows binary..."

$previousGradleOpts = $env:GRADLE_OPTS
$previousNativeImageOptions = $env:NATIVE_IMAGE_OPTIONS
$previousGradleUserHome = $env:GRADLE_USER_HOME

$env:GRADLE_OPTS = $GradleOptsValue
$env:NATIVE_IMAGE_OPTIONS = $NativeImageOptionsValue
$env:GRADLE_USER_HOME = $GradleUserHome

try {
    if (-not (Test-Path $GradleUserHome)) {
        New-Item -Path $GradleUserHome -ItemType Directory -Force | Out-Null
    }

    Push-Location $BackDir
    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        if ($Verbose) {
            & $GradleWrapper --no-daemon clean nativeCompile
        } else {
            & $GradleWrapper --no-daemon clean nativeCompile *> $null
        }
    } finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed. Re-run with -Verbose for details."
    }
} finally {
    Pop-Location
    $env:GRADLE_OPTS = $previousGradleOpts
    $env:NATIVE_IMAGE_OPTIONS = $previousNativeImageOptions
    $env:GRADLE_USER_HOME = $previousGradleUserHome
}

if (-not (Test-Path $NativeBinary)) {
    throw "Build failed: binary not found at $NativeBinary"
}

Copy-Item -Path $NativeBinary -Destination $OutputBinary -Force
Write-Output "[OK] Windows binary ready: .\$OutputBinaryName"

if (-not (Test-Path $StravaCacheDir)) {
    New-Item -Path $StravaCacheDir -ItemType Directory | Out-Null
    Write-Output "[INFO] Created strava-cache directory."
}

$FamousClimbSource = Join-Path $BackDir "famous-climb"
if (Test-Path $FamousClimbSource) {
    Copy-Item -Path $FamousClimbSource -Destination $StravaCacheDir -Recurse -Force
}

if (-not (Test-Path $StravaFilePath)) {
    @"
clientId=
clientSecret=
useCache=false
"@ | Set-Content -Path $StravaFilePath -Encoding utf8
    Write-Output "[INFO] Any registered Strava user can obtain an access_token by first creating an application at https://www.strava.com/settings/api."
    Write-Output "[TODO] Please add your Strava API credentials to strava-cache/.strava file."
}

if (-not (Test-Path $EnvFilePath)) {
    New-Item -Path $EnvFilePath -ItemType File | Out-Null
    Write-Output "[INFO] Created '.env' file."
}

$existingEnv = Get-Content -Path $EnvFilePath -ErrorAction SilentlyContinue
if (-not ($existingEnv -match '^STRAVA_CACHE_PATH=')) {
    Add-Content -Path $EnvFilePath -Value "STRAVA_CACHE_PATH=$StravaCacheDir"
}

$Elapsed = [int]((Get-Date) - $StartTime).TotalSeconds
Write-Output "[DONE] Build process completed in $Elapsed seconds."
Write-Output "[INFO] Run with: .\$OutputBinaryName"

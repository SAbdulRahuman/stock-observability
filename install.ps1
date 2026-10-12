# Stock Observability native installer for Windows (x64), per user: no admin, no Docker.
#
#   irm https://raw.githubusercontent.com/SAbdulRahuman/stock-observability/main/install.ps1 | iex
#   & ([scriptblock]::Create((irm https://raw.githubusercontent.com/SAbdulRahuman/stock-observability/main/install.ps1))) -Version v0.1.1
#   & ([scriptblock]::Create((irm https://raw.githubusercontent.com/SAbdulRahuman/stock-observability/main/install.ps1))) -Uninstall [-Purge]
#
# Connect your Zerodha Kite account in Settings > Data Sources afterwards, or pass -KiteApiKey /
# KITE_API_KEY and KITE_API_SECRET for unattended installs. Services run as Scheduled Tasks at logon of the current user.
param(
    [string]$Version = 'latest',
    [string]$KiteApiKey = $env:KITE_API_KEY,
    [string]$Listen = '',
    [int]$Port = 0,
    [string]$Archive = '',
    [switch]$NoStart,
    [switch]$NoBrowser,
    [switch]$Uninstall,
    [switch]$Purge
)

$Repo = if ($env:STOCKOBS_REPO) { $env:STOCKOBS_REPO } else { 'SAbdulRahuman/stock-observability' }
$AppDir = if ($env:STOCKOBS_APP_DIR) { $env:STOCKOBS_APP_DIR } else { Join-Path $env:LOCALAPPDATA 'StockObservability' }
$ConfDir = Join-Path $AppDir 'config'
$DataDir = Join-Path $AppDir 'data'
$LogDir = Join-Path $AppDir 'logs'
$Services = @('prometheus', 'alertmanager', 'grafana', 'exporter', 'openbb', 'home', 'portal')
$TaskPrefix = 'StockObservability-'

function Write-Step([string]$Message) { Write-Host "==> $Message" -ForegroundColor Cyan }
function Write-Warn([string]$Message) { Write-Host "warning: $Message" -ForegroundColor Yellow }

# Native commands do not throw on failure.
function Invoke-Native([string]$Exe, [string[]]$Arguments) {
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$(Split-Path -Leaf $Exe) failed with exit code $LASTEXITCODE" }
}

function Write-Text([string]$Path, [string]$Text) {
    [IO.File]::WriteAllText($Path, $Text)
}

function Get-File([string]$Url, [string]$OutFile) {
    try {
        Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $OutFile
    } catch {
        throw "download failed: $Url ($($_.Exception.Message))"
    }
}

function Assert-Hash([string]$Path, [string]$Sha256) {
    if (-not $Sha256 -or (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash -ne $Sha256) {
        throw "checksum mismatch: $(Split-Path -Leaf $Path)"
    }
}

# Extracts a zip and drops a single top-level directory, like tar --strip-components=1.
function Expand-Zip([string]$Zip, [string]$Dest) {
    $tmp = "$Dest.extract"
    Remove-Item -Recurse -Force -LiteralPath $tmp, $Dest -ErrorAction SilentlyContinue
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [IO.Compression.ZipFile]::ExtractToDirectory($Zip, $tmp)
    $children = @(Get-ChildItem -LiteralPath $tmp)
    if ($children.Count -eq 1 -and $children[0].PSIsContainer) {
        Move-Item -LiteralPath $children[0].FullName -Destination $Dest
        Remove-Item -Recurse -Force -LiteralPath $tmp
    } else {
        Move-Item -LiteralPath $tmp -Destination $Dest
    }
}

function Assert-Preflight {
    if (-not [Environment]::Is64BitOperatingSystem) { throw 'only 64-bit Windows is supported' }
    if ($PSVersionTable.PSVersion.Major -lt 5) { throw 'PowerShell 5.1 or later is required' }
    New-Item -ItemType Directory -Force -Path $AppDir | Out-Null
    $drive = Get-PSDrive -Name (Split-Path -Qualifier $AppDir).TrimEnd(':')
    if ($drive.Free -lt 3GB) { throw "at least 3 GB free space is needed on $($drive.Name):" }
}

function Get-Settings {
    $path = Join-Path $ConfDir 'settings.json'
    if (Test-Path -LiteralPath $path) { return Get-Content -Raw -LiteralPath $path | ConvertFrom-Json }
    return $null
}

function Resolve-Settings {
    $saved = Get-Settings
    if (-not $S.Listen) { $S.Listen = if ($saved) { $saved.Listen } else { '127.0.0.1' } }
    if (-not $S.Port) { $S.Port = if ($saved) { [int]$saved.Port } else { 8080 } }
    if ($S.Listen -notmatch '^[0-9a-fA-F.:]+$') { throw "invalid -Listen address: $($S.Listen)" }

    # Credentials are optional: without them, configure Kite in Settings > Data Sources.
    $credFile = Join-Path $ConfDir 'kite_credentials.json'
    $oldSecret = Join-Path $ConfDir 'kite_api_secret'
    $S.KiteSecret = $env:KITE_API_SECRET
    # The v0.1.0 installer kept the key in settings.json and the secret in kite_api_secret.
    if (-not $S.KiteApiKey -and -not $S.KiteSecret -and -not (Test-Path -LiteralPath $credFile) -and $saved -and $saved.KiteApiKey -and (Test-Path -LiteralPath $oldSecret)) {
        Write-Step "Moving Kite credentials to $credFile"
        $S.KiteApiKey = $saved.KiteApiKey
        $S.KiteSecret = (Get-Content -Raw -LiteralPath $oldSecret).Trim()
    }
    if ($S.KiteApiKey -or $S.KiteSecret) {
        if ($S.KiteApiKey -notmatch '^[A-Za-z0-9]{6,128}$') { throw 'invalid Kite API key (-KiteApiKey or KITE_API_KEY): 6-128 letters or digits' }
        if ($S.KiteSecret -notmatch '^[A-Za-z0-9]{6,128}$') { throw 'KITE_API_SECRET must be set with the API key: 6-128 letters or digits' }
    }
    $S.KiteConfigured = [bool]$S.KiteApiKey -or ((Test-Path -LiteralPath $credFile) -and (Get-Content -Raw -LiteralPath $credFile) -match '"api_key"\s*:\s*"[A-Za-z0-9]')
}

function Get-Release([string]$Tmp) {
    $asset = 'stock-observability_windows_amd64.zip'
    $zip = Join-Path $Tmp $asset
    if ($S.Archive) {
        Write-Step "Using local archive $($S.Archive)"
        Copy-Item -LiteralPath $S.Archive -Destination $zip
    } else {
        if ($S.Version -notmatch '^[A-Za-z0-9._-]+$') { throw "invalid -Version: $($S.Version)" }
        $base = if ($S.Version -eq 'latest') { "https://github.com/$Repo/releases/latest/download" } else { "https://github.com/$Repo/releases/download/$($S.Version)" }
        Write-Step "Downloading $Repo release ($($S.Version))"
        $sums = Join-Path $Tmp 'checksums.txt'
        Get-File "$base/checksums.txt" $sums
        Get-File "$base/$asset" $zip
        $line = Get-Content -LiteralPath $sums | Where-Object { ($_ -split '\s+')[1] -eq $asset } | Select-Object -First 1
        Assert-Hash $zip (($line -split '\s+')[0])
    }
    $rel = Join-Path $Tmp 'stock-observability'
    Expand-Zip $zip $rel
    $manifest = @{}
    foreach ($l in Get-Content -LiteralPath (Join-Path $rel 'manifest.env')) {
        if ($l -match '^([A-Z0-9_]+)=(.+)$') { $manifest[$Matches[1]] = $Matches[2] }
    }
    return [pscustomobject]@{
        Dir      = $rel
        Version  = (Get-Content -Raw -LiteralPath (Join-Path $rel 'VERSION')).Trim()
        Manifest = $manifest
    }
}

# Downloads the pinned upstream zip into $AppDir\vendor\NAME-VERSION once.
function Install-Vendor($Release, [string]$Name, [string]$Tmp) {
    $key = $Name.ToUpper()
    $m = $Release.Manifest
    $dir = Join-Path $AppDir "vendor\$Name-$($m["${key}_VERSION"])"
    if (-not (Test-Path -LiteralPath (Join-Path $dir '.complete'))) {
        Write-Step "Downloading $Name $($m["${key}_VERSION"])"
        $zip = Join-Path $Tmp "$Name.zip"
        Get-File $m["${key}_WINDOWS_URL"] $zip
        Assert-Hash $zip $m["${key}_WINDOWS_SHA256"]
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dir) | Out-Null
        Expand-Zip $zip $dir
        Write-Text (Join-Path $dir '.complete') ''
    }
    return $dir
}

function Install-OpenBB($Release, [string]$Uv) {
    $venv = Join-Path $AppDir 'openbb'
    $stamp = Join-Path $venv '.stockobs-version'
    if ((Test-Path -LiteralPath $stamp) -and (Get-Content -Raw -LiteralPath $stamp).Trim() -eq $Release.Version) { return }
    $py = $Release.Manifest['OPENBB_PYTHON_VERSION']
    Write-Step "Installing OpenBB (Python $py); this takes a few minutes"
    $env:UV_PYTHON_INSTALL_DIR = Join-Path $AppDir 'python'
    $env:UV_CACHE_DIR = Join-Path $AppDir 'cache\uv'
    $env:UV_NO_CONFIG = '1'
    Invoke-Native $Uv @('python', 'install', '--quiet', $py)
    Remove-Item -Recurse -Force -LiteralPath $venv -ErrorAction SilentlyContinue
    Invoke-Native $Uv @('venv', '--quiet', '--python', $py, $venv)
    $wheelDir = Join-Path $Release.Dir 'openbb\wheels'
    $wheels = foreach ($w in Get-ChildItem -LiteralPath $wheelDir -Filter *.whl) {
        if ($w.Name -like 'openbb_core-*') { "$($w.FullName)[pandas]" } else { $w.FullName }
    }
    $python = Join-Path $venv 'Scripts\python.exe'
    Invoke-Native $Uv (@('pip', 'install', '--quiet', '--python', $python, '--find-links', $wheelDir) + $wheels +
        "openbb-yfinance==$($Release.Manifest['OPENBB_YFINANCE_VERSION'])")
    $env:OPENBB_AUTO_BUILD = 'false'
    Push-Location $venv
    try { Invoke-Native $python @('-c', 'import openbb; openbb.build()') | Out-Null } finally { Pop-Location }
    Write-Text $stamp $Release.Version
}

# Forward slashes: these paths are written into YAML and INI files.
function Format-Template([string]$Src, [string]$Dst, [string]$Password = '') {
    $text = [IO.File]::ReadAllText($Src)
    $text = $text.Replace('@CONFIG_DIR@', ($ConfDir -replace '\\', '/')).Replace('@DATA_DIR@', ($DataDir -replace '\\', '/'))
    $text = $text.Replace('@PORTAL_PORT@', [string]$S.Port).Replace('@GRAFANA_ADMIN_PASSWORD@', $Password)
    Write-Text $Dst $text
}

# User-editable files are never overwritten: a changed default is written next to them as .new.
function Copy-UserFile([string]$Src, [string]$Dst) {
    if (-not (Test-Path -LiteralPath $Dst)) {
        Copy-Item -LiteralPath $Src -Destination $Dst
    } elseif ((Get-FileHash -LiteralPath $Src).Hash -ne (Get-FileHash -LiteralPath $Dst).Hash) {
        Copy-Item -LiteralPath $Src -Destination "$Dst.new" -Force
        Write-Warn "kept your $Dst; the new default is in $Dst.new"
    }
}

function Copy-Dir([string]$Src, [string]$Dst) {
    Remove-Item -Recurse -Force -LiteralPath $Dst -ErrorAction SilentlyContinue
    Copy-Item -Recurse -LiteralPath $Src -Destination $Dst
}

function Install-Config($Release) {
    $c = Join-Path $Release.Dir 'config'
    Write-Step "Writing configuration to $ConfDir"
    foreach ($d in @('prometheus', 'alertmanager', 'grafana\provisioning\datasources', 'grafana\provisioning\dashboards', 'stock_exporter')) {
        New-Item -ItemType Directory -Force -Path (Join-Path $ConfDir $d) | Out-Null
    }
    foreach ($d in @('prometheus', 'alertmanager', 'grafana', 'kite', 'caddy')) {
        New-Item -ItemType Directory -Force -Path (Join-Path $DataDir $d) | Out-Null
    }
    New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

    foreach ($f in @('prometheus\prometheus.yml', 'alertmanager\alertmanager.yml', 'stock_exporter\watchlist.yml', 'Caddyfile')) {
        Copy-UserFile (Join-Path $c $f) (Join-Path $ConfDir $f)
    }
    foreach ($d in @('prometheus\rules', 'alertmanager\templates', 'grafana\dashboards', 'stock_exporter\textfile')) {
        Copy-Dir (Join-Path $c $d) (Join-Path $ConfDir $d)
    }
    Copy-Item -Force -LiteralPath (Join-Path $c 'stock_exporter\holidays.yml') -Destination (Join-Path $ConfDir 'stock_exporter\holidays.yml')
    Copy-Item -Force -LiteralPath (Join-Path $c 'grafana\provisioning\datasources\datasources.yml') -Destination (Join-Path $ConfDir 'grafana\provisioning\datasources\datasources.yml')
    Format-Template (Join-Path $c 'grafana\provisioning\dashboards\dashboards.yml') (Join-Path $ConfDir 'grafana\provisioning\dashboards\dashboards.yml')

    # Grafana applies the admin password only when it creates its database.
    $ini = Join-Path $ConfDir 'grafana\grafana.ini'
    if (-not (Test-Path -LiteralPath $ini)) {
        $bytes = New-Object byte[] 48
        [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
        $password = ([Convert]::ToBase64String($bytes) -replace '[^A-Za-z0-9]', '').Substring(0, 20)
        Write-Text (Join-Path $ConfDir 'grafana-admin-password') $password
        Format-Template (Join-Path $c 'grafana\grafana.ini') $ini $password
    }

    if ($S.KiteApiKey) {
        $cred = [ordered]@{ api_key = $S.KiteApiKey; api_secret = $S.KiteSecret }
        Write-Text (Join-Path $ConfDir 'kite_credentials.json') ($cred | ConvertTo-Json -Compress)
    }
    Remove-Item -Force -LiteralPath (Join-Path $ConfDir 'kite_api_secret') -ErrorAction SilentlyContinue
    $settings = [pscustomobject]@{ Listen = $S.Listen; Port = $S.Port }
    Write-Text (Join-Path $ConfDir 'settings.json') ($settings | ConvertTo-Json)
}

# Run by each Scheduled Task: starts one service from services.json and restarts it when it exits.
$Launcher = @'
param([Parameter(Mandatory = $true)][string]$Service)
$ErrorActionPreference = 'Stop'
$app = Split-Path -Parent $PSScriptRoot
$def = (Get-Content -Raw -LiteralPath (Join-Path $app 'services.json') | ConvertFrom-Json).$Service
if (-not $def) { throw "unknown service $Service" }
foreach ($p in $def.env.PSObject.Properties) { Set-Item -Path "env:$($p.Name)" -Value $p.Value }

# Windows PowerShell joins -ArgumentList without quoting.
function Format-Arg([string]$a) {
    if ($a -and $a -notmatch '[\s"]') { return $a }
    '"' + (($a -replace '(\\*)"', '$1$1\"') -replace '(\\+)$', '$1$1') + '"'
}
$argLine = ($def.args | ForEach-Object { Format-Arg $_ }) -join ' '
$start = @{ FilePath = $def.exe; WorkingDirectory = $def.dir; NoNewWindow = $true; PassThru = $true }
if ($argLine) { $start.ArgumentList = $argLine }
$log = Join-Path $app "logs\$Service.log"
$err = Join-Path $app "logs\$Service.err.log"

while ($true) {
    # A previous instance survives when only its launcher was stopped.
    Get-Process | Where-Object { $path = $_.Path; $path -and @($def.kill | Where-Object { $path.StartsWith($_, [StringComparison]::OrdinalIgnoreCase) }).Count } |
        Stop-Process -Force -ErrorAction SilentlyContinue
    foreach ($f in @($log, $err)) { if (Test-Path -LiteralPath $f) { Move-Item -Force -LiteralPath $f -Destination "$f.1" } }
    $proc = Start-Process @start -RedirectStandardOutput $log -RedirectStandardError $err
    $proc.WaitForExit()
    Start-Sleep -Seconds 5
}
'@

function Install-Services([string]$Prom, [string]$Am, [string]$Grafana, [string]$Caddy) {
    Write-Step 'Registering Scheduled Tasks'
    $c = $ConfDir
    $d = $DataDir
    $bin = Join-Path $AppDir 'bin'
    $venv = Join-Path $AppDir 'openbb'
    $port = $S.Port
    $defs = [ordered]@{
        prometheus   = @{ exe = "$Prom\prometheus.exe"; dir = $Prom; env = @{}; args = @(
                "--config.file=$c\prometheus\prometheus.yml", "--storage.tsdb.path=$d\prometheus", '--web.listen-address=127.0.0.1:9090',
                "--web.external-url=http://localhost:$port/prometheus/", '--web.route-prefix=/prometheus', '--web.enable-lifecycle') }
        alertmanager = @{ exe = "$Am\alertmanager.exe"; dir = $Am; env = @{}; args = @(
                "--config.file=$c\alertmanager\alertmanager.yml", "--storage.path=$d\alertmanager", '--web.listen-address=127.0.0.1:9093',
                "--web.external-url=http://localhost:$port/alertmanager/", '--web.route-prefix=/alertmanager', '--cluster.listen-address=') }
        grafana      = @{ exe = "$Grafana\bin\grafana.exe"; dir = $Grafana; env = @{}; args = @(
                'server', "--homepath=$Grafana", "--config=$c\grafana\grafana.ini") }
        exporter     = @{ exe = "$bin\stock_exporter.exe"; dir = $AppDir; env = @{}; args = @(
                '--web.listen-address=127.0.0.1:9101', '--feed.source=kite',
                "--kite.credentials-file=$c\kite_credentials.json", '--kite.post-login-redirect=/settings?tab=data-sources&source=kite', "--kite.token-file=$d\kite\kite_session.json",
                "--stock.watchlist-file=$c\stock_exporter\watchlist.yml", "--market.holidays-file=$c\stock_exporter\holidays.yml",
                "--collector.textfile.directory=$c\stock_exporter\textfile") }
        openbb       = @{ exe = "$venv\Scripts\openbb-api.exe"; dir = $venv; args = @()
            env = @{ OPENBB_API_HOST = '127.0.0.1'; OPENBB_API_PORT = '6900'; OPENBB_AUTO_BUILD = 'false'; PYTHONUNBUFFERED = '1' } }
        home         = @{ exe = "$bin\stock_home.exe"; dir = $AppDir; env = @{}; args = @(
                '--web.listen-address=127.0.0.1:8081', '--prometheus.url=http://127.0.0.1:9090/prometheus',
                '--alertmanager.url=http://127.0.0.1:9093/alertmanager', '--grafana.url=http://127.0.0.1:3000/grafana',
                "--market.holidays-file=$c\stock_exporter\holidays.yml", '--openbb.url=http://127.0.0.1:6900', '--exporter.url=http://127.0.0.1:9101') }
        portal       = @{ exe = "$Caddy\caddy.exe"; dir = $AppDir; args = @('run', "--config=$c\Caddyfile", '--adapter=caddyfile')
            env = @{ STOCKOBS_BIND = $S.Listen; STOCKOBS_PORT = [string]$port; XDG_DATA_HOME = "$d\caddy"; XDG_CONFIG_HOME = "$d\caddy" } }
    }
    foreach ($name in $defs.Keys) { $defs[$name].kill = @($defs[$name].exe) }
    # The venv launcher starts the uv-managed interpreter as a child process.
    $defs.openbb.kill = @($venv, (Join-Path $AppDir 'python'))
    Write-Text (Join-Path $AppDir 'services.json') ($defs | ConvertTo-Json -Depth 5)
    Write-Text (Join-Path $bin 'stockobs-run.ps1') $Launcher

    $user = "$env:USERDOMAIN\$env:USERNAME"
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable `
        -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew
    $principal = New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Limited
    $trigger = New-ScheduledTaskTrigger -AtLogOn -User $user
    foreach ($name in $Services) {
        $action = New-ScheduledTaskAction -Execute 'powershell.exe' -WorkingDirectory $AppDir `
            -Argument "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$bin\stockobs-run.ps1`" -Service $name"
        Register-ScheduledTask -TaskName "$TaskPrefix$name" -Action $action -Trigger $trigger -Settings $settings `
            -Principal $principal -Description "Stock Observability: $name" -Force | Out-Null
    }
}

function Get-OurTasks { @(Get-ScheduledTask -TaskName "$TaskPrefix*" -ErrorAction SilentlyContinue) }

function Stop-StockObs {
    $tasks = Get-OurTasks
    if ($tasks.Count -eq 0) { return }
    Write-Step 'Stopping running services'
    $tasks | Stop-ScheduledTask -ErrorAction SilentlyContinue
    Get-Process | Where-Object { $_.Path -and $_.Path.StartsWith($AppDir, [StringComparison]::OrdinalIgnoreCase) } |
        Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

function Assert-PortsFree {
    $busy = foreach ($p in @(9090, 9093, 3000, 9101, 6900, 8081, $S.Port)) {
        if (Get-NetTCPConnection -State Listen -LocalPort $p -ErrorAction SilentlyContinue) { $p }
    }
    if ($busy) { throw "ports already in use: $($busy -join ' ') (stop the programs using them and re-run)" }
}

function Wait-Ready([string]$Name, [string]$Url) {
    for ($i = 0; $i -lt 90; $i++) {
        try {
            Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 2 | Out-Null
            return
        } catch {
            # Any HTTP answer counts for OpenBB, which has no health endpoint.
            if ($Name -eq 'openbb' -and $_.Exception.Response) { return }
        }
        Start-Sleep -Seconds 2
    }
    Write-Warn "$Name is not ready yet; check $LogDir\$Name.err.log"
}

function Test-PortalRoute {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Marker,
        [switch]$Redirect
    )
    $portalHost = $S.Listen
    if ($portalHost -eq '0.0.0.0') { $portalHost = '127.0.0.1' }
    elseif ($portalHost -eq '::') { $portalHost = '::1' }
    if ($portalHost.Contains(':')) { $portalHost = "[$portalHost]" }
    $url = 'http://{0}:{1}{2}' -f $portalHost, $S.Port, $Path

    for ($i = 0; $i -lt 5; $i++) {
        $response = $null
        $status = 0
        $location = ''
        $content = ''
        try {
            $request = [System.Net.HttpWebRequest]::Create($url)
            $request.AllowAutoRedirect = $false
            $request.Timeout = 2000
            $request.ReadWriteTimeout = 2000
            try {
                $response = $request.GetResponse()
            } catch [System.Net.WebException] {
                $response = $_.Exception.Response
            }
            if ($response) {
                $status = [int]$response.StatusCode
                $location = $response.Headers['Location']
                if (-not $Redirect -and $status -ge 200 -and $status -lt 300) {
                    $reader = [System.IO.StreamReader]::new($response.GetResponseStream())
                    try { $content = $reader.ReadToEnd() } finally { $reader.Dispose() }
                }
                $response.Close()
            }
        } catch {
            if ($response) { $response.Close() }
        }

        if ($Redirect) {
            if ($status -ge 300 -and $status -lt 400 -and $location) { return }
        } elseif ($status -ge 200 -and $status -lt 300 -and $content.Contains($Marker)) {
            return
        }
        Start-Sleep -Seconds 1
    }
    $kind = if ($Redirect) { 'is not redirecting' } else { 'is not responding correctly' }
    Write-Warn "$Name portal route $kind`: $url"
}

function Start-StockObs {
    Write-Step 'Starting services'
    foreach ($name in $Services) { Start-ScheduledTask -TaskName "$TaskPrefix$name" }
    Wait-Ready prometheus 'http://127.0.0.1:9090/prometheus/-/ready'
    Wait-Ready alertmanager 'http://127.0.0.1:9093/alertmanager/-/ready'
    Wait-Ready grafana 'http://127.0.0.1:3000/grafana/api/health'
    Wait-Ready exporter 'http://127.0.0.1:9101/'
    Wait-Ready home 'http://127.0.0.1:8081/healthz'
    Wait-Ready openbb 'http://127.0.0.1:6900/'
    Wait-Ready portal "http://127.0.0.1:$($S.Port)/healthz"
    Test-PortalRoute -Name home -Path '/' -Marker 'Stock Observability Platform'
    Test-PortalRoute -Name Grafana -Path '/grafana/api/health' -Marker '"database"'
    Test-PortalRoute -Name Prometheus -Path '/prometheus/-/ready' -Marker 'Prometheus Server is Ready.'
    Test-PortalRoute -Name Alertmanager -Path '/alertmanager/-/ready' -Marker 'OK'
    Test-PortalRoute -Name 'Kite login' -Path '/exporter/login' -Redirect
}

function Remove-Vendor([string[]]$Keep) {
    foreach ($d in @(Get-ChildItem -Directory -LiteralPath (Join-Path $AppDir 'vendor') -ErrorAction SilentlyContinue)) {
        if ($Keep -notcontains $d.FullName) { Remove-Item -Recurse -Force -LiteralPath $d.FullName }
    }
}

function Show-Summary($Release) {
    $url = "http://localhost:$($S.Port)"
    $raw = "https://raw.githubusercontent.com/$Repo/main/install.ps1"
    $kite = if ($S.KiteConfigured) {
        "Kite Connect: log in once per trading day at $url/exporter/login`n  (your app's redirect URL must be http://127.0.0.1:$($S.Port)/exporter/kite/callback)"
    } else {
        "Next step: connect your Zerodha account in Settings > Data Sources at $url/settings?tab=data-sources&source=kite"
    }
    Write-Host @"

Stock Observability $($Release.Version) is installed.

  Home          $url/
  Grafana       $url/grafana/   (user admin, password in $ConfDir\grafana-admin-password)
  Prometheus    $url/prometheus/
  Alertmanager  $url/alertmanager/

$kite

Services start at logon (Task Scheduler, tasks $TaskPrefix*).
Logs:    $LogDir
Config:  $ConfDir
Remove:  & ([scriptblock]::Create((irm $raw))) -Uninstall
"@
}

function Install-StockObs {
    Assert-Preflight
    Resolve-Settings
    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("stockobs-" + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        $release = Get-Release $tmp
        Write-Step "Installing Stock Observability $($release.Version) into $AppDir"
        $prom = Install-Vendor $release 'prometheus' $tmp
        $am = Install-Vendor $release 'alertmanager' $tmp
        $grafana = Install-Vendor $release 'grafana' $tmp
        $caddy = Install-Vendor $release 'caddy' $tmp
        $uvDir = Install-Vendor $release 'uv' $tmp

        Stop-StockObs
        Assert-PortsFree
        $bin = Join-Path $AppDir 'bin'
        New-Item -ItemType Directory -Force -Path $bin | Out-Null
        Copy-Item -Force -Path (Join-Path $release.Dir 'bin\*.exe') -Destination $bin
        Install-OpenBB $release (Join-Path $uvDir 'uv.exe')
        Install-Config $release
        Install-Services $prom $am $grafana $caddy
        Remove-Vendor @($prom, $am, $grafana, $caddy, $uvDir)
        if (-not $NoStart) { Start-StockObs }
        Show-Summary $release
        if (-not $NoStart -and -not $NoBrowser -and -not $S.KiteConfigured) { Start-Process "http://localhost:$($S.Port)/settings?tab=data-sources&source=kite" }
    } finally {
        Remove-Item -Recurse -Force -LiteralPath $tmp -ErrorAction SilentlyContinue
    }
}

function Uninstall-StockObs {
    Write-Step 'Removing Stock Observability services'
    Stop-StockObs
    Get-OurTasks | Unregister-ScheduledTask -Confirm:$false
    foreach ($d in @('bin', 'vendor', 'openbb', 'python', 'cache', 'services.json')) {
        Remove-Item -Recurse -Force -LiteralPath (Join-Path $AppDir $d) -ErrorAction SilentlyContinue
    }
    if ($Purge) {
        Write-Step 'Deleting config and data'
        Remove-Item -Recurse -Force -LiteralPath $AppDir -ErrorAction SilentlyContinue
    } else {
        Write-Step "Kept config in $ConfDir and data in $DataDir (use -Purge to delete them)"
    }
}

function Main {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    # Functions called from Main see $S through dynamic scoping (also under & scriptblock invocation).
    $S = @{ Version = $Version; KiteApiKey = $KiteApiKey; Listen = $Listen; Port = $Port; Archive = $Archive; KiteSecret = $null; KiteConfigured = $false }
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    if ($Uninstall) { Uninstall-StockObs } else { Install-StockObs }
}

# Everything runs from Main, so a partially downloaded script does nothing.
Main

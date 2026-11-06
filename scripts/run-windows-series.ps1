# Runs several plans on Obeid's laptop one after another, unattended: each is
# scripts\run-windows.ps1 start --plan P --machine win-ryzen. It stops at the
# first plan that does not finish (a failure, or a batch that stopped and can be
# resumed: resume it with run-windows.ps1, then run the series again for the
# rest). Windows PowerShell 5.1, ASCII only.
#
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows-series.ps1 [--delay SECONDS] [--cooling TEXT] PLAN[@VARIANT] ...
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows-series.ps1 --delay 120 e1 e10 e4@b2-portable
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows-series.ps1 status    # is a series running, and where is it
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows-series.ps1 stop      # stop it after the current block (graceful)
#
# It runs in the window it was started from, which must stay open; AnyDesk can
# disconnect (--delay gives time for that before the first plan). A lock refuses
# a second series while one runs (two launchers at once would measure on top of
# each other). The log is %LOCALAPPDATA%\qchain-kpi\series-<stamp>.log. Its lines:
#   === <time> <plan>       a plan starts
#   done: <plan>            a plan finished (checksums verified, privacy scan clean)
#   preflight refused ...   the laptop was not quiet at that moment; it waits 5 minutes and tries again (6 times)
#   FAILED: <plan> (exit N) the series stopped here (exit 5: the batch stopped and can be resumed)
#   === series finished     every plan finished

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$state = Join-Path $env:LOCALAPPDATA 'qchain-kpi'
$lockFile = Join-Path $state 'series.lock'
$pidFile = Join-Path $state 'series.pid'
$stopFile = Join-Path $state 'series.stop'
$utf8 = New-Object System.Text.UTF8Encoding $false

# The launcher and the waiting times; the environment settings exist for the tests.
$launcher = Join-Path $PSScriptRoot 'run-windows.ps1'
if ($env:KPI_SERIES_LAUNCHER) { $launcher = $env:KPI_SERIES_LAUNCHER }
$gap = 65          # run ids have minute resolution: the next plan starts in a later minute
if ($env:KPI_SERIES_GAP) { $gap = [int]$env:KPI_SERIES_GAP }
$retries = 6       # a plan whose preflight is refused (exit 3) is tried again this many times,
if ($env:KPI_SERIES_RETRIES) { $retries = [int]$env:KPI_SERIES_RETRIES }
$retryWait = 300   # after this many seconds: a short burst of system work must not end a days-long series
if ($env:KPI_SERIES_RETRY_WAIT) { $retryWait = [int]$env:KPI_SERIES_RETRY_WAIT }

# The launcher runs in the same PowerShell as this script.
$shell = (Get-Process -Id $PID).Path
$shellArgs = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', $launcher)

function Die([string]$Message) {
  [Console]::Error.WriteLine("run-windows-series.ps1: $Message")
  exit 1
}

# Test-SeriesRunning: a running series holds the lock file open. Windows frees it
# when that process ends, however it ends, so a lock can never be left behind.
function Test-SeriesRunning {
  if (-not (Test-Path -LiteralPath $lockFile)) { return $false }
  try {
    $f = [IO.File]::Open($lockFile, [IO.FileMode]::Open, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
    $f.Dispose()
    return $false
  } catch {
    return $true
  }
}

function Get-SeriesPid {
  if (Test-Path -LiteralPath $pidFile) { return ([IO.File]::ReadAllText($pidFile)).Trim() }
  return 'unknown'
}

# Show-Log COUNT: the newest log's main lines.
function Show-Log([int]$Count) {
  if (-not (Test-Path -LiteralPath $state)) { return }
  $newest = @(Get-ChildItem -LiteralPath $state -Filter 'series-*.log' | Sort-Object LastWriteTime | Select-Object -Last 1)
  if ($newest.Count -eq 0) { return }
  [Console]::Out.WriteLine("log: $($newest[0].FullName)")
  $lines = @([IO.File]::ReadAllLines($newest[0].FullName) | Where-Object { $_ -match '^===|^done:|^FAILED|^STOPPED|^preflight refused' })
  foreach ($line in @($lines | Select-Object -Last $Count)) { [Console]::Out.WriteLine($line) }
}

# Write-SeriesLog LINE: to the window and the log.
function Write-SeriesLog([string]$Line) {
  [Console]::Out.WriteLine($Line)
  [IO.File]::AppendAllText($script:log, "$Line`n", $utf8)
}

if ($args.Count -lt 1) { Die 'usage: run-windows-series.ps1 [--delay SECONDS] [--cooling TEXT] PLAN[@VARIANT] ... | status | stop' }
$items = @($args | ForEach-Object { [string]$_ })

if ($items.Count -eq 1 -and $items[0] -ceq 'status') {
  if (Test-SeriesRunning) {
    [Console]::Out.WriteLine("a series is running (process $(Get-SeriesPid))")
    Show-Log 8
  } else {
    [Console]::Out.WriteLine('no series is running')
    Show-Log 4
  }
  exit 0
}

if ($items.Count -eq 1 -and $items[0] -ceq 'stop') {
  if (-not (Test-SeriesRunning)) { Die 'no series is running' }
  [IO.File]::WriteAllText($stopFile, '', $utf8)
  # Ask the batch running now to stop after its current block; the series then ends.
  $rid = ''
  foreach ($name in @(& docker ps --filter 'label=org.qchain.kpi=1' --format '{{.Names}}')) {
    if ([string]$name -like 'kpi-run-*') { $rid = ([string]$name).Substring('kpi-run-'.Length); break }
  }
  if ($rid -ne '') {
    $a = $shellArgs + @('stop', $rid)
    & $shell @a
    exit $LASTEXITCODE
  }
  [Console]::Out.WriteLine('no batch is running at this moment; the series ends before its next plan')
  exit 0
}

# Every other argument is a plan, optionally with the liboqs build: PLAN@VARIANT.
$specs = @()
$delay = 0
$cooling = ''
for ($i = 0; $i -lt $items.Count; $i++) {
  $x = $items[$i]
  if ($x -ceq '--delay' -or $x -ceq '--cooling') {
    if ($i + 1 -ge $items.Count) { Die "$x needs a value" }
    $i++
    if ($x -ceq '--cooling') {
      $cooling = $items[$i]
    } else {
      if ($items[$i] -notmatch '^[0-9]+$') { Die "--delay needs a number of seconds, not $($items[$i])" }
      $delay = [int]$items[$i]
    }
  } elseif ($x -cmatch '^[a-z0-9-]+(@[a-z0-9-]+)?$') {
    $specs += $x
  } else {
    Die "bad plan name: $x"
  }
}
if ($specs.Count -eq 0) { Die 'no plan given' }
foreach ($spec in $specs) {
  $plan = ($spec -split '@')[0]
  $file = Join-Path (Join-Path (Join-Path $repo 'plans') 'windows') "$plan.json"
  if (-not (Test-Path -LiteralPath $file)) { Die "no plan plans/windows/$plan.json" }
}

New-Item -ItemType Directory -Force -Path $state | Out-Null
try {
  # Held open until this process ends (see Test-SeriesRunning).
  $lock = [IO.File]::Open($lockFile, [IO.FileMode]::OpenOrCreate, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None)
} catch {
  Die "a series is already running (process $(Get-SeriesPid)); not starting another"
}
[IO.File]::WriteAllText($pidFile, "$PID`n", $utf8)
if (Test-Path -LiteralPath $stopFile) { Remove-Item -LiteralPath $stopFile -Force }
$script:log = Join-Path $state ('series-{0}.log' -f [DateTime]::UtcNow.ToString('yyyyMMdd-HHmmss'))

Write-SeriesLog "series started (process $PID): $($specs -join ' ')"
if ($delay -gt 0) {
  Write-SeriesLog "waiting $delay s before the first plan (time to disconnect AnyDesk)"
  Start-Sleep -Seconds $delay
}

for ($n = 0; $n -lt $specs.Count; $n++) {
  $spec = $specs[$n]
  if (Test-Path -LiteralPath $stopFile) {
    Write-SeriesLog "STOPPED on request before $spec"
    exit 5
  }
  $parts = $spec -split '@'
  $a = $shellArgs + @('start', '--plan', $parts[0], '--machine', 'win-ryzen')
  if ($parts.Count -gt 1) { $a += @('--variant', $parts[1]) }
  if ($cooling -ne '') { $a += @('--cooling', $cooling) }
  Write-SeriesLog "=== $(Get-Date -Format 'HH:mm:ss') $spec"
  $tried = 0
  while ($true) {
    & $shell @a
    $rc = $LASTEXITCODE
    if ($rc -ne 3 -or $tried -ge $retries) { break }
    $tried++
    Write-SeriesLog "preflight refused for $spec; trying again in ${retryWait}s ($tried of $retries)"
    Start-Sleep -Seconds $retryWait
    if (Test-Path -LiteralPath $stopFile) {
      Write-SeriesLog "STOPPED on request before $spec"
      exit 5
    }
  }
  if ($rc -ne 0) {
    Write-SeriesLog "FAILED: $spec (exit $rc)"
    exit $rc
  }
  Write-SeriesLog "done: $spec"
  if ($n + 1 -lt $specs.Count) { Start-Sleep -Seconds $gap }
}
Write-SeriesLog "=== series finished $(Get-Date -Format 'HH:mm:ss')"
exit 0

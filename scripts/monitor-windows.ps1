# Host monitor for Obeid's Windows laptop (coding plan section 8). Started and
# stopped by scripts\run-windows.ps1. Windows PowerShell 5.1, ASCII only.
#
#   powershell -NoProfile -ExecutionPolicy Bypass -File monitor-windows.ps1 <work-dir> <period-s> <rows-per-chunk> <launcher-pid>
#
# Every sample becomes one CSV row (format in internal/quality/host.go):
#
#   utc_ns,other_pct,throttle,on_ac,alert,detail
#
#   other_pct  host CPU used by everything except the WSL2 VM that runs the
#              container (vmmem, vmmemWSL), the container runtime's helpers
#              (com.docker.*, the docker CLI, wsl*, vmcompute) and this
#              monitor and launcher, as % of all logical CPUs (-1 if the
#              counters are missing)
#   throttle   1 if a performance limit was active (Processor Information
#              "% Performance Limit" below 100, or a thermal zone's
#              "% Passive Limit" below 100), 0 if not, -1 if neither exists
#   on_ac      1 on mains power, 0 on battery, -1 unknown
#   alert      1 if a container that is not ours runs (checked every
#              <rows-per-chunk> samples) or Energy Saver is on
#   detail     the raw readings (numbers and program names only). proc_perf is
#              "% Processor Performance": the CPU's speed as % of its base
#              frequency, recorded for the analysis and never used for a flag
#
# The counters are created once and read with NextValue(), which is much
# lighter than Get-Counter. Rows go into chunk files <work-dir>\host-NNNNNN.csv;
# a finished chunk gets a .ready marker, and the launcher delivers it into the
# run volume. All rows are also kept in host-all.csv. The monitor ends when the
# launcher does, or when the launcher creates <work-dir>\STOP (it then closes
# its last chunk). No user names, host names or paths are recorded.
param(
  [Parameter(Mandatory = $true)][string]$Dir,
  [Parameter(Mandatory = $true)][int]$Period,
  [Parameter(Mandatory = $true)][int]$Rows,
  [Parameter(Mandatory = $true)][int]$LauncherPid
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

$utf8 = New-Object System.Text.UTF8Encoding $false
$header = 'utc_ns,other_pct,throttle,on_ac,alert,detail'
$ncpu = [Environment]::ProcessorCount
$inv = [Globalization.CultureInfo]::InvariantCulture
$all = Join-Path $Dir 'host-all.csv'
if (-not (Test-Path -LiteralPath $all)) { [IO.File]::WriteAllText($all, "$header`n", $utf8) }
Add-Type -AssemblyName System.Windows.Forms

function Get-NowNs { return ([DateTime]::UtcNow.Ticks - 621355968000000000) * 100 }

# New-Counter: a read-only performance counter, primed (its first value is
# always 0), or $null when it does not exist on this machine.
function New-Counter([string]$Category, [string]$Name, [string]$Instance) {
  try {
    $c = New-Object System.Diagnostics.PerformanceCounter($Category, $Name, $Instance, $true)
    [void]$c.NextValue()
    return $c
  } catch {
    return $null
  }
}

# The container runtime: the WSL2 VM and Docker Desktop's helpers (not its
# window, whose activity is ordinary background load).
$runtimePattern = '^(vmmem|vmmemWSL|com\.docker\.[^#]+|docker|wslservice|wslhost|wslrelay|vmcompute)(#\d+)?$'

function Get-RuntimeCounters {
  $list = New-Object System.Collections.ArrayList
  try {
    $names = (New-Object System.Diagnostics.PerformanceCounterCategory 'Process').GetInstanceNames()
  } catch {
    return , $list
  }
  foreach ($n in $names) {
    if ($n -match $runtimePattern) {
      $c = New-Counter 'Process' '% Processor Time' $n
      if ($null -ne $c) { [void]$list.Add($c) }
    }
  }
  return , $list
}

function Get-OwnMs {
  $ms = 0.0
  foreach ($id in @($PID, $LauncherPid)) {
    try { $ms += [Diagnostics.Process]::GetProcessById($id).TotalProcessorTime.TotalMilliseconds } catch { }
  }
  return $ms
}

function Get-EnergySaver {
  try {
    $s = [Windows.System.Power.PowerManager,Windows.System.Power,ContentType=WindowsRuntime]::EnergySaverStatus
    return [string]$s
  } catch {
    return 'n/a'
  }
}

# Busiest process outside the runtime over half a second (diagnosis only,
# when other CPU is high): name and % of one CPU.
function Get-Busiest {
  $a = @{}
  foreach ($p in [Diagnostics.Process]::GetProcesses()) {
    try { $a[$p.Id] = $p.TotalProcessorTime.TotalMilliseconds } catch { }
  }
  Start-Sleep -Milliseconds 500
  $best = ''
  $bestMs = 0.0
  foreach ($p in [Diagnostics.Process]::GetProcesses()) {
    try {
      if ($p.Id -eq $PID -or $p.Id -eq $LauncherPid -or $p.ProcessName -match $runtimePattern -or -not $a.ContainsKey($p.Id)) { continue }
      $d = $p.TotalProcessorTime.TotalMilliseconds - $a[$p.Id]
      if ($d -gt $bestMs) { $bestMs = $d; $best = $p.ProcessName }
    } catch { }
  }
  if ($best -eq '') { return '' }
  return ';busiest_other={0}:{1}' -f ($best -replace '[;,=\s]', '_'), ($bestMs / 5).ToString('0.0', $inv)
}

# Counters of the alert check: containers that are not ours.
function Get-Found {
  $what = @()
  $old = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try {
    # Counts of the running containers and of ours (no output parsing, so no
    # empty lines or quoting to go wrong).
    $all = @(& docker ps -q 2>$null)
    $code = $LASTEXITCODE
    $ours = @(& docker ps -q --filter 'label=org.qchain.kpi=1' 2>$null)
  } finally {
    $ErrorActionPreference = $old
  }
  if ($code -ne 0) {
    $what += 'docker_unreachable'
  } else {
    $n = @($all | Where-Object { $_ -ne $null -and $_ -ne '' }).Count - @($ours | Where-Object { $_ -ne $null -and $_ -ne '' }).Count
    if ($n -gt 0) { $what += "containers_not_ours=$n" }
  }
  return $what
}

$total = New-Counter 'Processor' '% Processor Time' '_Total'
$perfLimit = New-Counter 'Processor Information' '% Performance Limit' '_Total'
$limitFlags = New-Counter 'Processor Information' 'Performance Limit Flags' '_Total'
$procPerf = New-Counter 'Processor Information' '% Processor Performance' '_Total'
$zones = @()
try { $zones = (New-Object System.Diagnostics.PerformanceCounterCategory 'Thermal Zone Information').GetInstanceNames() } catch { }
$temps = New-Object System.Collections.ArrayList
$passives = New-Object System.Collections.ArrayList
foreach ($z in $zones) {
  $t = New-Counter 'Thermal Zone Information' 'Temperature' $z
  if ($null -ne $t) { [void]$temps.Add($t) }
  $p = New-Counter 'Thermal Zone Information' '% Passive Limit' $z
  if ($null -ne $p) { [void]$passives.Add($p) }
}
$runtime = Get-RuntimeCounters
$ownMs = Get-OwnMs
$clock = [Diagnostics.Stopwatch]::StartNew()
$lastMs = 0.0

$seq = 0
$nrow = 0
$chunk = $null
$alert = 0
$found = 'not_checked'
$samples = 0
$stopFile = Join-Path $Dir 'STOP'

function Close-Chunk {
  if ($null -ne $script:chunk) {
    [IO.File]::WriteAllText("$script:chunk.ready", '', $script:utf8)
    $script:chunk = $null
  }
}

try {
  while ($true) {
    $next = ($samples + 1) * $Period * 1000
    while ($clock.Elapsed.TotalMilliseconds -lt $next) {
      if (Test-Path -LiteralPath $stopFile) { break }
      Start-Sleep -Milliseconds 250
    }
    if ((Test-Path -LiteralPath $stopFile) -or $null -eq (Get-Process -Id $LauncherPid -ErrorAction SilentlyContinue)) { break }

    $nowMs = $clock.Elapsed.TotalMilliseconds
    $wallMs = $nowMs - $lastMs
    $lastMs = $nowMs
    $samples++

    # Readings at the end of the interval the row describes.
    $other = -1.0
    $tot = -1.0
    $rt = 0.0
    if ($null -ne $total) {
      $tot = [double]$total.NextValue()
      foreach ($c in $runtime) { try { $rt += [double]$c.NextValue() } catch { } }
      $rt = $rt / $ncpu
      $o = Get-OwnMs
      $ownPct = 0.0
      if ($wallMs -gt 0) { $ownPct = 100.0 * ($o - $ownMs) / ($wallMs * $ncpu) }
      $ownMs = $o
      $other = [Math]::Max(0.0, $tot - $rt - $ownPct)
    }
    $limit = -1.0
    if ($null -ne $perfLimit) { $limit = [double]$perfLimit.NextValue() }
    $flags = -1
    if ($null -ne $limitFlags) { $flags = [int]$limitFlags.NextValue() }
    $speed = -1.0
    if ($null -ne $procPerf) { $speed = [double]$procPerf.NextValue() }
    $tempC = 'n/a'
    if ($temps.Count -gt 0) { $tempC = ([double]$temps[0].NextValue() - 273.15).ToString('0', $inv) }
    $passive = 100.0
    foreach ($c in $passives) { $passive = [Math]::Min($passive, [double]$c.NextValue()) }
    $throttle = -1
    if ($null -ne $perfLimit -or $passives.Count -gt 0) {
      $throttle = 0
      if (($null -ne $perfLimit -and $limit -lt 99.5) -or $passive -lt 99.5) { $throttle = 1 }
    }
    $ps = [System.Windows.Forms.SystemInformation]::PowerStatus
    $onAC = -1
    if ($ps.PowerLineStatus -eq 'Online') { $onAC = 1 } elseif ($ps.PowerLineStatus -eq 'Offline') { $onAC = 0 }
    $battery = [int]([Math]::Round($ps.BatteryLifePercent * 100))
    $saver = Get-EnergySaver

    if ((($samples - 1) % $Rows) -eq 0) {
      $what = @(Get-Found)
      $found = 'none'
      if ($what.Count -gt 0) { $found = $what -join '+' }
      $alert = 0
      if (@($what | Where-Object { $_ -ne 'docker_unreachable' }).Count -gt 0) { $alert = 1 }
    }
    $rowAlert = $alert
    if ($saver -eq 'On') { $rowAlert = 1 }
    $busiest = ''
    if ($other -gt 5) { $busiest = Get-Busiest }

    $row = '{0},{1},{2},{3},{4},total={5};runtime={6};perf_limit={7};limit_flags={8};passive_limit={9};proc_perf={15};temp_c={10};battery_pct={11};energy_saver={12};found={13}{14}' -f `
      (Get-NowNs), $other.ToString('0.000', $inv), $throttle, $onAC, $rowAlert, $tot.ToString('0.0', $inv), $rt.ToString('0.0', $inv), `
      $limit.ToString('0.0', $inv), $flags, $passive.ToString('0.0', $inv), $tempC, $battery, $saver, $found, $busiest, $speed.ToString('0.0', $inv)

    # The runtime's processes come and go (docker CLI calls): re-read the list
    # after each check, so the new counters are primed for the next interval.
    if ((($samples - 1) % $Rows) -eq 0) {
      foreach ($c in $runtime) { $c.Dispose() }
      $runtime = Get-RuntimeCounters
    }

    if ($null -eq $chunk) {
      $seq++
      $chunk = Join-Path $Dir ('host-{0:D6}' -f $seq)
      [IO.File]::WriteAllText("$chunk.csv", "$header`n", $utf8)
      $nrow = 0
    }
    [IO.File]::AppendAllText("$chunk.csv", "$row`n", $utf8)
    [IO.File]::AppendAllText($all, "$row`n", $utf8)
    $nrow++
    if ($nrow -ge $Rows) { Close-Chunk }
  }
} finally {
  Close-Chunk
}

# QChain KPI launcher for Obeid's Windows laptop (Windows PowerShell 5.1,
# Docker Desktop on WSL2). The Mac has scripts/run-mac.sh and the VM
# scripts/run-linux.sh (one launcher per machine). ASCII only.
#
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 start --plan NAME --machine win-ryzen
#       [--variant b1-prod|b2-portable|b3-native] [--seed N] [--cooling TEXT] [--rebuild] [--force]
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 resume RUN_ID [--cooling TEXT] [--force]
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 extend RUN_ID --blocks ID[,ID...] [--cooling TEXT] [--force]
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 stop RUN_ID    # graceful: stops after the current block
#   powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 status
#
# The launcher only prepares and supervises; all measurement and
# orchestration logic is Go inside the image (coding plan sections 2 and 9),
# built from the same recipe as on the other machines. The container is pinned
# to CPU 1 of Docker's WSL2 VM with --cpuset-cpus (E12 plans: the CPUs the plan
# names) and has NO CPU-time quota (--cpus; decision of 10 Oct). It:
#   - refuses to start on a machine that is not ready (trial and main plans;
#     smoke plans only warn), unless --force is given (recorded): on mains
#     power, Energy Saver off, power mode "Best performance" (the laptop's own
#     setting), no container running (Obeid's are stopped for the run), no
#     window of VS Code, Teams, Edge or Chrome open, the laptop quiet, 10 GB
#     free; Docker must use cgroup v2;
#   - builds the image from a clean commit with the buildx builder "kpi" and
#     labels it with that commit. The laptop's liboqs builds are b1-prod and,
#     for E4, b2-portable and b3-native (decision of 21 Oct);
#   - creates the run's container and named volume, measures the host-to-
#     container clock offset and delivers it as session.json;
#   - runs the host monitor (scripts\monitor-windows.ps1) and delivers its
#     chunks into the volume every 10 s, keeps Windows awake for the batch
#     (SetThreadExecutionState) and shows the batch's output. Ctrl-C asks the
#     batch to stop after its current block (resume it later). If the window
#     is closed, the batch stops by itself within minutes (no host samples)
#     and can be resumed;
#   - when the batch is done: copies the results out, verifies SHA256SUMS,
#     runs the privacy scan, adds the INDEX row, and removes our container and
#     volume (only ours, found by name and label; never a prune, never anyone
#     else's). Pushing stays manual.
#
# Windows PowerShell 5.1 rules kept here: no PowerShell 7 syntax; no stderr
# redirection of native commands while errors stop the script; no embedded
# double quotes in native arguments; files written as UTF-8 without BOM with
# LF line endings; $LASTEXITCODE checked after every docker and git call.

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2.0

$Launcher = 'run-windows.ps1 1'
$Machine = 'win-ryzen'
$repo = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$PeriodS = 5        # host monitor sampling period
$RowsPerChunk = 12  # one chunk per minute
$osdir = 'windows'
$runtime = 'docker-wsl2'
$cmem = '3g'
$plandir = 'plans/windows'
# The launcher's notes per run (copied into the run's launcher folder at the end).
$stateRoot = Join-Path $env:LOCALAPPDATA 'qchain-kpi'
$utf8 = New-Object System.Text.UTF8Encoding $false
$inv = [Globalization.CultureInfo]::InvariantCulture
Add-Type -AssemblyName System.Windows.Forms

function Die([string]$Message) {
  [Console]::Error.WriteLine("run-windows.ps1: $Message")
  exit 1
}

function Note([string]$Message) { [Console]::Error.WriteLine("run-windows.ps1: $Message") }

function Write-Text([string]$Path, [string]$Text) { [IO.File]::WriteAllText($Path, $Text, $utf8) }

function Get-NowNs { return ([DateTime]::UtcNow.Ticks - 621355968000000000) * 100 }

# Get-Prop OBJECT NAME: a JSON property, or $null when it is absent.
function Get-Prop($Object, [string]$Name) {
  $p = $Object.PSObject.Properties[$Name]
  if ($null -eq $p) { return $null }
  return $p.Value
}

function Read-Json([string]$Path) { return ([IO.File]::ReadAllText($Path) | ConvertFrom-Json) }

# Invoke-Native EXE ARGS [-Quiet]: runs a program, returns its standard output
# as one string and sets $script:Code. Its standard error goes to the console
# (nowhere with -Quiet).
function Invoke-Native([string]$Exe, [string[]]$A, [switch]$Quiet) {
  $old = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try {
    if ($Quiet) { $out = & $Exe @A 2>$null } else { $out = & $Exe @A }
    $script:Code = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $old
  }
  if ($null -eq $out) { return '' }
  return ((@($out) | ForEach-Object { [string]$_ }) -join "`n").TrimEnd()
}

function Invoke-Docker([string[]]$A, [switch]$Quiet) { return Invoke-Native 'docker' $A -Quiet:$Quiet }

# Invoke-DockerShown ARGS: runs docker with its output on the console.
function Invoke-DockerShown([string[]]$A) {
  $old = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try {
    & docker @A | Out-Host
    $script:Code = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $old
  }
}

function Invoke-Git([string[]]$A) { return Invoke-Native 'git' (@('-C', $repo) + $A) }

# ------------------------------------------------------------------ checks

$script:Problems = New-Object System.Collections.ArrayList
$script:Notes = New-Object System.Collections.ArrayList
function Add-Problem([string]$Text) { [void]$script:Problems.Add($Text) }

function Test-FreeSpace([string]$Path, [string]$Label) {
  $root = [IO.Path]::GetPathRoot((Resolve-Path -LiteralPath $Path).Path)
  $free = (New-Object IO.DriveInfo $root).AvailableFreeSpace
  if ($free -lt 10GB) { Add-Problem "less than 10 GB free on the disk of $Label" }
}

# Assert-DockerReady: Docker reachable, with cgroup v2 (the harness reads
# memory.max, cpu.stat and memory.peak).
function Assert-DockerReady {
  [void](Invoke-Docker @('info') -Quiet)
  if ($script:Code -ne 0) { Die 'Docker is not reachable (is Docker Desktop running?)' }
  $v = Invoke-Docker @('info', '-f', '{{.CgroupVersion}}')
  if ($v -ne '2') { Die 'Docker uses cgroup v1 here; the harness reads cgroup v2 files and cannot run on this machine as built' }
}

function Get-EnergySaver {
  try {
    return [string][Windows.System.Power.PowerManager,Windows.System.Power,ContentType=WindowsRuntime]::EnergySaverStatus
  } catch {
    return 'n/a'
  }
}

# Get-PowerMode: the power mode on mains power (Settings > System > Power).
function Get-PowerMode {
  try {
    $g = (Get-ItemProperty -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Control\Power\User\PowerSchemes' -Name ActiveOverlayAcPowerScheme).ActiveOverlayAcPowerScheme
  } catch {
    return 'n/a'
  }
  switch ($g) {
    'ded574b5-45a0-4f42-8737-46345c09c238' { return 'Best performance' }
    '00000000-0000-0000-0000-000000000000' { return 'Balanced' }
    '961cc777-2547-4f9d-8174-7d86181b8a7a' { return 'Best power efficiency' }
    default { return "overlay $g" }
  }
}

function Get-UpdatePause {
  try {
    return [string](Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings' -Name PauseUpdatesExpiryTime).PauseUpdatesExpiryTime
  } catch {
    return 'not paused'
  }
}

# Get-HostBusy SECONDS: the laptop's CPU use over that time, % of all CPUs (-1 if unknown).
function Get-HostBusy([int]$Seconds) {
  try {
    $c = New-Object System.Diagnostics.PerformanceCounter('Processor', '% Processor Time', '_Total', $true)
    [void]$c.NextValue()
    Start-Sleep -Seconds $Seconds
    return [int][Math]::Round($c.NextValue())
  } catch {
    return -1
  }
}

function Test-Preflight([string]$Kind) {
  $script:Problems.Clear()
  $script:Notes.Clear()
  Assert-DockerReady
  Test-FreeSpace $repo 'the repository'
  Test-FreeSpace $env:SystemDrive 'the system drive (Docker Desktop keeps its data there)'
  $n = @((Invoke-Docker @('ps', '-q')) -split "`n" | Where-Object { $_ -ne '' }).Count
  if ($n -gt 0) { Add-Problem "$n container(s) running; none may run during a batch (Obeid's are stopped for the run)" }
  $ps = [System.Windows.Forms.SystemInformation]::PowerStatus
  if ($ps.PowerLineStatus -ne 'Online') { Add-Problem 'on battery power (plug the laptop in)' }
  $saver = Get-EnergySaver
  if ($saver -eq 'On') { Add-Problem 'Energy Saver is on' }
  $mode = Get-PowerMode
  if ($mode -ne 'Best performance' -and $mode -ne 'n/a') { Add-Problem "the power mode on mains power is '$mode', not 'Best performance' (the laptop's setting)" }
  foreach ($app in @('Code', 'ms-teams', 'Teams', 'msedge', 'chrome')) {
    if (@(Get-Process -Name $app -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowHandle -ne [IntPtr]::Zero }).Count -gt 0) {
      Add-Problem "$app has a window open (close it during trial and main runs)"
    }
  }
  $busy = Get-HostBusy 6
  if ($busy -gt 10) { Add-Problem "the laptop was $busy% busy over 6 s (not quiet)" }
  $pause = Get-UpdatePause
  [void]$script:Notes.Add("laptop busy $busy% over 6 s before the session")
  [void]$script:Notes.Add("power mode on mains power: $mode; Energy Saver: $saver")
  [void]$script:Notes.Add("Windows Update paused until: $pause")
  Note "Windows Update paused until: $pause (it can restart the laptop once the pause ends)"
  Resolve-Preflight $Kind
}

# Resolve-Preflight KIND: reports the problems found; refuses trial and main
# runs with problems unless --force was given.
function Resolve-Preflight([string]$Kind) {
  if ($script:Problems.Count -eq 0) { return }
  foreach ($p in $script:Problems) { Note "preflight: $p" }
  if ($Kind -eq 'smoke') {
    Note 'smoke plan: preflight problems are recorded, not enforced'
  } elseif ($force) {
    Note '--force: continuing despite preflight problems (recorded in the launcher files)'
  } else {
    Die 'preflight failed; fix the problems above (or use --force, which is recorded)'
  }
}

function Get-RepoCommit {
  $dirty = Invoke-Git @('status', '--porcelain', '--', '.', ':!results')
  if ($script:Code -ne 0) { Die 'git status failed' }
  if ($dirty -ne '') { Die 'the harness has uncommitted changes (outside results/); commit them first' }
  $c = Invoke-Git @('log', '-1', '--format=%H', '--', '.', ':!results')
  if ($script:Code -ne 0 -or $c -notmatch '^[0-9a-f]{40}$') { Die 'cannot read the harness commit' }
  return $c
}

# Test-Variant VARIANT PLAN: the laptop's liboqs builds (21 Oct), and a
# non-production build only for E4.
function Test-Variant([string]$Variant, $Plan, [string]$Name) {
  if (@('b1-prod', 'b2-portable', 'b3-native') -notcontains $Variant) {
    Die "the laptop runs the b1-prod build and, for E4, b2-portable and b3-native (decision of 21 Oct); got $Variant"
  }
  $hasE4 = @($Plan.items | Where-Object { $_.exp -eq 'E4' }).Count -gt 0
  if ($Variant -ne 'b1-prod' -and -not $hasE4) { Die "--variant $Variant is for E4 plans (e4, smoke-e4); $Name has no E4 item" }
}

# Assert-Builder: the image builder of the study (created once at set-up, removed after the study).
function Assert-Builder {
  [void](Invoke-Docker @('buildx', 'version') -Quiet)
  if ($script:Code -ne 0) { Die 'docker buildx is not available' }
  [void](Invoke-Docker @('buildx', 'inspect', 'kpi') -Quiet)
  if ($script:Code -ne 0) { Die "no buildx builder 'kpi' on this machine; create it once with: docker buildx create --name kpi --driver docker-container" }
}

function Test-Image([string]$Tag) {
  [void](Invoke-Docker @('image', 'inspect', $Tag) -Quiet)
  return ($script:Code -eq 0)
}

$script:Built = $false
function Build-Image([string]$Variant, [string]$Commit, [bool]$Rebuild) {
  $tag = 'kpi-bench:{0}-{1}' -f $Variant, $Commit.Substring(0, 12)
  if ($Rebuild -or -not (Test-Image $tag)) {
    Assert-Builder
    $script:Built = $true
    Note "building $tag (liboqs gate and harness tests run inside the build)"
    try {
      Invoke-DockerShown @('buildx', 'build', '--builder', 'kpi', '--load', '--progress=plain',
        '--build-arg', "VARIANT=$Variant", '--build-arg', "HARNESS_COMMIT=$Commit",
        '-t', $tag, '-f', (Join-Path $repo 'docker\Dockerfile'), $repo)
      $buildCode = $script:Code
    } finally {
      # The builder container is stopped so it never runs alongside a measurement.
      [void](Invoke-Docker @('buildx', 'stop', 'kpi') -Quiet)
    }
    if ($buildCode -ne 0) { Die 'the image build failed (see above)' }
  }
  $labels = Invoke-Docker @('image', 'inspect', '-f', '{{json .Config.Labels}}', $tag) | ConvertFrom-Json
  $label = Get-Prop $labels 'org.qchain.kpi.harness-commit'
  if ($label -ne $Commit) { Die "image $tag is labelled with commit $label, not $Commit" }
  return $tag
}

# ----------------------------------------------------------------- session

function Measure-Offset([string]$Tag) {
  $bestOff = [long]0
  $bestUnc = [long]-1
  for ($i = 0; $i -lt 3; $i++) {
    $h0 = Get-NowNs
    $c = Invoke-Docker @('run', '--rm', '--network', 'none', '--label', 'org.qchain.kpi=1', '--entrypoint', '/opt/kpi/bin/kpi-batch', $Tag, 'clock')
    $h1 = Get-NowNs
    if ($script:Code -ne 0) { Die 'the clock-offset probe failed' }
    $off = ($h0 -shr 1) + ($h1 -shr 1) - [long]::Parse($c.Trim(), $inv)
    $unc = ($h1 - $h0) -shr 1
    if ($bestUnc -lt 0 -or $unc -lt $bestUnc) { $bestOff = $off; $bestUnc = $unc }
  }
  return @($bestOff, $bestUnc)
}

function Get-ThermalC {
  try {
    $z = @((New-Object System.Diagnostics.PerformanceCounterCategory 'Thermal Zone Information').GetInstanceNames())
    if ($z.Count -eq 0) { return 'n/a' }
    $c = New-Object System.Diagnostics.PerformanceCounter('Thermal Zone Information', 'Temperature', $z[0], $true)
    return ([double]$c.NextValue() - 273.15).ToString('0', $inv)
  } catch {
    return 'n/a'
  }
}

function Write-Session([string]$Path, [long]$Offset, [long]$Unc, [string]$Cpuset) {
  $os = Get-CimInstance Win32_OperatingSystem
  $ver = ''
  try { $ver = (Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion' -Name DisplayVersion).DisplayVersion } catch { }
  $mem = 0
  foreach ($m in @(Get-CimInstance Win32_PhysicalMemory)) { $mem += $m.Capacity }
  $wsl = ''
  try { $wsl = [string](Get-AppxPackage -Name 'MicrosoftCorporationII.WindowsSubsystemForLinux' | Select-Object -First 1).Version } catch { }
  $kernel = Invoke-Docker @('info', '-f', '{{.KernelVersion}}')
  $ps = [System.Windows.Forms.SystemInformation]::PowerStatus
  $power = 'battery'
  if ($ps.PowerLineStatus -eq 'Online') { $power = 'AC' }
  $saver = Get-EnergySaver
  $lowPower = 'n/a'
  if ($saver -eq 'On') { $lowPower = '1' } elseif ($saver -ne 'n/a') { $lowPower = '0' }
  $host_ = [ordered]@{
    runtime          = $runtime
    host_os          = ('{0} {1} ({2})' -f $os.Caption, $ver, $os.Version).Replace('  ', ' ')
    host_cpu         = [string](Get-CimInstance Win32_Processor | Select-Object -First 1).Name.Trim()
    host_mem_gb      = [string][Math]::Round($mem / 1GB)
    docker           = Invoke-Docker @('version', '--format', '{{.Server.Version}}')
    docker_desktop   = Invoke-Docker @('version', '--format', '{{.Server.Platform.Name}}')
    wsl              = ('{0} (kernel {1})' -f $wsl, $kernel).Trim()
    vm_cpus          = Invoke-Docker @('info', '-f', '{{.NCPU}}')
    vm_mem_gb        = ([double](Invoke-Docker @('info', '-f', '{{.MemTotal}}')) / 1GB).ToString('0.00', $inv)
    container_mem    = $cmem
    container_cpuset = $Cpuset
    power            = $power
    battery_pct      = [string][int][Math]::Round($ps.BatteryLifePercent * 100)
    cooling          = $cooling
    low_power_mode   = $lowPower
    thermal_at_start = Get-ThermalC
    launcher         = $Launcher
  }
  foreach ($k in @($host_.Keys)) {
    $v = ([string]$host_[$k]) -replace '[\r\n"\\]', ''
    if ($v.Length -gt 120) { $v = $v.Substring(0, 120) }
    $host_[$k] = $v
  }
  $s = [ordered]@{
    host_offset_ns        = $Offset
    host_offset_unc_ns    = $Unc
    native_p_rate         = 0
    native_e_rate         = 0
    host_logical_cpus     = [Environment]::ProcessorCount
    host_monitor_period_s = $PeriodS
    host                  = $host_
  }
  Write-Text $Path (($s | ConvertTo-Json -Depth 3) -replace "`r`n", "`n")
  [void](Read-Json $Path)
}

function Send-Chunks([string]$Work, [string]$Rid, [int]$Session) {
  foreach ($r in @(Get-ChildItem -LiteralPath $Work -Filter 'host-*.ready' -ErrorAction SilentlyContinue)) {
    $base = $r.FullName.Substring(0, $r.FullName.Length - '.ready'.Length)
    $name = 'host-s{0}-{1}' -f $Session, $r.BaseName.Substring('host-'.Length)
    [void](Invoke-Docker @('cp', "$base.csv", ('kpi-run-{0}:/out/monitor/{1}.csv' -f $Rid, $name)) -Quiet)
    if ($script:Code -ne 0) { continue }
    [void](Invoke-Docker @('cp', (Join-Path $Work 'empty.ok'), ('kpi-run-{0}:/out/monitor/{1}.ok' -f $Rid, $name)) -Quiet)
    if ($script:Code -eq 0) { Move-Item -LiteralPath $r.FullName -Destination "$base.delivered" -Force }
  }
}

function Request-Stop([string]$Rid) {
  $tmp = [IO.Path]::GetTempFileName()
  [void](Invoke-Docker @('cp', $tmp, ('kpi-run-{0}:/out/STOP' -f $Rid)))
  $c = $script:Code
  Remove-Item -LiteralPath $tmp -Force
  return ($c -eq 0)
}

# Invoke-Session RID: one start or resume, from session.json to the batch's
# exit. Returns the batch's exit code (or 'start-failed').
function Invoke-Session([string]$Rid) {
  if ($null -eq ('KpiWin.Power' -as [type])) {
    Add-Type -Namespace KpiWin -Name Power -MemberDefinition '[DllImport("kernel32.dll")] public static extern uint SetThreadExecutionState(uint esFlags);'
  }
  $tag = Invoke-Docker @('inspect', '-f', '{{.Config.Image}}', "kpi-run-$Rid")
  $persist = Join-Path $stateRoot $Rid
  New-Item -ItemType Directory -Force -Path $persist | Out-Null
  $n = @(Get-ChildItem -LiteralPath $persist -Filter 'session-s*.json').Count + 1
  $work = Join-Path $env:TEMP ('kpi-mon-' + [Guid]::NewGuid().ToString('N').Substring(0, 8))
  New-Item -ItemType Directory -Force -Path $work | Out-Null
  $pre = New-Object System.Text.StringBuilder
  foreach ($p in $script:Problems) { [void]$pre.Append("$p`n") }
  if ($force) { [void]$pre.Append("forced`n") }
  foreach ($x in $script:Notes) { [void]$pre.Append("note: $x`n") }
  Write-Text (Join-Path $persist "preflight-s$n.txt") $pre.ToString()

  # The host monitor starts first, so it is delivering before the batch's first block.
  Write-Text (Join-Path $work 'empty.ok') ''
  $monArgs = '-NoProfile -ExecutionPolicy Bypass -File "{0}" "{1}" {2} {3} {4}' -f (Join-Path $repo 'scripts\monitor-windows.ps1'), $work, $PeriodS, $RowsPerChunk, $PID
  $mon = Start-Process -FilePath 'powershell.exe' -ArgumentList $monArgs -WindowStyle Hidden -PassThru
  try { $mon.PriorityClass = [Diagnostics.ProcessPriorityClass]::Idle } catch { }

  $offs = Measure-Offset $tag
  Note ('host-to-container clock offset {0} ns (+/- {1} ns)' -f $offs[0], $offs[1])
  $cpuset = Invoke-Docker @('inspect', '-f', '{{.HostConfig.CpusetCpus}}', "kpi-run-$Rid")
  Write-Session (Join-Path $persist "session-s$n.json") $offs[0] $offs[1] $cpuset
  [void](Invoke-Docker @('cp', (Join-Path $persist "session-s$n.json"), ('kpi-run-{0}:/out/session.json' -f $Rid)))
  if ($script:Code -ne 0) { Die 'could not deliver session.json' }

  [void][KpiWin.Power]::SetThreadExecutionState([uint32]2147483649) # ES_CONTINUOUS | ES_SYSTEM_REQUIRED
  Note "starting batch $Rid (session $n); Ctrl-C or 'scripts\run-windows.ps1 stop $Rid' stops it after the current block"
  $started = Invoke-Docker @('inspect', '-f', '{{.State.StartedAt}}', "kpi-run-$Rid")
  [void](Invoke-Docker @('start', "kpi-run-$Rid"))
  $logs = Start-Process -FilePath 'docker' -ArgumentList "logs -f kpi-run-$Rid" -NoNewWindow -PassThru
  $stopAsked = $false
  $tick = 0
  try { [Console]::TreatControlCAsInput = $true } catch { }
  try {
    while (-not $logs.HasExited) {
      Start-Sleep -Milliseconds 500
      try {
        while ([Console]::KeyAvailable) {
          $k = [Console]::ReadKey($true)
          if (($k.Modifiers -band [ConsoleModifiers]::Control) -and $k.Key -eq 'C' -and -not $stopAsked) {
            $stopAsked = Request-Stop $Rid
            Note "Ctrl-C: the batch stops after its current block; resume it with: scripts\run-windows.ps1 resume $Rid"
          }
        }
      } catch { }
      $tick++
      if (($tick % 20) -eq 0) { Send-Chunks $work $Rid $n }
    }
  } finally {
    try { [Console]::TreatControlCAsInput = $false } catch { }
  }
  while ((Invoke-Docker @('inspect', '-f', '{{.State.Running}}', "kpi-run-$Rid")) -eq 'true') { Start-Sleep -Seconds 2 }
  if ((Invoke-Docker @('inspect', '-f', '{{.State.StartedAt}}', "kpi-run-$Rid")) -eq $started) {
    $code = 'start-failed'
  } else {
    $code = Invoke-Docker @('inspect', '-f', '{{.State.ExitCode}}', "kpi-run-$Rid")
  }
  [void][KpiWin.Power]::SetThreadExecutionState([uint32]2147483648) # ES_CONTINUOUS: normal again

  # The monitor closes its last chunk when it sees STOP.
  Write-Text (Join-Path $work 'STOP') ''
  if (-not $mon.WaitForExit(15000)) { Stop-Process -Id $mon.Id -Force -ErrorAction SilentlyContinue }
  Send-Chunks $work $Rid $n
  if (Test-Path -LiteralPath (Join-Path $work 'host-all.csv')) {
    Copy-Item -LiteralPath (Join-Path $work 'host-all.csv') -Destination (Join-Path $persist "host-monitor-s$n.csv")
  }
  $end = Measure-Offset $tag
  Write-Text (Join-Path $persist "clock-offset-s$n.txt") ("start {0} {1}`nend {2} {3}`n" -f $offs[0], $offs[1], $end[0], $end[1])
  Remove-Item -LiteralPath $work -Recurse -Force
  return $code
}

# ----------------------------------------------------------------- finish

function Get-PrivacyPatterns {
  $p = New-Object System.Collections.ArrayList
  foreach ($v in @($env:COMPUTERNAME, [Environment]::MachineName, $env:USERNAME, $env:USERDOMAIN, $env:USERPROFILE, ($env:USERPROFILE -replace '\\', '/'))) {
    [void]$p.Add([string]$v)
  }
  try {
    foreach ($u in @(Get-CimInstance Win32_UserAccount -Filter "LocalAccount=True")) { if ($u.Name -eq $env:USERNAME) { [void]$p.Add([string]$u.FullName) } }
  } catch { }
  try {
    foreach ($a in @(Get-NetIPAddress -ErrorAction Stop)) {
      $ip = ([string]$a.IPAddress) -replace '%.*$', ''
      if ($ip -notmatch '^(127\.|::1$)') { [void]$p.Add($ip) }
    }
  } catch { }
  try {
    foreach ($a in @(Get-NetAdapter -ErrorAction Stop)) {
      if ($a.MacAddress) { [void]$p.Add([string]$a.MacAddress); [void]$p.Add(([string]$a.MacAddress) -replace '-', ':') }
    }
  } catch { }
  try { [void]$p.Add([string](Get-ItemProperty -LiteralPath 'HKLM:\SOFTWARE\Microsoft\Cryptography' -Name MachineGuid).MachineGuid) } catch { }
  try {
    $serial = [string](Get-CimInstance Win32_BIOS).SerialNumber
    if ($serial -notmatch 'default|o\.?e\.?m|filled|none|^0+$') { [void]$p.Add($serial) }
  } catch { }
  # The laptop's AnyDesk ID (numbers only).
  foreach ($f in @((Join-Path $env:ProgramData 'AnyDesk\system.conf'), (Join-Path $env:ProgramData 'AnyDesk\service.conf'))) {
    try {
      foreach ($m in [regex]::Matches([IO.File]::ReadAllText($f), 'ad\.anynet\.id=(\d{9,10})')) { [void]$p.Add($m.Groups[1].Value) }
    } catch { }
  }
  return @($p | ForEach-Object { $_.Trim() } | Where-Object { $_.Length -ge 4 -and $_ -notmatch '^(localhost|WORKGROUP)$' } | Sort-Object -Unique)
}

function Read-AnyText([string]$Path) {
  if (-not $Path.EndsWith('.gz')) { return [IO.File]::ReadAllText($Path) }
  $fs = [IO.File]::OpenRead($Path)
  try {
    $gz = New-Object IO.Compression.GZipStream($fs, [IO.Compression.CompressionMode]::Decompress)
    return (New-Object IO.StreamReader($gz)).ReadToEnd()
  } finally {
    $fs.Dispose()
  }
}

# Test-Privacy DIR: false if any file contains this machine's identifying
# text. The patterns are only held in memory; matches are reported by file
# name only.
function Test-Privacy([string]$Dir) {
  $pats = Get-PrivacyPatterns
  if ($pats.Count -eq 0) { Die 'privacy scan: no patterns computed' }
  $clean = $true
  foreach ($f in @(Get-ChildItem -LiteralPath $Dir -Recurse -File)) {
    $text = Read-AnyText $f.FullName
    foreach ($pat in $pats) {
      if ($text.IndexOf($pat, [StringComparison]::OrdinalIgnoreCase) -ge 0) {
        Note ('privacy scan: identifying text in {0}' -f $f.FullName.Substring($Dir.Length + 1))
        $clean = $false
        break
      }
    }
  }
  return $clean
}

function Test-Sums([string]$Dir) {
  foreach ($line in ([IO.File]::ReadAllText((Join-Path $Dir 'SHA256SUMS')) -split "`n")) {
    if ($line -eq '') { continue }
    if ($line -notmatch '^([0-9a-f]{64}) [ *](.+)$') { return $false }
    $want = $Matches[1]
    $file = Join-Path $Dir ($Matches[2] -replace '/', '\')
    if (-not (Test-Path -LiteralPath $file)) { return $false }
    if ((Get-FileHash -Algorithm SHA256 -LiteralPath $file).Hash.ToLowerInvariant() -ne $want) { return $false }
  }
  return $true
}

function Add-IndexRow([string]$Dest, [string]$Rid) {
  $idx = Join-Path $repo "results\$osdir\INDEX.md"
  if (-not (Test-Path -LiteralPath $idx)) {
    Write-Text $idx "# Runs on this machine`n`n| Run | Plan | Kind | Status | Attempts | Flagged | Harness commit | Seed |`n|---|---|---|---|---|---|---|---|`n"
  }
  $text = [IO.File]::ReadAllText((Join-Path $Dest 'manifest.json'))
  $m = $text | ConvertFrom-Json
  $seed = [regex]::Match($text, '"seed"\s*:\s*(\d+)').Groups[1].Value
  $row = '| {0} | {1} | {2} | {3} | {4} | {5} | {6} | {7} |' -f $Rid, $m.plan.name, $m.plan.kind, $m.status, `
    $m.counts.attempts, $m.counts.flagged_attempts, ([string]$m.harness_commit).Substring(0, 12), $seed
  [IO.File]::AppendAllText($idx, "$row`n", $utf8)
}

function Save-Console([string]$Rid, [string]$Path, [string]$Prefix) {
  $o = [IO.Path]::GetTempFileName()
  $e = [IO.Path]::GetTempFileName()
  Start-Process -FilePath 'docker' -ArgumentList "logs kpi-run-$Rid" -NoNewWindow -Wait -RedirectStandardOutput $o -RedirectStandardError $e
  $text = $Prefix + [IO.File]::ReadAllText($o) + [IO.File]::ReadAllText($e)
  Write-Text $Path ($text -replace "`r`n", "`n")
  Remove-Item -LiteralPath $o, $e -Force
}

$script:Extending = $false
$script:ExtendFolder = ''
function Complete-Run([string]$Rid, [string]$Code) {
  $persist = Join-Path $stateRoot $Rid
  if ($Code -eq 'start-failed') {
    Note "the batch container did not start (see the messages above); kpi-run-$Rid and kpi-out-$Rid were kept"
    exit 1
  }
  if ($Code -eq '5') {
    Note "batch stopped and can be resumed: scripts\run-windows.ps1 resume $Rid"
    return
  }
  if ($Code -ne '0') {
    $stage = Join-Path $repo "scratch\failed\$Rid"
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
    New-Item -ItemType Directory -Force -Path $stage | Out-Null
    [void](Invoke-Docker @('cp', ('kpi-run-{0}:/out/.' -f $Rid), $stage))
    Save-Console $Rid (Join-Path $stage 'console.txt') ''
    Note "batch ended with exit code $Code; its files were copied to scratch\failed\$Rid for inspection"
    Note "the container kpi-run-$Rid and volume kpi-out-$Rid were kept"
    exit 1
  }
  $stage = Join-Path $repo "scratch\staging\$Rid"
  $dest = Join-Path $repo "results\$osdir\$Rid"
  if (-not $script:Extending -and (Test-Path -LiteralPath $dest)) { Die "$dest already exists" }
  if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
  New-Item -ItemType Directory -Force -Path $stage | Out-Null
  [void](Invoke-Docker @('cp', ('kpi-run-{0}:/out/.' -f $Rid), $stage))
  if ($script:Code -ne 0) { Die "could not copy the results out (container kept)" }
  $prefix = ''
  if ($script:Extending) {
    $old = Join-Path $script:ExtendFolder 'console.txt'
    if (Test-Path -LiteralPath $old) { $prefix = [IO.File]::ReadAllText($old) }
    $prefix += "`n=== extension session (blocks without a clean attempt) ===`n"
  }
  Save-Console $Rid (Join-Path $stage 'console.txt') $prefix
  New-Item -ItemType Directory -Force -Path (Join-Path $stage 'launcher') | Out-Null
  Copy-Item -Path (Join-Path $persist '*') -Destination (Join-Path $stage 'launcher')
  if (-not (Test-Sums $stage)) { Die "SHA256SUMS verification failed in scratch\staging\$Rid (container kept)" }
  if (-not (Test-Privacy $stage)) { Die "privacy scan failed: nothing was moved into results (scratch\staging\$Rid; container kept)" }
  New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dest) | Out-Null
  if ($script:Extending) {
    # The original folder is kept in scratch\backup (made before the extension) and in git.
    Remove-Item -LiteralPath $dest -Recurse -Force
    $idx = Join-Path $repo "results\$osdir\INDEX.md"
    $keep = @(([IO.File]::ReadAllText($idx) -split "`n") | Where-Object { -not $_.StartsWith("| $Rid |") })
    Write-Text $idx (($keep -join "`n").TrimEnd("`n") + "`n")
  }
  Move-Item -LiteralPath $stage -Destination $dest
  Add-IndexRow $dest $Rid
  [void](Invoke-Docker @('rm', "kpi-run-$Rid"))
  [void](Invoke-Docker @('volume', 'rm', "kpi-out-$Rid"))
  Remove-Item -LiteralPath $persist -Recurse -Force
  Note "done: results\$osdir\$Rid (checksums verified, privacy scan clean); container and volume removed"
  Note "to commit: git add results/$osdir/$Rid results/$osdir/INDEX.md; git commit -m `"results: $Rid`""
}

# New-RunContainer: the batch's container and named volume (extra arguments go to kpi-batch run).
function New-RunContainer([string]$Rid, [string]$Plan, [string]$Kind, [string]$Cpuset, [string]$Tag, [string]$Seed, [string]$ImageId, [string[]]$Extra) {
  [void](Invoke-Docker @('volume', 'create', '--label', 'org.qchain.kpi=1', "kpi-out-$Rid"))
  if ($script:Code -ne 0) { Die "could not create the volume kpi-out-$Rid" }
  $a = @('create', '--init', '--name', "kpi-run-$Rid", '--stop-timeout', '30',
    '--label', 'org.qchain.kpi=1', '--label', "org.qchain.kpi.run=$Rid", '--label', "org.qchain.kpi.plan=$Plan",
    '--label', "org.qchain.kpi.kind=$Kind", '--label', "org.qchain.kpi.machine=$Machine",
    '--network', 'none', "--cpuset-cpus=$Cpuset", "--memory=$cmem", "--memory-swap=$cmem", '-e', 'GOMAXPROCS=1',
    '--mount', "type=volume,src=kpi-out-$Rid,dst=/out",
    $Tag, 'run', '--plan', "/opt/kpi/$plandir/$Plan.json", '--run-id', $Rid, '--machine', $Machine,
    '--seed', $Seed, '--image-id', $ImageId) + $Extra
  [void](Invoke-Docker $a)
  if ($script:Code -ne 0) { Die "could not create the container kpi-run-$Rid" }
}

function Get-PlanCpuset($Plan) {
  $c = [string](Get-Prop $Plan 'cpuset')
  if ($c -notmatch '^[0-9]+([-,][0-9]+)*$') { $c = '1' }
  return $c
}

function Wait-Cooldown {
  Note 'cooling down for 120 s after the image build'
  Start-Sleep -Seconds 120
}

# ------------------------------------------------------------------- main

if ($args.Count -lt 1) { Die 'usage: run-windows.ps1 start|resume|extend|stop|status ...' }
$cmd = [string]$args[0]
$rest = @($args | Select-Object -Skip 1 | ForEach-Object { [string]$_ })
$force = $false
$cooling = 'unstated'

# Get-Options: parses the options after the command into a hashtable.
function Get-Options([string[]]$Items, [string[]]$WithValue, [string[]]$Flags) {
  $o = @{ '_' = @() }
  for ($i = 0; $i -lt $Items.Count; $i++) {
    $x = $Items[$i]
    if ($WithValue -contains $x) {
      if ($i + 1 -ge $Items.Count) { Die "$x needs a value" }
      $o[$x] = $Items[$i + 1]
      $i++
    } elseif ($Flags -contains $x) {
      $o[$x] = $true
    } elseif ($x.StartsWith('--')) {
      Die "unknown option $x"
    } else {
      $o['_'] += $x
    }
  }
  return $o
}

switch ($cmd) {
  'start' {
    $o = Get-Options $rest @('--plan', '--machine', '--variant', '--seed', '--cooling') @('--rebuild', '--force')
    if ($o['_'].Count -gt 0) { Die "unexpected argument $($o['_'][0])" }
    $plan = [string]$o['--plan']
    $mach = [string]$o['--machine']
    if ($plan -eq '' -or $mach -eq '') { Die 'start needs --plan and --machine' }
    if ($mach -ne $Machine) { Die "this launcher runs on $Machine only (got $mach)" }
    $variant = 'b1-prod'
    if ($o.ContainsKey('--variant')) { $variant = [string]$o['--variant'] }
    if ($o.ContainsKey('--cooling')) { $cooling = [string]$o['--cooling'] }
    $force = $o.ContainsKey('--force')
    $planfile = Join-Path $repo "$plandir\$plan.json"
    if (-not (Test-Path -LiteralPath $planfile)) { Die "no plan $plandir/$plan.json" }
    $p = Read-Json $planfile
    Test-Variant $variant $p $plan
    $kind = [string]$p.kind
    $cpuset = Get-PlanCpuset $p
    $commit = Get-RepoCommit
    Assert-DockerReady
    $tag = Build-Image $variant $commit $o.ContainsKey('--rebuild')
    if ($script:Built -and $kind -ne 'smoke') { Wait-Cooldown }
    Test-Preflight $kind
    $imageid = Invoke-Docker @('image', 'inspect', '-f', '{{.Id}}', $tag)
    $seed = ''
    if ($o.ContainsKey('--seed')) { $seed = [string]$o['--seed'] }
    if ($seed -eq '') {
      $bytes = New-Object byte[] 8
      [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
      $seed = [BitConverter]::ToUInt64($bytes, 0).ToString($inv)
    }
    $rid = '{0}_{1}_{2}_{3}' -f [DateTime]::UtcNow.ToString('yyyyMMdd-HHmm', $inv), $mach, $runtime, $variant
    if (Test-Path -LiteralPath (Join-Path $repo "results\$osdir\$rid")) { Die "results\$osdir\$rid already exists" }
    New-RunContainer $rid $plan $kind $cpuset $tag $seed $imageid @()
    Note "run ${rid}: plan $plan ($kind), seed $seed, image $tag"
    $code = Invoke-Session $rid
    Complete-Run $rid $code
  }
  'resume' {
    $o = Get-Options $rest @('--cooling') @('--force')
    if ($o['_'].Count -ne 1) { Die 'resume needs a run id' }
    $rid = [string]$o['_'][0]
    if ($o.ContainsKey('--cooling')) { $cooling = [string]$o['--cooling'] }
    $force = $o.ContainsKey('--force')
    Assert-DockerReady
    $labels = Invoke-Docker @('inspect', '-f', '{{json .Config.Labels}}', "kpi-run-$rid") -Quiet
    if ($script:Code -ne 0) { Die "no container kpi-run-$rid" }
    $labels = $labels | ConvertFrom-Json
    if ((Get-Prop $labels 'org.qchain.kpi') -ne '1') { Die "kpi-run-$rid is not a KPI container" }
    if ((Invoke-Docker @('inspect', '-f', '{{.State.Running}}', "kpi-run-$rid")) -ne 'false') { Die "kpi-run-$rid is already running" }
    $kind = [string](Get-Prop $labels 'org.qchain.kpi.kind')
    Test-Preflight $kind
    $code = Invoke-Session $rid
    Complete-Run $rid $code
  }
  'extend' {
    $o = Get-Options $rest @('--blocks', '--cooling') @('--force')
    if ($o['_'].Count -ne 1) { Die 'extend needs a run id' }
    $rid = [string]$o['_'][0]
    $blocks = [string]$o['--blocks']
    if ($blocks -eq '') { Die 'extend needs --blocks ID[,ID...] (for example E1.hqc-3.encaps.r3)' }
    if ($o.ContainsKey('--cooling')) { $cooling = [string]$o['--cooling'] }
    $force = $o.ContainsKey('--force')
    $folder = Join-Path $repo "results\$osdir\$rid"
    if (-not (Test-Path -LiteralPath $folder)) { Die "no results folder $folder" }
    if (-not (Test-Sums $folder)) { Die "the SHA256SUMS of $rid does not verify" }
    $mtext = [IO.File]::ReadAllText((Join-Path $folder 'manifest.json'))
    $m = $mtext | ConvertFrom-Json
    if ($m.status -ne 'done') { Die 'only a finished run can be extended' }
    $kind = [string]$m.plan.kind
    if ($kind -ne 'main') { Die "only main runs can be extended (this one is $kind)" }
    $plan = [string]$m.plan.name
    $cpuset = Get-PlanCpuset (Read-Json (Join-Path $repo "$plandir\$plan.json"))
    $seed = [regex]::Match($mtext, '"seed"\s*:\s*(\d+)').Groups[1].Value
    $variant = [string]$m.build.variant
    $old = [string]$m.harness_commit
    if ([string]$m.machine -ne $Machine) { Die "this launcher runs on $Machine only" }
    $commit = Get-RepoCommit
    [void](Invoke-Git @('cat-file', '-e', "$old^{commit}"))
    if ($script:Code -ne 0) { Die "the run's harness commit $old is not in this repository" }
    # Everything that produces the numbers must be identical. The orchestrator
    # (cmd/kpi-batch) may differ, because it is the part that implements the
    # extension; its changed files are recorded in the run's manifest.
    [void](Invoke-Git @('diff', '--quiet', $old, '--', 'cmd/kpi-prim', 'cmd/kpi-env', 'cmd/kpi-canary-native', 'internal', 'csrc', 'docker', $plandir, 'config', 'go.mod', 'go.sum'))
    if ($script:Code -ne 0) { Die "the measuring sources (cmd/kpi-prim, cmd/kpi-env, cmd/kpi-canary-native, internal, csrc, docker, $plandir, config, go.mod) changed since $old; the run cannot be extended" }
    $orch = ((Invoke-Git @('diff', '--name-only', $old, '--', 'cmd/kpi-batch')) -split "`n" | Where-Object { $_ -ne '' }) -join ','
    $shown = $orch
    if ($shown -eq '') { $shown = 'none' }
    Note "measuring sources are identical to those of the run ($old); orchestrator files that differ: $shown"
    Assert-DockerReady
    [void](Invoke-Docker @('inspect', "kpi-run-$rid") -Quiet)
    if ($script:Code -eq 0) { Die "container kpi-run-$rid already exists" }
    $backup = Join-Path $repo "scratch\backup\$rid"
    if (Test-Path -LiteralPath $backup) { Remove-Item -LiteralPath $backup -Recurse -Force }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $backup) | Out-Null
    Copy-Item -LiteralPath $folder -Destination $backup -Recurse
    Note "backup of the finished run: scratch\backup\$rid"
    $tag = Build-Image $variant $commit $false
    if ($script:Built) { Wait-Cooldown }
    Test-Preflight $kind
    $imageid = Invoke-Docker @('image', 'inspect', '-f', '{{.Id}}', $tag)
    # One argument: Windows PowerShell 5.1 drops empty arguments to programs.
    New-RunContainer $rid $plan $kind $cpuset $tag $seed $imageid @('--extend', $blocks, '--allow-rebuild', "--orchestrator-changed=$orch")
    # The finished run goes back into the volume (without the launcher's own
    # files, which are re-added at the end) and the volume is handed to the
    # unprivileged user.
    $prep = Join-Path $env:TEMP ('kpi-ext-' + [Guid]::NewGuid().ToString('N').Substring(0, 8))
    Copy-Item -LiteralPath $folder -Destination $prep -Recurse
    foreach ($x in @('launcher', 'console.txt')) {
      $px = Join-Path $prep $x
      if (Test-Path -LiteralPath $px) { Remove-Item -LiteralPath $px -Recurse -Force }
    }
    [void](Invoke-Docker @('cp', (($prep -replace '\\', '/') + '/.'), ('kpi-run-{0}:/out/' -f $rid)))
    if ($script:Code -ne 0) { Die 'could not copy the finished run into the volume' }
    Remove-Item -LiteralPath $prep -Recurse -Force
    [void](Invoke-Docker @('run', '--rm', '--network', 'none', '--label', 'org.qchain.kpi=1', '--user', 'root', '--entrypoint', '/bin/chown',
        '--mount', "type=volume,src=kpi-out-$rid,dst=/out", $tag, '-R', 'kpi:kpi', '/out'))
    if ($script:Code -ne 0) { Die 'could not hand the volume to the batch user' }
    $persist = Join-Path $stateRoot $rid
    if (Test-Path -LiteralPath $persist) { Remove-Item -LiteralPath $persist -Recurse -Force }
    New-Item -ItemType Directory -Force -Path $persist | Out-Null
    Copy-Item -Path (Join-Path $folder 'launcher\*') -Destination $persist
    $script:Extending = $true
    $script:ExtendFolder = $folder
    Note "extending run $rid with: $blocks (image $tag)"
    $code = Invoke-Session $rid
    Complete-Run $rid $code
  }
  'stop' {
    if ($rest.Count -ne 1) { Die 'stop needs a run id' }
    $rid = $rest[0]
    if ((Invoke-Docker @('inspect', '-f', '{{.State.Running}}', "kpi-run-$rid") -Quiet) -ne 'true') { Die "kpi-run-$rid is not running" }
    if (-not (Request-Stop $rid)) { Die 'could not deliver the STOP file' }
    Note "requested a graceful stop of ${rid}: it stops after the current block"
  }
  'status' {
    Invoke-DockerShown @('ps', '-a', '--filter', 'label=org.qchain.kpi=1', '--format', 'table {{.Names}}\t{{.Status}}')
  }
  default {
    Die "unknown command $cmd"
  }
}
exit 0

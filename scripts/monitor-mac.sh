#!/bin/bash
# Host monitor for macOS (coding plan §8). Started and stopped by run.sh.
#
#   monitor-mac.sh <work-dir> <period-s> <rows-per-chunk> <exclude-pids>
#
# One `top -l 0 -s <period>` stream is parsed by a small Perl program. Every
# sample becomes one CSV row (format in internal/quality/host.go):
#
#   utc_ns,other_pct,throttle,on_ac,alert,detail
#
#   other_pct  host CPU used by everything except the benchmark VM
#              (com.apple.Virtualization.VirtualMachine), the container
#              runtime's helpers (limactl, colima, ssh, docker) and this
#              monitor and launcher, as % of all logical CPUs
#   throttle   1 if macOS thermal pressure is above nominal
#   on_ac      1 on mains power
#   alert      1 if Low Power Mode is on
#
# Rows go into chunk files <work-dir>/host-NNNNNN.csv; a finished chunk gets
# a .ready marker, and run.sh delivers it into the run volume. All rows are
# also kept in <work-dir>/host-all.csv. Only numbers are recorded: no process
# names, user names or paths.
set -euo pipefail

dir="$1"
period="$2"
rows="$3"
exclude="$4"
ncpu="$(/usr/sbin/sysctl -n hw.logicalcpu)"
mkdir -p "$dir"
fifo="$dir/top.fifo"
rm -f "$fifo"
mkfifo "$fifo"

read -r -d '' PERL_CODE <<'PERL' || true
use strict;
use warnings;
use Time::HiRes qw(gettimeofday);
use IO::Handle;

my ($dir, $ncpu, $rows, $exclude) = @ARGV;
my %excl_pid = map { $_ => 1 } grep { /^\d+$/ } split /,/, $exclude;
$excl_pid{$$} = 1;
# top truncates command names to 16 characters.
my $excl_cmd = qr/^(com\.apple\.Virtua|limactl|colima|ssh|top|docker|caffeinate)/;

my ($seq, $nrow, $fh) = (0, 0, undef);
open(my $all, '>>', "$dir/host-all.csv") or die "host-all.csv: $!";
$all->autoflush(1);
print $all "utc_ns,other_pct,throttle,on_ac,alert,detail\n" if -z "$dir/host-all.csv";

sub open_chunk {
    $seq++;
    my $name = sprintf("%s/host-%06d.csv", $dir, $seq);
    open($fh, '>', $name) or die "$name: $!";
    $fh->autoflush(1);
    print $fh "utc_ns,other_pct,throttle,on_ac,alert,detail\n";
    $nrow = 0;
}

sub close_chunk {
    return unless $fh;
    close($fh);
    $fh = undef;
    my $ready = sprintf("%s/host-%06d.ready", $dir, $seq);
    open(my $r, '>', $ready) or die "$ready: $!";
    close($r);
}

my ($ts, $user, $sys, $idle, $excl) = (undef, undef, undef, undef, 0);
my ($topcmd, $topcpu) = ("", 0);   # busiest process that is not excluded (diagnosis only)
my ($samples, $lpm, $tl, $ac, $pct) = (0, 0, -1, -1, -1);

# Thermal pressure, power source and Low Power Mode, read at the moment top
# prints a sample, i.e. at the end of the interval the sample describes.
sub read_power {
    my $t = `/usr/bin/notifyutil -g com.apple.system.thermalpressurelevel 2>/dev/null`;
    ($tl) = $t =~ /(\d+)\s*$/;
    $tl = -1 unless defined $tl;
    my $b = `/usr/bin/pmset -g batt 2>/dev/null`;
    $ac = $b =~ /AC Power/ ? 1 : 0;
    ($pct) = $b =~ /(\d+)%/;
    $pct = -1 unless defined $pct;
    if ($samples % 12 == 0) {
        my $g = `/usr/bin/pmset -g 2>/dev/null`;
        $lpm = ($g =~ /lowpowermode\s+1/) ? 1 : 0;
    }
}

sub finish_sample {
    $samples++;
    # top's first two samples have no valid per-process CPU figures.
    return if $samples <= 2 || !defined $user || !defined $ts;
    my $other = $user + $sys - $excl / $ncpu;
    $other = 0 if $other < 0;
    my $throttle = $tl < 0 ? -1 : ($tl > 0 ? 1 : 0);
    my $who = "";
    if ($other > 5 && $topcmd ne "") {
        (my $c = $topcmd) =~ s/[;,\r\n]/_/g;
        $who = sprintf(";busiest_other=%s:%.1f", $c, $topcpu);
    }
    my $row = sprintf("%s,%.3f,%d,%d,%d,user=%.2f;sys=%.2f;excluded_core_pct=%.1f;thermal_level=%d;battery_pct=%d%s\n",
        $ts, $other, $throttle, $ac, $lpm, $user, $sys, $excl, $tl, $pct, $who);
    open_chunk() unless $fh;
    print $fh $row;
    print $all $row;
    $nrow++;
    close_chunk() if $nrow >= $rows;
}

while (my $line = <STDIN>) {
    if ($line =~ /^Processes:/) {
        finish_sample();
        my ($s, $us) = gettimeofday();
        $ts = sprintf("%d%06d000", $s, $us);
        ($user, $sys, $idle, $excl) = (undef, undef, undef, 0);
        ($topcmd, $topcpu) = ("", 0);
        read_power();
        next;
    }
    if ($line =~ /^CPU usage:\s+([\d.]+)% user,\s+([\d.]+)% sys,\s+([\d.]+)% idle/) {
        ($user, $sys, $idle) = ($1, $2, $3);
        next;
    }
    if ($line =~ /^\s*(\d+)\s+(.+?)\s+([\d.]+)\s*$/) {
        my ($pid, $cmd, $cpu) = ($1, $2, $3);
        if ($excl_pid{$pid} || $cmd =~ $excl_cmd) { $excl += $cpu; }
        elsif ($cpu > $topcpu) { ($topcmd, $topcpu) = ($cmd, $cpu); }
    }
}
finish_sample();
close_chunk();
PERL

/usr/bin/top -l 0 -s "$period" -n 25 -stats pid,command,cpu -o cpu > "$fifo" &
toppid=$!
/usr/bin/perl -e "$PERL_CODE" -- "$dir" "$ncpu" "$rows" "$exclude,$$" < "$fifo" &
perlpid=$!
trap 'kill "$toppid" 2>/dev/null || true' TERM INT
wait "$perlpid" || true            # returns when perl ends or a signal arrives
kill "$toppid" 2>/dev/null || true
wait "$perlpid" 2>/dev/null || true # let perl write its last row and close the chunk
rm -f "$fifo"

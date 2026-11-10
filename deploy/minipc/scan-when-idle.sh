#!/bin/sh
# Called every 30 minutes by gpuscanner-scan.timer. Scans when the CPU was
# under 30% busy in the last sysstat sample (10-minute average), or when the
# last successful scan is 2 hours old.
set -u
STAMP="$HOME/.local/state/gpuscanner/last_scan"
mkdir -p "$(dirname "$STAMP")"

now=$(date +%s)
last=$(cat "$STAMP" 2>/dev/null || echo 0)
idle=$(LC_ALL=C sar -u 2>/dev/null | awk '$1 ~ /^[0-9]/ && $NF ~ /^[0-9.]+$/ { v = $NF } END { print v }')
busy=$(awk -v i="${idle:-0}" 'BEGIN { printf "%d", 100 - i }')

if [ $((now - last)) -lt 7200 ] && [ "$busy" -ge 30 ]; then
	echo "skip: CPU ${busy}% busy, last scan $(((now - last) / 60)) min ago"
	exit 0
fi

echo "scan: CPU ${busy}% busy, last scan $(((now - last) / 60)) min ago"
"$(dirname "$(readlink -f "$0")")/run-job.sh" scan && echo "$now" >"$STAMP"

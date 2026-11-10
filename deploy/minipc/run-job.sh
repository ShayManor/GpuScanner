#!/bin/sh
# Runs one compose job (scan or blog) and records the outcome for Pantheos
# monitoring as a node_exporter textfile, which vmagent scrapes every 15s.
set -u
JOB="$1"
DIR="$(dirname "$(readlink -f "$0")")"
PROM_DIR=/var/lib/node_exporter/textfile
PROM="$PROM_DIR/gpuscanner_$JOB.prom"

start=$(date +%s)
"$DIR/compose.sh" run --rm "$JOB"
code=$?
end=$(date +%s)

# Keep the previous success time when this run failed.
last_ok=$(awk -v j="$JOB" '$1 == "gpuscanner_job_last_success_timestamp_seconds{job=\"" j "\"}" { print $2 }' "$PROM" 2>/dev/null)
[ "$code" -eq 0 ] && last_ok=$end

{
	echo "# HELP gpuscanner_job_last_run_timestamp_seconds When the job last finished."
	echo "# TYPE gpuscanner_job_last_run_timestamp_seconds gauge"
	echo "gpuscanner_job_last_run_timestamp_seconds{job=\"$JOB\"} $end"
	echo "# HELP gpuscanner_job_last_success_timestamp_seconds When the job last exited 0."
	echo "# TYPE gpuscanner_job_last_success_timestamp_seconds gauge"
	[ -n "$last_ok" ] && echo "gpuscanner_job_last_success_timestamp_seconds{job=\"$JOB\"} $last_ok"
	echo "# HELP gpuscanner_job_exit_code Exit code of the last run."
	echo "# TYPE gpuscanner_job_exit_code gauge"
	echo "gpuscanner_job_exit_code{job=\"$JOB\"} $code"
	echo "# HELP gpuscanner_job_duration_seconds Wall time of the last run."
	echo "# TYPE gpuscanner_job_duration_seconds gauge"
	echo "gpuscanner_job_duration_seconds{job=\"$JOB\"} $((end - start))"
	if [ "$JOB" = scan ]; then
		echo "# HELP gpuscanner_rows GPU offers in the database, by provider."
		echo "# TYPE gpuscanner_rows gauge"
		docker exec gpuscanner-db-1 psql -U postgres -d gpuscanner -At -F ' ' \
			-c "select source, count(*) from gpus group by source order by source" |
			awk '{ print "gpuscanner_rows{source=\"" $1 "\"} " $2 }'
	fi
} >"$PROM.tmp" && mv "$PROM.tmp" "$PROM"

exit "$code"

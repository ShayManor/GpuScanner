#!/bin/sh
# Runs once, when the db volume is first created. Mirrors the Supabase tables
# and gives PostgREST a read-only anon role and a read-write service role.
set -eu
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" <<SQL
create table gpus (
	id uuid primary key,
	source text,
	location text,
	reliability double precision,
	duration_hours double precision,
	name text,
	vram_mb integer,
	total_flops double precision,
	gpu_mem_bw_gbps double precision,
	num_gpus integer,
	cpu_cores double precision,
	cpu_name text,
	cpu_ghz double precision,
	cpu_arch text,
	ram_mb integer,
	disk_space_gb double precision,
	disk_bw_gbps double precision,
	disk_name text,
	upload_mbps double precision,
	download_mbps double precision,
	total_cost_ph double precision,
	gpu_cost_ph double precision,
	disk_cost_ph double precision,
	upload_cost_ph double precision,
	download_cost_ph double precision,
	flops_per_dollar_ph double precision,
	updated_at timestamptz,
	url text,
	_id text,
	score double precision,
	score_dollar_ph double precision
);
create index gpus_source_idx on gpus (source);

create table blogs (
	id bigint generated always as identity primary key,
	title text not null,
	data text not null,
	created_at timestamptz not null default now()
);

create role web_anon nologin;
create role service_role nologin;
create role authenticator login noinherit password '$AUTHENTICATOR_PASSWORD';
grant web_anon, service_role to authenticator;

grant usage on schema public to web_anon, service_role;
grant select on gpus, blogs to web_anon;
grant select, insert, update, delete on gpus, blogs to service_role;
SQL

create extension if not exists pgcrypto;

create table users (
    id uuid primary key default gen_random_uuid(),
    email text unique not null,
    password_hash text not null,
    created_at timestamptz not null default now()
);

create table jobs (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id),
    status text not null default 'queued', -- queued | processing | done | failed
    audio_path text not null,
    error text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index jobs_user_id_idx on jobs(user_id);

create table transcripts (
    job_id uuid primary key references jobs(id),
    segments jsonb not null -- [{start, end, speaker, text}, ...]
);

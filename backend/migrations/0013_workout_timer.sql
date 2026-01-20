create table if not exists workout_plans (
    id bigserial primary key,
    chat_id bigint not null unique,
    payload jsonb not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists workout_sessions (
    id bigserial primary key,
    chat_id bigint not null,
    plan_id bigint references workout_plans(id) on delete set null,
    plan_snapshot jsonb not null,
    status text not null,
    phase text not null,
    exercise_index int not null default 0,
    set_index int not null default 0,
    timer_kind text,
    timer_started_at timestamptz,
    timer_duration_sec int,
    warmup_ended_at timestamptz,
    paused_at timestamptz,
    paused_total_sec int not null default 0,
    started_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_workout_sessions_chat_status on workout_sessions(chat_id, status);

create table if not exists workout_sets (
    id bigserial primary key,
    session_id bigint not null references workout_sessions(id) on delete cascade,
    exercise_index int not null,
    set_index int not null,
    is_warmup boolean not null default false,
    exercise_name text not null,
    exercise_type text not null,
    target_weight numeric(6,2) not null default 0,
    target_reps int not null default 0,
    target_duration_sec int not null default 0,
    actual_weight numeric(6,2) not null default 0,
    actual_reps int not null default 0,
    actual_duration_sec int not null default 0,
    completed_at timestamptz not null default now()
);

create index if not exists idx_workout_sets_session on workout_sets(session_id);

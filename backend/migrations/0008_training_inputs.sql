do $$
begin
    if exists (
        select 1
        from information_schema.tables
        where table_schema = 'public'
          and table_name = 'users'
    ) and not exists (
        select 1
        from information_schema.tables
        where table_schema = 'public'
          and table_name = 'workout_users'
    ) then
        execute 'alter table users rename to workout_users';
    end if;
end $$;

create extension if not exists "pgcrypto";

do $$
begin
    if not exists (select 1 from pg_type where typname = 'fitness_level') then
        create type fitness_level as enum ('beginner', 'intermediate', 'advanced');
    end if;
    if not exists (select 1 from pg_type where typname = 'fitness_goal') then
        create type fitness_goal as enum ('hypertrophy', 'strength', 'fat_loss');
    end if;
end $$;

create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    telegram_chat_id bigint not null unique,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists training_inputs (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    level fitness_level not null,
    goal fitness_goal not null,
    days_per_week int not null check (days_per_week between 2 and 6),
    injuries text[] not null default '{}'::text[],
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (user_id)
);

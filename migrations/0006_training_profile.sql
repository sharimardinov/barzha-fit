alter table user_profiles
    add column if not exists training_years int;

create table if not exists training_profiles (
    chat_id bigint primary key,
    bench_kg int,
    pullups int,
    run_km numeric(5,2),
    injuries text,
    goal text,
    pharma boolean,
    trainings_per_week int,
    dislikes text,
    cannot_do text,
    updated_at timestamptz not null default now()
);

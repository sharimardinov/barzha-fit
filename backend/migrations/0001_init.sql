create table if not exists plans (
                                     user_id    bigint primary key,
                                     text       text not null,
                                     created_at timestamptz not null default now()
);

create table if not exists users (
                                     user_id    bigint primary key,
                                     cycle_day  int not null default 1,
                                     created_at timestamptz not null default now(),
                                     updated_at timestamptz not null default now()
);

create table if not exists workout_days (
                                            user_id    bigint not null,
                                            day_date   date not null,
                                            cycle_day  int not null,
                                            status     text not null, -- 'done' | 'skip'
                                            created_at timestamptz not null default now(),
                                            primary key (user_id, day_date)
);

create table if not exists bot_users (
                                         chat_id bigint primary key,
                                         created_at timestamptz not null default now()
);

alter table bot_users
    add column if not exists morning_enabled boolean not null default true;

create table if not exists meals (
                                     id bigserial primary key,
                                     chat_id bigint not null,
                                     eaten_at timestamptz not null default now(),
                                     text text not null,
                                     kcal int not null default 0,
                                     protein_g int not null default 0,
                                     fat_g int not null default 0,
                                     carbs_g int not null default 0,
                                     ai_raw jsonb
);

create index if not exists idx_meals_chat_date on meals(chat_id, eaten_at);

create table if not exists user_profiles (
                                             chat_id bigint primary key,
                                             sex text,              -- 'm'/'f'/null
                                             age int,
                                             height_cm int,
                                             weight_kg numeric(5,2),
                                             bodyfat_pct numeric(5,2),
                                             activity text,         -- 'low'|'mid'|'high'
                                             updated_at timestamptz not null default now()
);

create table if not exists user_targets (
                                            chat_id bigint primary key,
                                            kcal_target int not null,
                                            protein_g int not null,
                                            fat_g int not null,
                                            carbs_g int not null,
                                            source text not null,  -- 'manual'|'calc'|'gpt'
                                            updated_at timestamptz not null default now()
);

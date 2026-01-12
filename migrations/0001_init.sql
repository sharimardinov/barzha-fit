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
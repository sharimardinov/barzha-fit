create table if not exists plans (
                                     user_id    bigint primary key,
                                     text       text not null,
                                     created_at timestamptz not null default now()
);
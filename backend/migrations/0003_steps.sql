create table if not exists steps_days (
                                     user_id bigint not null,
                                     day_date date not null,
                                     steps int not null default 0,
                                     created_at timestamptz not null default now(),
                                     primary key (user_id, day_date)
);

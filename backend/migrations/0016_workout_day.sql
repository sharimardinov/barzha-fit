alter table workout_sessions
    add column if not exists workout_day int not null default 1;

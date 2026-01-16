alter table user_targets add column if not exists steps_target int not null default 10000;

alter table bot_users add column if not exists hard_enabled boolean not null default false;

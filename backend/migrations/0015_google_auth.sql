create table if not exists google_auth (
    google_sub text primary key,
    chat_id bigint not null unique,
    email text,
    name text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_google_auth_chat_id on google_auth(chat_id);

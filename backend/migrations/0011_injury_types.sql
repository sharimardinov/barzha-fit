create table if not exists injury_types (
    code text primary key,
    label text not null,
    created_at timestamptz not null default now()
);

insert into injury_types (code, label)
values
    ('shoulder', 'Плечо'),
    ('lower_back', 'Поясница'),
    ('knee', 'Колено')
on conflict (code) do update set
    label=excluded.label;

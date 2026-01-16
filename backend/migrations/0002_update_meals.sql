-- Удаляем старую колонку calories
alter table meals drop column if exists calories;

-- Добавляем новые колонки
alter table meals add column if not exists kcal int not null default 0;
alter table meals add column if not exists protein_g int not null default 0;
alter table meals add column if not exists fat_g int not null default 0;
alter table meals add column if not exists carbs_g int not null default 0;
alter table meals add column if not exists ai_raw jsonb;
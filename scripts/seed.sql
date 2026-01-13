\set ON_ERROR_STOP on

-- Required variables: chat_id, days

insert into bot_users (chat_id, morning_enabled)
values (:chat_id, true)
on conflict (chat_id) do update set morning_enabled = true;

insert into plans (user_id, text)
values (
  :chat_id,
  '1\nWorkout A\n2\nRest\n3\nWorkout B\n4\nRest\n5\nWorkout C\n6\nRest\n7\nRest'
)
on conflict (user_id) do update set text = excluded.text;

insert into user_profiles (chat_id, sex, age, height_cm, weight_kg, bodyfat_pct, activity)
values (:chat_id, 'm', 30, 180, 82.5, 18.0, 'mid')
on conflict (chat_id) do update set
  sex = excluded.sex,
  age = excluded.age,
  height_cm = excluded.height_cm,
  weight_kg = excluded.weight_kg,
  bodyfat_pct = excluded.bodyfat_pct,
  activity = excluded.activity,
  updated_at = now();

insert into user_targets (chat_id, kcal_target, protein_g, fat_g, carbs_g, steps_target, source)
values (:chat_id, 2400, 170, 70, 250, 10000, 'manual')
on conflict (chat_id) do update set
  kcal_target = excluded.kcal_target,
  protein_g = excluded.protein_g,
  fat_g = excluded.fat_g,
  carbs_g = excluded.carbs_g,
  steps_target = excluded.steps_target,
  source = excluded.source,
  updated_at = now();

select to_regclass('steps_days') is not null as steps_exists \gset
\if :steps_exists
with days as (
  select generate_series(current_date - (:days - 1), current_date, interval '1 day')::date as day_date
)
insert into steps_days (user_id, day_date, steps)
select :chat_id, day_date, (5000 + (random() * 12000))::int
from days
on conflict (user_id, day_date) do update set steps = excluded.steps;
\endif

with days as (
  select generate_series(current_date - (:days - 1), current_date, interval '1 day')::date as day_date
)
insert into workout_days (user_id, day_date, cycle_day, status)
select
  :chat_id,
  day_date,
  ((extract(dow from day_date)::int + 6) % 7) + 1,
  case when random() < 0.7 then 'done' else 'skip' end
from days
on conflict (user_id, day_date) do update set
  cycle_day = excluded.cycle_day,
  status = excluded.status;

with days as (
  select generate_series(current_date - (:days - 1), current_date, interval '1 day')::date as day_date
), meals as (
  select day_date, meal_no
  from days
  cross join generate_series(1, 2) as meal_no
)
insert into meals (chat_id, eaten_at, text, kcal, protein_g, fat_g, carbs_g)
select
  :chat_id,
  day_date + make_interval(hours => 8 + meal_no * 5),
  'Meal ' || meal_no,
  (350 + (random() * 500))::int,
  (20 + (random() * 40))::int,
  (10 + (random() * 20))::int,
  (30 + (random() * 80))::int
from meals;

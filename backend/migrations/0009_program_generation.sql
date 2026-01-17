create table if not exists exercises (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    muscle_group text not null,
    type text[] not null default '{}'::text[],
    level fitness_level[] not null default '{}'::fitness_level[],
    priority text not null check (priority in ('main', 'secondary', 'accessory')),
    contraindications text[] not null default '{}'::text[],
    substitute_for text[] not null default '{}'::text[],
    prehab_target text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_exercises_muscle_group on exercises(muscle_group);

create table if not exists program_templates (
    id uuid primary key default gen_random_uuid(),
    name text not null unique,
    days text[] not null,
    structure jsonb not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists periodization (
    week int primary key,
    intensity text not null check (intensity in ('light', 'medium', 'heavy')),
    percent_1rm text,
    reps text,
    rest text,
    created_at timestamptz not null default now()
);

create table if not exists user_programs (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    template_id uuid not null references program_templates(id),
    start_date date not null,
    current_week int not null default 1 check (current_week > 0),
    days_generated jsonb not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_user_programs_user_id on user_programs(user_id);

insert into program_templates (name, days, structure)
values
    (
        'full_body',
        array['full_body','full_body','full_body'],
        $${
          "days": [
            {"name":"Day 1","type":"full_body","muscle_groups":["chest","back","quads","hamstrings","glutes","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]},
            {"name":"Day 2","type":"full_body","muscle_groups":["chest","back","quads","hamstrings","glutes","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]},
            {"name":"Day 3","type":"full_body","muscle_groups":["chest","back","quads","hamstrings","glutes","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]}
          ]
        }$$::jsonb
    ),
    (
        'push_pull_legs',
        array['push','pull','legs'],
        $${
          "days": [
            {"name":"Push","type":"push","muscle_groups":["chest","shoulders_front","shoulders_side","triceps"]},
            {"name":"Pull","type":"pull","muscle_groups":["back","biceps","shoulders_rear"]},
            {"name":"Legs","type":"legs","muscle_groups":["quads","hamstrings","glutes"]}
          ]
        }$$::jsonb
    ),
    (
        'upper_lower_x2',
        array['upper','lower','upper','lower'],
        $${
          "days": [
            {"name":"Upper 1","type":"upper","muscle_groups":["chest","back","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]},
            {"name":"Lower 1","type":"lower","muscle_groups":["quads","hamstrings","glutes"]},
            {"name":"Upper 2","type":"upper","muscle_groups":["chest","back","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]},
            {"name":"Lower 2","type":"lower","muscle_groups":["quads","hamstrings","glutes"]}
          ]
        }$$::jsonb
    ),
    (
        'upper_lower_arm_day',
        array['upper','lower','upper','lower','arms'],
        $${
          "days": [
            {"name":"Upper 1","type":"upper","muscle_groups":["chest","back","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]},
            {"name":"Lower 1","type":"lower","muscle_groups":["quads","hamstrings","glutes"]},
            {"name":"Upper 2","type":"upper","muscle_groups":["chest","back","shoulders_front","shoulders_side","shoulders_rear","biceps","triceps"]},
            {"name":"Lower 2","type":"lower","muscle_groups":["quads","hamstrings","glutes"]},
            {"name":"Arms","type":"arms","muscle_groups":["biceps","triceps","shoulders_side","shoulders_rear"]}
          ]
        }$$::jsonb
    ),
    (
        'ppl_x2',
        array['push','pull','legs','push','pull','legs'],
        $${
          "days": [
            {"name":"Push 1","type":"push","muscle_groups":["chest","shoulders_front","shoulders_side","triceps"]},
            {"name":"Pull 1","type":"pull","muscle_groups":["back","biceps","shoulders_rear"]},
            {"name":"Legs 1","type":"legs","muscle_groups":["quads","hamstrings","glutes"]},
            {"name":"Push 2","type":"push","muscle_groups":["chest","shoulders_front","shoulders_side","triceps"]},
            {"name":"Pull 2","type":"pull","muscle_groups":["back","biceps","shoulders_rear"]},
            {"name":"Legs 2","type":"legs","muscle_groups":["quads","hamstrings","glutes"]}
          ]
        }$$::jsonb
    )
on conflict (name) do update set
    days=excluded.days,
    structure=excluded.structure,
    updated_at=now();

insert into periodization (week, intensity, percent_1rm, reps, rest)
values
    (1, 'light', '60-70%', '10-12', '60-90s'),
    (2, 'medium', '70-80%', '8-10', '90-120s'),
    (3, 'heavy', '80-90%', '4-6', '120-180s'),
    (4, 'light', '55-65%', '10-12', '60-90s')
on conflict (week) do update set
    intensity=excluded.intensity,
    percent_1rm=excluded.percent_1rm,
    reps=excluded.reps,
    rest=excluded.rest;

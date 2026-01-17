delete from periodization where week > 3;

insert into periodization (week, intensity, percent_1rm, reps, rest)
values
    (1, 'light', '45-50%', '20-25', '1:00-1:30'),
    (2, 'medium', '60-70%', '10-12', 'up to 2:30'),
    (3, 'heavy', '80-90%', '3-5', '2+ min')
on conflict (week) do update set
    intensity=excluded.intensity,
    percent_1rm=excluded.percent_1rm,
    reps=excluded.reps,
    rest=excluded.rest;

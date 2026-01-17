package db

import (
	"context"
	"database/sql"

	"barzhafit/backend/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExerciseRepo struct {
	db *pgxpool.Pool
}

func NewExerciseRepo(db *pgxpool.Pool) *ExerciseRepo { return &ExerciseRepo{db: db} }

func (r *ExerciseRepo) ListByMuscleGroups(ctx context.Context, groups []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error) {
	if len(groups) == 0 {
		return []domain.Exercise{}, nil
	}
	if injuries == nil {
		injuries = []string{}
	}
	rows, err := r.db.Query(ctx, `
		select id, name, muscle_group, type, level, priority, contraindications, substitute_for, prehab_target
		from exercises
		where muscle_group = any($1)
		  and (cardinality(level)=0 or $2::fitness_level = any(level))
		  and not (contraindications && $3::text[])
		order by name
	`, groups, string(level), injuries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Exercise, 0)
	for rows.Next() {
		var item domain.Exercise
		var typ []string
		var lvl []string
		var contraindications []string
		var substituteFor []string
		var prehab sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.MuscleGroup,
			&typ,
			&lvl,
			&item.Priority,
			&contraindications,
			&substituteFor,
			&prehab,
		); err != nil {
			return nil, err
		}
		item.Type = typ
		item.Level = toFitnessLevels(lvl)
		item.Contraindications = contraindications
		item.SubstituteFor = substituteFor
		if prehab.Valid {
			item.PrehabTarget = prehab.String
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *ExerciseRepo) ListPrehabByTargets(ctx context.Context, targets []string, level domain.FitnessLevel, injuries []string) ([]domain.Exercise, error) {
	if len(targets) == 0 {
		return []domain.Exercise{}, nil
	}
	if injuries == nil {
		injuries = []string{}
	}
	rows, err := r.db.Query(ctx, `
		select id, name, muscle_group, type, level, priority, contraindications, substitute_for, prehab_target
		from exercises
		where prehab_target = any($1)
		  and (cardinality(level)=0 or $2::fitness_level = any(level))
		  and not (contraindications && $3::text[])
		order by name
	`, targets, string(level), injuries)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Exercise, 0)
	for rows.Next() {
		var item domain.Exercise
		var typ []string
		var lvl []string
		var contraindications []string
		var substituteFor []string
		var prehab sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.MuscleGroup,
			&typ,
			&lvl,
			&item.Priority,
			&contraindications,
			&substituteFor,
			&prehab,
		); err != nil {
			return nil, err
		}
		item.Type = typ
		item.Level = toFitnessLevels(lvl)
		item.Contraindications = contraindications
		item.SubstituteFor = substituteFor
		if prehab.Valid {
			item.PrehabTarget = prehab.String
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func toFitnessLevels(in []string) []domain.FitnessLevel {
	if len(in) == 0 {
		return []domain.FitnessLevel{}
	}
	out := make([]domain.FitnessLevel, 0, len(in))
	for _, item := range in {
		out = append(out, domain.FitnessLevel(item))
	}
	return out
}

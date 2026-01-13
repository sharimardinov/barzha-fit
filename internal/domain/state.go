package domain

type State string

const (
	StateNone               State = ""
	StateWaitMealText       State = "wait_meal_text"
	StateWaitPlanText       State = "wait_plan_text"
	StateWaitProfileSex     State = "wait_profile_sex"
	StateWaitProfileHeight  State = "wait_profile_height"
	StateWaitProfileWeight  State = "wait_profile_weight"
	StateWaitProfileBodyFat State = "wait_profile_bodyfat"
	StateWaitProfileAge     State = "wait_profile_age"
	StateWaitProfileGoal    State = "wait_profile_goal"
	StateWaitStepsCount     State = "wait_steps_count"
)

type StateSetter interface {
	Set(chatID int64, st State)
	Clear(chatID int64)
}

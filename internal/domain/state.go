package domain

type State string

const (
	StateNone         State = ""
	StateWaitMealText State = "wait_meal_text"
	StateWaitPlanText State = "wait_plan_text"
)

type StateSetter interface {
	Set(chatID int64, st State)
	Clear(chatID int64)
}

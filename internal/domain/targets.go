package domain

type Targets struct {
	ChatID   int64
	Kcal     int
	ProteinG int
	FatG     int
	CarbsG   int
	Source   string // "calc"|"manual"
}

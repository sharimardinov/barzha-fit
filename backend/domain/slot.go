package domain

type Slot struct {
	Name            string
	RequiredGroup   []string
	RequiredPattern []string
	PreferPriority  string
	PreferCompound  bool
}

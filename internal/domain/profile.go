package domain

type Profile struct {
	ChatID     int64
	Sex        string // "m"|"f"|"" (unknown)
	Age        int
	HeightCM   int
	WeightKG   float64
	BodyFatPct float64
	Activity   string // "low"|"mid"|"high"
}

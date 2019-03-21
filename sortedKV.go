package helper

type StrKIntV struct {
	Key   string
	Value int64
}

type SKIVSlice []*StrKIntV

func (s SKIVSlice) Len() int {
	return len(s)
}

func (s SKIVSlice) Less(i, j int) bool {
	return s[i].Value <= s[j].Value
}

func (s SKIVSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

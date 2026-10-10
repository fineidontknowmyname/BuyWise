package provenance

type Source string

const (
	Stub       Source = "stub"
	Fixture    Source = "fixture"
	Manual     Source = "manual"
	Calculated Source = "calculated"
)

func (s Source) IsReal() bool {
	return s == Manual || s == Calculated
}

type Labelled struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Source Source `json:"provenance"`
}

func AllReal(values []Labelled) bool {
	for _, v := range values {
		if !v.Source.IsReal() {
			return false
		}
	}
	return true
}

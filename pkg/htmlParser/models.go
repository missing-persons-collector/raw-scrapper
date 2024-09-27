package htmlParser

type RawPerson struct {
	Name             string
	LastName         string
	MaidenName       string
	Gender           string
	DOB              string
	POB              string
	Citizenship      string
	PrimaryAddress   string
	SecondaryAddress string
	Country          string
	ImageURL         string

	Height   string
	Hair     string
	EyeColor string
	Weight   string

	// Date of disappearance
	DOD string
	// place of disappearance
	POD string

	Description string
}

func NewRawPerson() RawPerson {
	return RawPerson{}
}

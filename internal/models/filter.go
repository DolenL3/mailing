package models

// Filter is a filter to be applied when searching for client's.
type Filter struct {
	PhoneOperator int
	Tag           string
	Timezone      string
}

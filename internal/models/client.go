package models

// Client is a struct that represents client.
type Client struct {
	ID            int64  `db:"id"`
	PhoneNumber   int64  `db:"phone_number"`
	PhoneOperator int    `db:"phone_operator"`
	Tag           string `db:"tag"`
	Timezone      string `db:"timezone"`
}

// ClientUpdate is a struct with updates which should be applied to client.
type ClientUpdate struct {
	PhoneNumber   int64
	PhoneOperator int
	Tag           string
	Timezone      string
}

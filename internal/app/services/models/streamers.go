package models

type Streamers struct {
	ID       int    `db:"id"`
	Platform string `db:"platform"`
	Username string `db:"username"`
	Quality  string `db:"quality"`
}

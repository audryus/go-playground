package main

import (
	"myplayground/ddd/users"
)

func main() {
	u := users.GetByID("teste")

	u.Save()

}

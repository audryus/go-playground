package users

import "fmt"

type usersRepo struct {
}

var repo *usersRepo

func init() {
	repo = &usersRepo{}
}

func (u *User) Save() {
	fmt.Printf("Saving %+v\n\n", u)
}

func GetByID(ID string) *User {
	u := &User{
		ID:   ID,
		Name: "User Name",
	}
	fmt.Printf("Getting by ID %s\n\n", u)
	return u
}

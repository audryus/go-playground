package main

import (
	"context"
	"fmt"
	"log"

	supa "github.com/nedpals/supabase-go"
)

const (
	supabaseUrl     = "SUPABASE_URL"
	supabaseAnonKey = "KEY"
)

func main() {
	supabase := supa.CreateClient(supabaseUrl, supabaseAnonKey)
	usr, err := signIn(supabase)
	if err != nil {
		log.Fatal(err)
	}

	user, err := verifyTokenSupabase(supabase, usr.AccessToken)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(usr.AccessToken)

	fmt.Printf("logado : \n%+v\n", user)

	mapz := make(map[string]interface{})
	mapz["a"] = "b"

	user, err = supabase.Auth.UpdateUser(context.Background(), usr.AccessToken, mapz)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("novo: \n%+v\n", user)
}

func signIn(supabase *supa.Client) (*supa.AuthenticatedDetails, error) {
	ctx := context.Background()
	user, err := supabase.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    "mail@example.com",
		Password: "password",
	})
	if err != nil {
		return nil, err
	}

	//_, b := supabase.Auth.InviteUserByEmail(ctx, "mail@example.com")
	//if b != nil {
	//	return nil, b
	//}

	return user, err
}

func verifyTokenSupabase(supabase *supa.Client, token string) (*supa.User, error) {
	ctx := context.Background()
	user, err := supabase.Auth.User(ctx, token)

	if err != nil {
		fmt.Println("erro ao consultar user ..")
		log.Fatal(err)
		return user, err
	}

	return user, nil

}

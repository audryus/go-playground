package main

import (
	"context"
	"fmt"
	"log"

	supa "github.com/nedpals/supabase-go"
)

const (
	supabaseUrl     = "http://supabase.localdev.arpa"
	supabaseAnonKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.ewogICJyb2xlIjogImFub24iLAogICJpc3MiOiAic3VwYWJhc2UiLAogICJpYXQiOiAxNjk3OTQzNjAwLAogICJleHAiOiAxODU1Nzk2NDAwCn0.d7sRUH9wJKF4XuJv8h9S0XlooVMWTEo3lV04hLDLzFc"
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

	fmt.Printf("\n%+v\n", user)

	mapz := make(map[string]interface{})
	mapz["a"] = "b"

	user, err = supabase.Auth.UpdateUser(context.Background(), usr.AccessToken, mapz)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n%+v\n", user)
}

func signIn(supabase *supa.Client) (*supa.AuthenticatedDetails, error) {
	ctx := context.Background()
	user, err := supabase.Auth.SignIn(ctx, supa.UserCredentials{
		Email:    "em@il.com",
		Password: "password",
	})
	if err != nil {
		return nil, err
	}

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

package main

import "fmt"

type Db struct{}

func Open(d *Db, conn string) {
}

type Entity struct {
	ID string
}

func (e *Entity) Create() {
	fmt.Println("saving ...")
}

func (e *Entity) Update() {
	fmt.Println("updating ...")
}

type Turma struct {
	Entity
}

func (t *Turma) Template() {
	t.Create()
	fmt.Println("template ...")
	t.Update()
}

func main() {
	t := &Turma{
		Entity: Entity{
			ID: "turma",
		},
	}

	t.Create()
	t.Update()

	t.Template()

	fmt.Println(t.ID)
}

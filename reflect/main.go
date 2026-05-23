package main

import (
	"fmt"
	"reflect"
)

type Component interface{}

type Position struct {
	x float64
	y float64
}

type Name struct {
	value string
}

type Velocity struct {
	value float64
}

type Entity []Component

func main() {
	components := make(map[reflect.Type][]Component)

	components[reflect.TypeOf(Name{})] = []Component{
		Name{value: "Alice"},
		Name{value: "Bob"},
		Name{value: "Charlie"},
	}
	components[reflect.TypeOf(Position{})] = []Component{
		Position{
			x: 1,
			y: 1,
		},
		Position{
			x: 2,
			y: 2,
		},
	}

	_ = []Entity{
		{
			&Position{
				x: 1,
				y: 1,
			},
			&Name{value: "Alice"},
		},
		{
			&Position{
				x: 2,
				y: 2,
			},
			&Name{value: "Bob"},
		},
		{
			&Name{value: "Charlie"},
		},
	}

	//system1(e)
	//for _, c := range e {
	// 	fmt.Printf("%+v ", c)
	//}
	// fmt.Println("---")
	t := reflect.TypeOf(fnc1)
	// fmt.Printf("Function has %d arguments:\n", t.NumIn())
	var types []reflect.Type
	for i := range t.NumIn() {
		funcArgType := t.In(i)
		// fmt.Printf("Arg %d: %v\n", i, funcArgType)
		if funcArgType.Kind() == reflect.Slice {
			// fmt.Println("Element type:", funcArgType.Elem()) // *main.Position
			types = append(types, funcArgType.Elem())
		} else {
			types = append(types, t.In(i))
		}
	}

	// Call fnc1 using reflect
	fn := reflect.ValueOf(fnc1)
	var args []reflect.Value

	for _, t := range types {
		fmt.Printf("%v - %v\n", t, components[t])
		args = append(args, ToTypedSliceReflect(t, components[t]))
	}

	fn.Call(args)

	fmt.Println("")
}

func ToTypedSliceReflect(sliceType reflect.Type, in []Component) reflect.Value {
	out := reflect.MakeSlice(reflect.SliceOf(sliceType), 0, len(in))
	for _, c := range in {
		// Only add if the type matches or can be assigned
		val := reflect.ValueOf(c)
		if val.Type().AssignableTo(sliceType) {
			out = reflect.Append(out, val)
		}
	}
	return out
}

func fnc1(v []Velocity, p []Position, n []Name) {
	fmt.Printf("1) %v = %v\n", p, n)
}

func fnc2(n *Name) {
	fmt.Printf("2) %s", n.value)
}

func system1(componentes []Component) {
	// contains := containsAll(componentes,
	// 	reflect.TypeOf(&Position{}),
	// 	reflect.TypeOf(&Name{}))
	// for _, c := range componentes {
	// 	fmt.Printf("%+v ", c)
	// }
	// fmt.Printf("%v ", contains)
}

func system2(ps []*Position, nms []*Name) {
}

func containsAll(components []Component, types []reflect.Type) bool {
	found := make(map[reflect.Type]bool)
	for _, c := range components {
		t := reflect.TypeOf(c)
		found[t] = true
	}
	for _, typ := range types {
		if !found[typ] {
			return false
		}
	}
	return true
}

type Query struct {
	q func(componentes ...Component) bool
}

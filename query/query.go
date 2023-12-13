package main

type query struct {
	sql string
}

type predicate func() bool

type lero func(sql string, condition predicate)

func (q *query) add(sql string, condition predicate) *query {
	return q
}

func NewQuery(sql string) *query {
	return &query{
		sql: sql,
	}
}

func main() {
	sql := NewQuery("select * from table_a a inner join table_b b ON a.id = b.a_id")
	name := ""
	sql.add("b.name = ?", func() bool {
		return len(name) > 0
	})
}

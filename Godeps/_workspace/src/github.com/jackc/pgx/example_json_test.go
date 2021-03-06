package pgx_test

import (
	"fmt"
	"github.com/flynn/flynn/Godeps/_workspace/src/github.com/jackc/pgx"
)

func Example_JSON() {
	conn, err := pgx.Connect(*defaultConnConfig)
	if err != nil {
		fmt.Printf("Unable to establish connection: %v", err)
		return
	}

	if _, ok := conn.PgTypes[pgx.JsonOid]; !ok {
		// No JSON type -- must be running against very old PostgreSQL
		// Pretend it works
		fmt.Println("John", 42)
		return
	}

	type person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	input := person{
		Name: "John",
		Age:  42,
	}

	var output person

	err = conn.QueryRow("select $1::json", input).Scan(&output)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(output.Name, output.Age)
	// Output:
	// John 42
}

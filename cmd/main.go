package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	memsql "github.com/twaaaadahardeep/mem-sql"
)

func runRepl(mb *memsql.MemoryBackend, reader *bufio.Reader) {
	for {
		fmt.Print("# ")
		text, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		text = strings.Replace(text, "\n", "", -1)

		ast, err := memsql.Parse(text)
		if err != nil {
			panic(err)
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case memsql.CreateTableKind:
				err := mb.CreateTable(ast.Statements[0].CreateTableStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("OK")

			case memsql.InsertKind:
				err := mb.Insert(stmt.InsertStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("OK")

			case memsql.SelectKind:
				res, err := mb.Select(stmt.SelectStatement)
				if err != nil {
					panic(err)
				}

				for _, col := range res.Columns {
					fmt.Printf("| %s", col.Name)
				}
				fmt.Println("|")

				for i := 0; i < 20; i++ {
					fmt.Print("==")
				}
				fmt.Println()

				for _, r := range res.Rows {
					fmt.Printf("|")

					for i, cell := range r {
						typ := res.Columns[i].Type
						s := ""

						switch typ {
						case memsql.IntType:
							s = fmt.Sprintf("%d", cell.AsInt32())
						case memsql.TextType:
							s = cell.AsText()
						}

						fmt.Printf("%s | ", s)
					}

					fmt.Println()
				}

				fmt.Print("OK")
			}
		}
	}
}

func main() {
	m := memsql.NewMemoryBackend()

	r := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to mem-sql")

	runRepl(m, r)
}

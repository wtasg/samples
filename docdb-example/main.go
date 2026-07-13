// main.go — DocDB REPL entry point.
//
// Usage: go run . [data-dir]
//
// The REPL reads commands (terminated by ';') and prints pretty JSON results.
// Data is persisted in <data-dir>/ (default: ./data).
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"docdb/internal/engine"
	"docdb/internal/parser"
)

const banner = `
╔══════════════════════════════════════════════╗
║          DocDB — Toy NoSQL DB in Go          ║
║   LSMTree · RobinHood · SkipList · Inverted  ║
╚══════════════════════════════════════════════╝
Type NoSQL commands ending with ;   (\q to quit)
`

func main() {
	dataDir := "data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	ex, err := engine.NewExecutor(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer ex.Close()

	fmt.Print(banner)

	scanner := bufio.NewScanner(os.Stdin)
	var buf strings.Builder

	for {
		if buf.Len() == 0 {
			fmt.Print("docdb> ")
		} else {
			fmt.Print("     > ")
		}

		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())

		if line == `\q` || line == "exit" || line == "quit" {
			fmt.Println("Bye!")
			break
		}
		if line == "" {
			continue
		}

		buf.WriteString(" ")
		buf.WriteString(line)

		// Execute once we see a semicolon.
		if !strings.Contains(line, ";") {
			continue
		}

		cmd := strings.TrimSpace(buf.String())
		buf.Reset()

		stmt, err := parser.Parse(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
			continue
		}
		if stmt == nil {
			continue
		}

		result, err := ex.Execute(stmt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		printResult(result)
	}
}

// printResult renders a Result as pretty-printed JSON.
func printResult(r *engine.Result) {
	if r == nil {
		return
	}
	if r.Message != "" {
		fmt.Println(r.Message)
	}
	if len(r.Docs) == 0 {
		if r.Message == "" {
			fmt.Println("(0 documents)")
		}
		return
	}

	for _, doc := range r.Docs {
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			fmt.Printf("%+v\n", doc)
		} else {
			fmt.Println(string(data))
		}
	}

	n := len(r.Docs)
	switch n {
	case 1:
		fmt.Println("(1 document)")
	default:
		fmt.Printf("(%d documents)\n", n)
	}
}

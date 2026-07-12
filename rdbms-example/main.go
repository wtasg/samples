// main.go — ToyDB REPL entry point.
//
// Usage: go run . [data-dir]
//
// The REPL reads SQL statements (terminated by ';') and prints tabular results.
// Data is persisted in <data-dir>/ (default: ./data).
package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"rdbms/internal/engine"
	"rdbms/internal/parser"
)

const banner = `
╔══════════════════════════════════════════════╗
║            ToyDB — Toy RDBMS in Go           ║
║  B+Tree · RedBlack · Trie · Bloom · R-Karp   ║
╚══════════════════════════════════════════════╝
Type SQL statements ending with ;   (\q to quit)
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
			fmt.Print("toydb> ")
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

		sql := strings.TrimSpace(buf.String())
		buf.Reset()

		stmt, err := parser.Parse(sql)
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

// printResult renders a Result as a pretty ASCII table.
func printResult(r *engine.Result) {
	if r == nil {
		return
	}
	if r.Message != "" {
		fmt.Println(r.Message)
	}
	if len(r.Columns) == 0 {
		return
	}

	// Calculate column widths.
	widths := make([]int, len(r.Columns))
	for i, col := range r.Columns {
		widths[i] = utf8.RuneCountInString(col)
	}
	for _, row := range r.Rows {
		for i, col := range r.Columns {
			s := formatVal(row[col])
			if w := utf8.RuneCountInString(s); w > widths[i] {
				widths[i] = w
			}
		}
	}

	sep := buildSep(widths)
	fmt.Println(sep)

	// Header.
	fmt.Print("│")
	for i, col := range r.Columns {
		fmt.Printf(" %-*s │", widths[i], col)
	}
	fmt.Println()
	fmt.Println(sep)

	// Rows.
	for _, row := range r.Rows {
		fmt.Print("│")
		for i, col := range r.Columns {
			s := formatVal(row[col])
			fmt.Printf(" %-*s │", widths[i], s)
		}
		fmt.Println()
	}
	fmt.Println(sep)

	n := len(r.Rows)
	switch n {
	case 0:
		fmt.Println("(0 rows)")
	case 1:
		fmt.Println("(1 row)")
	default:
		fmt.Printf("(%d rows)\n", n)
	}
}

func buildSep(widths []int) string {
	var sb strings.Builder
	sb.WriteRune('┼')
	for _, w := range widths {
		sb.WriteString(strings.Repeat("─", w+2))
		sb.WriteRune('┼')
	}
	// Replace outer ┼ with ├/┤ and ─ with the box chars.
	s := sb.String()
	s = "├" + s[len("┼"):len(s)-len("┼")] + "┤"
	return s
}

func formatVal(v any) string {
	if v == nil {
		return "NULL"
	}
	switch v := v.(type) {
	case float64:
		if v == math.Trunc(v) && math.Abs(v) < 1e15 {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

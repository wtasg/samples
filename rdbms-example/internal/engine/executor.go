// executor.go — Query executor: walks the AST and calls table operations.
//
// Data-structure dispatch summary:
//
//	WHERE pk = X           → Bloom Filter → B+ Tree → Pager           (point)
//	WHERE pk BETWEEN lo hi → B+ Tree range scan → Pager               (range)
//	WHERE col LIKE 'pre%'  → Trie prefix search → Pager               (prefix)
//	WHERE col LIKE '%sub%' → Rabin-Karp full scan                      (substr)
//	WHERE col LIKE '%sfx'  → Rabin-Karp suffix scan                    (suffix)
//	WHERE col OP val       → full scan with typed predicate            (general)
//	ORDER BY col           → Red-Black Tree in-order traversal         (sort)
package engine

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"rdbms/internal/catalog"
	"rdbms/internal/parser"
	"rdbms/internal/storage"
)

// Executor coordinates the catalog, open tables, and query execution.
type Executor struct {
	cat     *catalog.Catalog
	tables  map[string]*Table
	dataDir string
}

// NewExecutor creates an executor rooted at dataDir.
func NewExecutor(dataDir string) (*Executor, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	cat, err := catalog.Load(dataDir)
	if err != nil {
		return nil, err
	}
	ex := &Executor{
		cat:     cat,
		tables:  make(map[string]*Table),
		dataDir: dataDir,
	}
	// Pre-open all known tables.
	for _, name := range cat.Tables() {
		schema, _ := cat.Get(name)
		tbl, err := openTable(schema, dataDir)
		if err != nil {
			return nil, fmt.Errorf("open table %q: %w", name, err)
		}
		ex.tables[name] = tbl
	}
	return ex, nil
}

// Close flushes all open tables.
func (ex *Executor) Close() {
	for _, tbl := range ex.tables {
		tbl.close()
	}
}

// TableNames returns the names of all open tables (used by gRPC ListTables).
func (ex *Executor) TableNames() []string {
	return ex.cat.Tables()
}

// TableSchema returns the schema of the named table (used by gRPC DescribeTable).
func (ex *Executor) TableSchema(name string) (*catalog.TableSchema, error) {
	return ex.cat.Get(name)
}

// Result is the output of a query.
type Result struct {
	Columns []string
	Rows    []Row
	Message string
}

// Execute runs a parsed Statement and returns a Result.
func (ex *Executor) Execute(stmt parser.Statement) (*Result, error) {
	switch s := stmt.(type) {
	case *parser.CreateTableStmt:
		return ex.execCreate(s)
	case *parser.InsertStmt:
		return ex.execInsert(s)
	case *parser.SelectStmt:
		return ex.execSelect(s)
	case *parser.UpdateStmt:
		return ex.execUpdate(s)
	case *parser.DeleteStmt:
		return ex.execDelete(s)
	case *parser.DropTableStmt:
		return ex.execDrop(s)
	}
	return nil, fmt.Errorf("unsupported statement type")
}

// ── CREATE TABLE ─────────────────────────────────────────────────────────────

func (ex *Executor) execCreate(s *parser.CreateTableStmt) (*Result, error) {
	cols := make([]catalog.Column, len(s.Columns))
	for i, cd := range s.Columns {
		ct, err := catalog.ParseColType(cd.Type)
		if err != nil {
			return nil, err
		}
		cols[i] = catalog.Column{Name: cd.Name, Type: ct}
	}
	schema := &catalog.TableSchema{Name: s.Table, Columns: cols}
	if err := ex.cat.CreateTable(schema); err != nil {
		return nil, err
	}

	// Ensure the data file exists.
	path := filepath.Join(ex.dataDir, s.Table+".rows")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()

	tbl, err := openTable(schema, ex.dataDir)
	if err != nil {
		return nil, err
	}
	ex.tables[s.Table] = tbl
	return &Result{Message: fmt.Sprintf("Table %q created.", s.Table)}, nil
}

// ── INSERT ───────────────────────────────────────────────────────────────────

func (ex *Executor) execInsert(s *parser.InsertStmt) (*Result, error) {
	tbl, err := ex.getTable(s.Table)
	if err != nil {
		return nil, err
	}
	if err := tbl.Insert(s.Values); err != nil {
		return nil, err
	}
	return &Result{Message: "1 row inserted."}, nil
}

// ── SELECT ───────────────────────────────────────────────────────────────────

func (ex *Executor) execSelect(s *parser.SelectStmt) (*Result, error) {
	tbl, err := ex.getTable(s.Table)
	if err != nil {
		return nil, err
	}
	schema := tbl.schema

	var rows []Row

	if s.Where == nil {
		rows, err = tbl.Scan(nil)
	} else {
		rows, err = ex.applyWhere(tbl, s.Where)
	}
	if err != nil {
		return nil, err
	}

	// ORDER BY
	if s.OrderBy != "" {
		colIdx := schema.ColumnIndex(s.OrderBy)
		if colIdx < 0 {
			return nil, fmt.Errorf("unknown column %q in ORDER BY", s.OrderBy)
		}
		col := schema.Columns[colIdx]
		if col.Type == catalog.ColInt || col.Type == catalog.ColFloat {
			rows = OrderByInt(rows, s.OrderBy, s.Desc)
		} else {
			// Text ORDER BY: sort by string value using a simple insertion sort
			// (for a toy DB; production would use merge sort or a B-Tree index).
			rows = orderByString(rows, s.OrderBy, s.Desc)
		}
	}

	// Project columns.
	colNames := s.Columns
	if len(colNames) == 0 {
		colNames = make([]string, len(schema.Columns))
		for i, c := range schema.Columns {
			colNames[i] = c.Name
		}
	}

	projected := make([]Row, len(rows))
	for i, row := range rows {
		proj := make(Row, len(colNames))
		for _, col := range colNames {
			proj[col] = row[col]
		}
		projected[i] = proj
	}

	return &Result{Columns: colNames, Rows: projected}, nil
}

// applyWhere dispatches to the optimal data-structure path for the WHERE clause.
func (ex *Executor) applyWhere(tbl *Table, w *parser.WhereExpr) ([]Row, error) {
	schema := tbl.schema
	pkCol := schema.PKColumn()

	switch w.Op {
	case "BETWEEN":
		lo, err := strconv.ParseInt(w.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("BETWEEN lo: %w", err)
		}
		hi, err := strconv.ParseInt(w.Value2, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("BETWEEN hi: %w", err)
		}
		if w.Column == pkCol.Name {
			// ② B+ Tree range scan.
			return tbl.RangeScan(lo, hi)
		}
		// Non-PK BETWEEN: full scan with numeric predicate.
		return tbl.Scan(numericPred(w.Column, "BETWEEN", float64(lo), float64(hi)))

	case "LIKE":
		pattern := w.Value
		if strings.HasSuffix(pattern, "%") && !strings.HasPrefix(pattern, "%") {
			// 'prefix%' → Trie
			prefix := strings.TrimSuffix(pattern, "%")
			return tbl.PrefixScan(w.Column, prefix)
		}
		if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
			// '%substr%' → Rabin-Karp
			substr := strings.Trim(pattern, "%")
			return tbl.SubstringScan(w.Column, substr)
		}
		if strings.HasPrefix(pattern, "%") {
			// '%suffix' → Rabin-Karp suffix
			suffix := strings.TrimPrefix(pattern, "%")
			return tbl.SuffixScan(w.Column, suffix)
		}
		// Exact string match via Trie.
		return tbl.PrefixScan(w.Column, pattern)

	case "=":
		// PK equality: Bloom Filter → B+ Tree → Pager
		if w.Column == pkCol.Name {
			pk, err := strconv.ParseInt(w.Value, 10, 64)
			if err != nil {
				return nil, err
			}
			row, err := tbl.GetByPK(pk)
			if err != nil {
				return []Row{}, nil // not found → empty
			}
			return []Row{row}, nil
		}
		// Non-PK equality: full scan.
		return tbl.Scan(equalityPred(w.Column, w.Value, schema))

	default:
		return tbl.Scan(comparisonPred(w.Column, w.Op, w.Value, schema))
	}
}

// ── UPDATE ───────────────────────────────────────────────────────────────────

func (ex *Executor) execUpdate(s *parser.UpdateStmt) (*Result, error) {
	tbl, err := ex.getTable(s.Table)
	if err != nil {
		return nil, err
	}

	var pred func(Row) bool
	if s.Where != nil {
		pred = buildPred(s.Where, tbl.schema)
	}

	n, err := tbl.Update(pred, s.Assignments, tbl.schema)
	if err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("%d row(s) updated.", n)}, nil
}

// ── DELETE ───────────────────────────────────────────────────────────────────

func (ex *Executor) execDelete(s *parser.DeleteStmt) (*Result, error) {
	tbl, err := ex.getTable(s.Table)
	if err != nil {
		return nil, err
	}

	var pred func(Row) bool
	if s.Where != nil {
		pred = buildPred(s.Where, tbl.schema)
	}

	n, err := tbl.Delete(pred)
	if err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("%d row(s) deleted.", n)}, nil
}

// ── DROP TABLE ───────────────────────────────────────────────────────────────

func (ex *Executor) execDrop(s *parser.DropTableStmt) (*Result, error) {
	tbl, ok := ex.tables[s.Table]
	if !ok {
		return nil, fmt.Errorf("table %q does not exist", s.Table)
	}
	tbl.close()
	delete(ex.tables, s.Table)

	if err := ex.cat.DropTable(s.Table); err != nil {
		return nil, err
	}
	if err := storage.Drop(ex.dataDir, s.Table); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("Table %q dropped.", s.Table)}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (ex *Executor) getTable(name string) (*Table, error) {
	tbl, ok := ex.tables[name]
	if !ok {
		return nil, fmt.Errorf("table %q does not exist", name)
	}
	return tbl, nil
}

// buildPred constructs a generic row predicate from a WhereExpr.
func buildPred(w *parser.WhereExpr, schema *catalog.TableSchema) func(Row) bool {
	switch w.Op {
	case "LIKE":
		return likePred(w.Column, w.Value)
	case "BETWEEN":
		lo, _ := strconv.ParseFloat(w.Value, 64)
		hi, _ := strconv.ParseFloat(w.Value2, 64)
		return numericPred(w.Column, "BETWEEN", lo, hi)
	case "=":
		return equalityPred(w.Column, w.Value, schema)
	default:
		return comparisonPred(w.Column, w.Op, w.Value, schema)
	}
}

// likePred returns a predicate for LIKE pattern matching.
func likePred(col, pattern string) func(Row) bool {
	return func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		s, ok := v.(string)
		if !ok {
			return false
		}
		switch {
		case strings.HasSuffix(pattern, "%") && !strings.HasPrefix(pattern, "%"):
			return strings.HasPrefix(s, strings.TrimSuffix(pattern, "%"))
		case strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%"):
			substr := strings.Trim(pattern, "%")
			return strings.Contains(s, substr)
		case strings.HasPrefix(pattern, "%"):
			return strings.HasSuffix(s, strings.TrimPrefix(pattern, "%"))
		default:
			return s == pattern
		}
	}
}

// equalityPred returns a predicate for col = val with type coercion.
func equalityPred(col, val string, schema *catalog.TableSchema) func(Row) bool {
	idx := schema.ColumnIndex(col)
	if idx < 0 {
		return func(Row) bool { return false }
	}
	colType := schema.Columns[idx].Type
	return func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		coerced, err := coerceValue(val, colType)
		if err != nil {
			return false
		}
		return fmt.Sprintf("%v", v) == fmt.Sprintf("%v", coerced)
	}
}

// comparisonPred returns a predicate for col OP val (numeric comparison).
func comparisonPred(col, op, val string, schema *catalog.TableSchema) func(Row) bool {
	target, err := strconv.ParseFloat(val, 64)
	if err != nil {
		// Fall back to string comparison.
		return func(row Row) bool {
			v, ok := row[col]
			if !ok {
				return false
			}
			s := fmt.Sprintf("%v", v)
			switch op {
			case "<":
				return s < val
			case ">":
				return s > val
			case "<=":
				return s <= val
			case ">=":
				return s >= val
			case "!=":
				return s != val
			}
			return false
		}
	}
	return func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		var f float64
		switch v := v.(type) {
		case float64:
			f = v
		case int64:
			f = float64(v)
		default:
			return false
		}
		switch op {
		case "<":
			return f < target
		case ">":
			return f > target
		case "<=":
			return f <= target
		case ">=":
			return f >= target
		case "!=":
			return math.Abs(f-target) > 1e-12
		}
		return false
	}
}

// numericPred handles BETWEEN and numeric operators on non-PK columns.
func numericPred(col, op string, lo, hi float64) func(Row) bool {
	return func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		var f float64
		switch v := v.(type) {
		case float64:
			f = v
		case int64:
			f = float64(v)
		default:
			return false
		}
		if op == "BETWEEN" {
			return f >= lo && f <= hi
		}
		return false
	}
}

// orderByString sorts rows by a TEXT column value using insertion sort.
// (For a toy DB; a production system would use the Trie-based sorted set.)
func orderByString(rows []Row, col string, desc bool) []Row {
	sorted := make([]Row, len(rows))
	copy(sorted, rows)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		ks, _ := key[col].(string)
		j := i - 1
		for j >= 0 {
			vs, _ := sorted[j][col].(string)
			cond := vs > ks
			if desc {
				cond = vs < ks
			}
			if !cond {
				break
			}
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}
	return sorted
}

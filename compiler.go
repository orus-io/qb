package qb

import (
	"fmt"
	"strings"
)

// NewCompilerContext initialize a new compiler context
// During its lifetime, the dialect is expected to be dedicated
// to it, and Reset() afterward.
// The Reset() is the responsability of the caller
func NewCompilerContext(dialect Dialect) *CompilerContext {
	return &CompilerContext{
		Dialect:  dialect,
		Compiler: dialect.GetCompiler(),
		Vars:     make(map[string]interface{}),
		Binds:    []interface{}{},
	}
}

// CompilerContext is a data structure passed to all the Compiler visit
// functions. It contains the bindings, links to the Dialect and Compiler
// being used, and some contextual informations that can be used by the
// compiler functions to communicate during the compilation.
type CompilerContext struct {
	Binds            []interface{}
	DefaultTableName string
	InSubQuery       bool
	Vars             map[string]interface{}

	Dialect  Dialect
	Compiler Compiler
}

// Compiler is a visitor that produce SQL from various types of Clause
type Compiler interface {
	VisitAggregate(*CompilerContext, AggregateClause) string
	VisitAlias(*CompilerContext, AliasClause) string
	VisitBinary(*CompilerContext, BinaryExpressionClause) string
	VisitBind(*CompilerContext, BindClause) string
	VisitColumn(*CompilerContext, ColumnElem) string
	VisitCombiner(*CompilerContext, CombinerClause) string
	VisitDelete(*CompilerContext, DeleteStmt) string
	VisitExists(*CompilerContext, ExistsClause) string
	VisitHaving(*CompilerContext, HavingClause) string
	VisitInsert(*CompilerContext, InsertStmt) string
	VisitJoin(*CompilerContext, JoinClause) string
	VisitLabel(*CompilerContext, string) string
	VisitList(*CompilerContext, ListClause) string
	VisitOrderBy(*CompilerContext, OrderByClause) string
	VisitSelect(*CompilerContext, SelectStmt) string
	VisitTable(*CompilerContext, TableElem) string
	VisitText(*CompilerContext, TextClause) string
	VisitUpdate(*CompilerContext, UpdateStmt) string
	VisitUpsert(*CompilerContext, UpsertStmt) string
	VisitWhere(*CompilerContext, WhereClause) string
}

// SQLCompiler aims to provide a SQL ANSI-92 implementation of Compiler
type SQLCompiler struct {
	Dialect Dialect
}

// VisitAggregate compiles aggregate functions (COUNT, SUM...)
func (c SQLCompiler) VisitAggregate(context *CompilerContext, aggregate AggregateClause) string {
	return fmt.Sprintf("%s(%s)", aggregate.fn, aggregate.clause.Accept(context))
}

// VisitAlias compiles a '<selectable> AS <aliasname>' SQL clause
func (SQLCompiler) VisitAlias(context *CompilerContext, alias AliasClause) string {
	return fmt.Sprintf(
		"%s AS %s",
		alias.Selectable.Accept(context),
		context.Dialect.Escape(alias.Name),
	)
}

// VisitBinary compiles LEFT <op> RIGHT expressions
func (c SQLCompiler) VisitBinary(context *CompilerContext, binary BinaryExpressionClause) string {
	return fmt.Sprintf(
		"%s %s %s",
		binary.Left.Accept(context),
		binary.Op,
		binary.Right.Accept(context),
	)
}

// VisitBind renders a bounded value
func (SQLCompiler) VisitBind(context *CompilerContext, bind BindClause) string {
	context.Binds = append(context.Binds, bind.Value)
	return context.Dialect.Placeholder()
}

// VisitColumn returns a column name, optionnaly escaped depending on the dialect
// configuration
func (c SQLCompiler) VisitColumn(context *CompilerContext, column ColumnElem) string {
	sql := ""
	if context.InSubQuery || context.DefaultTableName != column.Table {
		sql += c.Dialect.Escape(column.Table) + "."
	}
	sql += c.Dialect.Escape(column.Name)
	return sql
}

// VisitCombiner compiles AND and OR sql clauses
func (c SQLCompiler) VisitCombiner(context *CompilerContext, combiner CombinerClause) string {
	sqls := []string{}
	for _, c := range combiner.clauses {
		sql := c.Accept(context)
		sqls = append(sqls, sql)
	}

	return fmt.Sprintf("(%s)", strings.Join(sqls, fmt.Sprintf(" %s ", combiner.operator)))
}

// VisitDelete compiles a DELETE statement
func (c SQLCompiler) VisitDelete(context *CompilerContext, delete DeleteStmt) string {
	sql := "DELETE FROM " + delete.table.Accept(context)

	if delete.where != nil {
		sql += "\n" + delete.where.Accept(context)
	}

	returning := []string{}
	for _, c := range delete.returning {
		returning = append(returning, context.Dialect.Escape(c.Name))
	}

	if len(returning) > 0 {
		sql += "\nRETURNING " + strings.Join(returning, ", ")
	}

	return sql
}

// VisitExists compile a EXISTS clause
func (SQLCompiler) VisitExists(context *CompilerContext, exists ExistsClause) string {
	var sql string
	if exists.Not {
		sql = "NOT "
	}
	sql += "EXISTS(%s)"
	context.InSubQuery = true
	defer func() { context.InSubQuery = false }()
	return fmt.Sprintf(sql, exists.Select.Accept(context))
}

// VisitHaving compiles a HAVING clause
func (c SQLCompiler) VisitHaving(context *CompilerContext, having HavingClause) string {
	aggSQL := having.aggregate.Accept(context)
	context.Binds = append(context.Binds, having.value)
	return fmt.Sprintf("HAVING %s %s %s", aggSQL, having.op, context.Dialect.Placeholder())
}

// VisitInsert compiles a INSERT statement
func (c SQLCompiler) VisitInsert(context *CompilerContext, insert InsertStmt) string {
	var (
		colNames     []string
		placeholders []string
	)

	context.DefaultTableName = insert.table.Name
	defer func() { context.DefaultTableName = "" }()

	for k, v := range insert.values {
		colNames = append(colNames, context.Compiler.VisitLabel(context, k))
		placeholders = append(placeholders, context.Dialect.Placeholder())
		context.Binds = append(context.Binds, v)
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s(%s)\nVALUES(%s)",
		insert.table.Accept(context),
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)

	returning := []string{}
	for _, r := range insert.returning {
		returning = append(returning, r.Accept(context))
	}
	if len(insert.returning) > 0 {
		sql += fmt.Sprintf(
			"\nRETURNING %s",
			strings.Join(returning, ", "),
		)
	}

	return sql
}

// VisitJoin compiles a JOIN (ON) clause
func (c SQLCompiler) VisitJoin(context *CompilerContext, join JoinClause) string {
	sql := fmt.Sprintf(
		"%s\n%s %s",
		join.Left.Accept(context),
		join.JoinType,
		join.Right.Accept(context),
	)
	if join.OnClause != nil {
		sql += " ON " + join.OnClause.Accept(context)
	}

	return sql
}

// VisitLabel returns a single label, optionally escaped
func (c SQLCompiler) VisitLabel(context *CompilerContext, label string) string {
	return c.Dialect.Escape(label)
}

// VisitList compiles a list of values
func (c SQLCompiler) VisitList(context *CompilerContext, list ListClause) string {
	var clauses []string
	for _, clause := range list.Clauses {
		clauses = append(clauses, clause.Accept(context))
	}
	return fmt.Sprintf("(%s)", strings.Join(clauses, ", "))
}

// VisitOrderBy compiles a ORDER BY sql clause
func (c SQLCompiler) VisitOrderBy(context *CompilerContext, orderBy OrderByClause) string {
	cols := []string{}
	for _, c := range orderBy.columns {
		cols = append(cols, c.Accept(context))
	}

	return fmt.Sprintf("ORDER BY %s %s", strings.Join(cols, ", "), orderBy.t)
}

// VisitSelect compiles a SELECT statement
func (c SQLCompiler) VisitSelect(context *CompilerContext, selectStmt SelectStmt) string {
	lines := []string{}
	addLine := func(s string) {
		lines = append(lines, s)
	}
	if !context.InSubQuery {
		context.DefaultTableName = selectStmt.from.DefaultName()
	}

	// select
	columns := []string{}
	for _, c := range selectStmt.sel {
		sql := c.Accept(context)
		columns = append(columns, sql)
	}
	addLine(fmt.Sprintf("SELECT %s", strings.Join(columns, ", ")))

	// from
	addLine(fmt.Sprintf("FROM %s", selectStmt.from.Accept(context)))

	// where
	if selectStmt.where != nil {
		addLine(selectStmt.where.Accept(context))
	}

	// group by
	groupByCols := []string{}
	for _, c := range selectStmt.groupBy {
		groupByCols = append(groupByCols, context.Dialect.Escape(c.Name))
	}
	if len(groupByCols) > 0 {
		addLine(fmt.Sprintf("GROUP BY %s", strings.Join(groupByCols, ", ")))
	}

	// having
	for _, h := range selectStmt.having {
		sql := h.Accept(context)
		addLine(sql)
	}

	// order by
	if selectStmt.orderBy != nil {
		sql := selectStmt.orderBy.Accept(context)
		addLine(sql)
	}

	if (selectStmt.offset != nil) && (selectStmt.count != nil) {
		addLine(fmt.Sprintf("LIMIT %d OFFSET %d", *selectStmt.count, *selectStmt.offset))
	}

	return strings.Join(lines, "\n")
}

// VisitTable returns a table name, optionally escaped
func (SQLCompiler) VisitTable(context *CompilerContext, table TableElem) string {
	return context.Compiler.VisitLabel(context, table.Name)
}

// VisitText return a raw SQL clause as is
func (SQLCompiler) VisitText(context *CompilerContext, text TextClause) string {
	return text.Text
}

// VisitUpdate compiles a UPDATE statement
func (c SQLCompiler) VisitUpdate(context *CompilerContext, update UpdateStmt) string {
	sql := "UPDATE " + update.table.Accept(context)

	var sets []string
	for k, v := range update.values {
		sets = append(sets, fmt.Sprintf(
			"%s = %s",
			context.Compiler.VisitLabel(context, k),
			context.Dialect.Placeholder(),
		))
		context.Binds = append(context.Binds, v)
	}

	if len(sets) > 0 {
		sql += "\nSET " + strings.Join(sets, ", ")
	}

	if update.where != nil {
		sql += "\n" + update.where.Accept(context)
	}

	returning := []string{}
	for _, c := range update.returning {
		returning = append(returning, context.Dialect.Escape(c.Name))
	}

	if len(returning) > 0 {
		sql += "\nRETURNING " + strings.Join(returning, ", ")
	}

	return sql
}

// VisitUpsert is not implemented and will panic.
// It should be implemented in each dialect
func (c SQLCompiler) VisitUpsert(context *CompilerContext, upsert UpsertStmt) string {
	panic("Upsert is not Implemented in this compiler")
}

// VisitWhere compiles a WHERE clause
func (c SQLCompiler) VisitWhere(context *CompilerContext, where WhereClause) string {
	return fmt.Sprintf("WHERE %s", where.clause.Accept(context))
}

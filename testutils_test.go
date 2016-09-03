package qb

func asDefSQL(clause Clause) string {
	return asSQL(clause, NewDialect("default"))
}

func asDefSQLBinds(clause Clause) (string, []interface{}) {
	return asSQLBinds(clause, NewDialect("default"))
}

func asSQL(clause Clause, dialect Dialect) string {
	sql, _ := asSQLBinds(clause, dialect)
	return sql
}

func asSQLBinds(clause Clause, dialect Dialect) (string, []interface{}) {
	defer dialect.Reset()
	ctx := NewCompilerContext(dialect)
	return clause.Accept(ctx), ctx.Binds
}

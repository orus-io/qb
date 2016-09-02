package qb

func asSQL(clause Clause, dialect Dialect) (string, []interface{}) {
	defer dialect.Reset()
	ctx := NewCompilerContext(dialect)
	return clause.Accept(ctx), ctx.Binds
}

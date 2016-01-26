package qbit

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuery(t *testing.T) {

	query := NewQuery()
	query.AddClause("SELECT name")
	query.AddClause("FROM user")

	query.AddClause("WHERE user.id = ?")
	query.AddBinding(5)

	assert.Equal(t, query.Clauses(), []string{"SELECT name", "FROM user", "WHERE user.id = ?"})
	assert.Equal(t, query.Bindings(), []interface{}{5})
}

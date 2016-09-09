package qb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWhereAnd(t *testing.T) {
	assert.Equal(t,
		"WHERE (X AND Y)", asDefSQL(
			Where(SQLText("X")).And(SQLText("Y"))))
	assert.Equal(t,
		"WHERE (X AND Y AND Z)",
		asDefSQL(
			Where(SQLText("X")).And(SQLText("Y"), SQLText("Z"))))
}

func TestWhereOr(t *testing.T) {
	assert.Equal(t,
		"WHERE (X OR Y)", asDefSQL(
			Where(SQLText("X")).Or(SQLText("Y"))))
	assert.Equal(t,
		"WHERE (X OR Y OR Z)",
		asDefSQL(
			Where(SQLText("X")).Or(SQLText("Y"), SQLText("Z"))))
}

package exec

import (
	"context"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestExecutor(t *testing.T) {
	e, err := NewExecutor("executor/test", "/Users/smintz/git/protoconf/protocorp/src", "localhost:4300")
	assert.NoError(t, err, "failed to create executor")
	defer e.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err = e.Start(ctx)
	assert.NoError(t, err)
}

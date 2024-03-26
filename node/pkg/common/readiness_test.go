package common

import (
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
)

func TestMustRegisterReadinessSyncing(t *testing.T) {
	// An invalid chainID should panic.
	assert.Panics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDUnset)
	})
}

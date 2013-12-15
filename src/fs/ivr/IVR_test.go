// IVR test

package ivr

import (
	"testing"
)

func TestIVR(t *testing.T) {

	DrawCallFlow()
	ExecuteCallFlow("welcome")
	t.Log("Test pass.")
}

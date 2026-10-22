package toolbox

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

type testTool struct {
	id     string
	closed bool
}

func (t *testTool) Info() Info     { return Info{ID: t.id, Name: "Test"} }
func (*testTool) View(*ui.Context) {}
func (t *testTool) Close()         { t.closed = true }

func TestRegistrationAndLifetime(t *testing.T) {
	var r Registry
	a, b := &testTool{id: "a"}, &testTool{id: "b"}
	if r.Register(a) != nil || r.Register(b) != nil || r.Register(a) == nil || r.Register(&testTool{}) == nil {
		t.Fatal("registration must reject duplicate and empty IDs")
	}
	if r.Get("a") != a || r.Get("missing") != nil || len(r.Tools()) != 2 {
		t.Fatal("registry lookup failed")
	}
	list := r.Tools()
	list[0] = b
	if r.Tools()[0] != a {
		t.Fatal("caller mutated registry storage")
	}
	r.Close()
	if !a.closed || !b.closed {
		t.Fatal("tools were not closed")
	}
}

package randomstate

import (
	"context"
	"testing"
	"unsafe"
)

func TestTableMemory(t *testing.T) {
	a, err := load333(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if a == nil || a.p1 == nil || a.p2 == nil || a.h1 == nil || a.h2 == nil {
		t.Fatal("333 tables were not initialized")
	}
	b, err := load222(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained table bytes 333=%d 222=%d", unsafe.Sizeof(*a)+unsafe.Sizeof(*a.p1)+unsafe.Sizeof(*a.p2)+unsafe.Sizeof(*a.h1)+unsafe.Sizeof(*a.h2), unsafe.Sizeof(*b)+uintptr(len(b.distance)))
}

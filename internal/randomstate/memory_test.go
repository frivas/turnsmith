package randomstate

import (
	"context"
	"testing"
	"unsafe"
)

func TestTableMemory(t *testing.T) {
	a, _ := load333(context.Background())
	b, _ := load222(context.Background())
	t.Logf("retained table bytes 333=%d 222=%d", unsafe.Sizeof(*a)+unsafe.Sizeof(*a.p1)+unsafe.Sizeof(*a.p2)+unsafe.Sizeof(*a.h1)+unsafe.Sizeof(*a.h2), unsafe.Sizeof(*b)+uintptr(len(b.distance)))
}

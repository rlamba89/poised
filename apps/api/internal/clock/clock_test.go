package clock

import (
	"testing"
	"time"
)

func TestFake(t *testing.T) {
	var f Fake
	start := f.Now()
	if d := time.Since(start); d < 0 || d > time.Minute {
		t.Fatalf("a new fake starts at the real time; it's %v off", d)
	}
	if !f.Now().Equal(start) {
		t.Error("a fake doesn't move by itself")
	}
	f.Add(8 * 24 * time.Hour)
	if got := f.Now().Sub(start); got != 8*24*time.Hour {
		t.Errorf("after Add(8 days) it moved %v", got)
	}
}

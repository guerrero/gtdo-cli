package todo

import (
	"testing"
	"time"
)

func TestIDAllocator(t *testing.T) {
	const existing = "20090213T044000.12Z"
	now := time.Date(2009, 2, 13, 4, 40, 0, 129_000_000, time.UTC)
	a := newIDAllocator([]string{
		"task without id",
		"(A) " + existing + " old",
	}, now)
	if got, want := a.nextID(), "20090213T044000.13Z"; got != want {
		t.Fatalf("nextID() = %q, want %q", got, want)
	}
	if got, want := a.nextID(), "20090213T044000.14Z"; got != want {
		t.Fatalf("second nextID() = %q, want %q", got, want)
	}
	if _, ok := a.used[existing]; !ok {
		t.Fatalf("existing id %q was not reserved", existing)
	}
}

func TestIDAllocatorRollsFractionIntoNextSecond(t *testing.T) {
	now := time.Date(2009, 2, 13, 4, 40, 59, 999_000_000, time.UTC)
	a := newIDAllocator(nil, now)
	if got, want := a.nextID(), "20090213T044059.99Z"; got != want {
		t.Fatalf("nextID() = %q, want %q", got, want)
	}
	if got, want := a.nextID(), "20090213T044100.00Z"; got != want {
		t.Fatalf("nextID() after rollover = %q, want %q", got, want)
	}
	if got, want := a.nextID(), "20090213T044100.01Z"; got != want {
		t.Fatalf("third nextID() = %q, want %q", got, want)
	}
}

func TestIDAllocatorReservesExplicitPrefixIDs(t *testing.T) {
	const explicit = "20090213T044000.12Z"
	now := time.Date(2009, 2, 13, 4, 40, 0, 129_000_000, time.UTC)
	a := newIDAllocator([]string{"x " + explicit + " done"}, now)
	a.reserve("20090213T044000.13Z")
	if got, want := a.nextID(), "20090213T044000.14Z"; got != want {
		t.Fatalf("nextID() = %q, want %q", got, want)
	}
	if _, ok := a.used[explicit]; !ok {
		t.Errorf("explicit prefix ID %q was not retained", explicit)
	}
}

package todo

import "time"

const (
	taskIDLayout = "20060102T150405.00Z"
	idStep       = 10 * time.Millisecond
)

// idAllocator generates timestamp IDs in one destination file's collision
// domain. next is advanced as a time.Time so fractional rollover follows the
// standard clock arithmetic at every second, minute, and date boundary.
type idAllocator struct {
	used map[string]struct{}
	next time.Time
}

// newIDAllocator reserves every canonical ID already present in lines and
// starts allocation at now truncated to the format's 10 ms precision.
func newIDAllocator(lines []string, now time.Time) *idAllocator {
	used := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if id := parseTaskPrefix(line).uuid; id != "" {
			used[id] = struct{}{}
		}
	}
	return &idAllocator{used: used, next: now.UTC().Truncate(idStep)}
}

// reserve marks an explicit or previously allocated ID as used. Empty IDs do
// not represent metadata and are intentionally ignored.
func (a *idAllocator) reserve(id string) {
	if id != "" {
		a.used[id] = struct{}{}
	}
}

// nextID returns the first free candidate at or after the allocator's clock,
// reserves it, and advances the clock for the next allocation.
func (a *idAllocator) nextID() string {
	for {
		candidate := a.next.Format(taskIDLayout)
		a.next = a.next.Add(idStep)
		if _, exists := a.used[candidate]; exists {
			continue
		}
		a.used[candidate] = struct{}{}
		return candidate
	}
}

// UUID returns the canonical timestamp ID at the task's reserved prefix
// position, or an empty string when the task has no such identifier.
func (t Task) UUID() string {
	return parseTaskPrefix(t.Text).uuid
}

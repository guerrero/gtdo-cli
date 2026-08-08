package todo

// Term filtering of task lists (plan §6.2.2, Task 3): terms are AND'ed with
// case-insensitive grep semantics, `-TERM` excludes, and `\|` inside a term
// means OR.

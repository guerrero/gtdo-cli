package todo

// Sorting per plan §6.2.3 (Task 3): case-insensitive by task text, ties
// broken by original file order via zero-padded line numbers, replicating
// `LC_COLLATE=C sort -f -k2`.

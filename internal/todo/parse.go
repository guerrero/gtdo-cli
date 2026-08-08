package todo

// Parsing of todo.txt lines into Tasks, per todo.sh conventions: the priority
// regex `^\([A-Z]\) `, the `(19|20)xx-xx-xx` date regex, and the `x ` done
// marker (plan §6.3, Task 3). The shell BREs are translated to RE2 and
// verified against the todo.sh tests.

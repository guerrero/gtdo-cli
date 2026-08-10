#!/usr/bin/env bash
#
# parity.sh — byte-parity verification between the real todo.sh
# (todo.txt-cli) and gtdo (design plan §7.3).
#
# Both binaries run the same flows against the same fixture and the same
# config values (todo.sh's bash config and gtdo's TOML carry the same
# paths), and stdout, stderr, exit codes, and resulting file states must
# match byte for byte. Any difference is printed and the script exits
# non-zero.
#
# Usage: scripts/parity.sh
#
# Requirements: Go toolchain; the todo.txt-cli checkout, by default
# /tmp/todo.txt-cli/todo.sh (override with $TODO_SH).
#
# The date is pinned the way the shell suite does it: todo.sh gets a
# bin/date shim over TODO_TEST_TIME (test-lib.sh), gtdo gets GTDO_TEST_NOW.
#
# Out of parity scope by design (§2, §6.4): usage/help/version texts are
# gtdo's own; addon flows do not exist in gtdo; the deferred edge cases
# listed in the task reports (empty-TERM del, zero-padded NRs, interactive
# readline prefill) are not exercised.

set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TODO_SH="${TODO_SH:-/tmp/todo.txt-cli/todo.sh}"
EPOCH=1234500000            # TODO_TEST_TIME (shell suite) …
NOW=2009-02-13T04:40:00Z    # … and GTDO_TEST_NOW (RFC 3339), same instant

export TZ=UTC LC_ALL=C

# macOS TMPDIR ends with a slash; a raw "$TMPDIR/..." would give the two
# binaries different path spellings (todo.sh echoes the config value
# verbatim, gtdo joins paths cleanly).
TMPDIR="${TMPDIR:-/tmp}"
TMPDIR="${TMPDIR%/}"

WORK="$(mktemp -d "$TMPDIR/gtdo-parity.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT
export TMPDIR="$WORK"       # todo.sh's sed wrapper stages its temp file here

TODO_DIR="$WORK/todo"
fail=0

# --- configs: the same values in both formats --------------------------
cat > "$WORK/todo.cfg" <<EOF
export TODO_DIR="$TODO_DIR"
export TODO_FILE="\$TODO_DIR/todo.txt"
export DONE_FILE="\$TODO_DIR/done.txt"
export REPORT_FILE="\$TODO_DIR/report.txt"
export TMP_FILE="\$TODO_DIR/todo.tmp"
EOF

cat > "$WORK/config.toml" <<EOF
[paths]
dir = "$TODO_DIR"
todo_file = "$TODO_DIR/todo.txt"
done_file = "$TODO_DIR/done.txt"
report_file = "$TODO_DIR/report.txt"
EOF

# --- fake date (test-lib.sh's bin/date shim) ---------------------------
mkdir -p "$WORK/bin"
if /bin/date -j -f %s 0 '+%Y' >/dev/null 2>&1; then
    cat > "$WORK/bin/date" <<'EOF'
#!/bin/bash
exec /bin/date -j -f %s "$TODO_TEST_TIME" "$@"
EOF
else
    cat > "$WORK/bin/date" <<'EOF'
#!/bin/bash
exec /bin/date -d "@$TODO_TEST_TIME" "$@"
EOF
fi
chmod +x "$WORK/bin/date"

# --- build gtdo into the scratch dir -----------------------------------
( cd "$ROOT" && go build -o "$WORK/gtdo" ./cmd/gtdo ) || exit 1

# --- fixture ------------------------------------------------------------
FIX="$WORK/fixture"
mkdir -p "$FIX"
cat > "$FIX/todo.txt" <<'EOF'
(A) Call mom +family @phone
(B) 2009-05-01 Pay rent +home
Water the plants +garden @home
x 2009-01-01 Done task
Buy milk
EOF
printf 'x 2009-01-02 Cleaned the garage\n' > "$FIX/done.txt"
printf 'notes for later\n' > "$FIX/inbox.txt"
printf 'keep me\n' > "$FIX/dest.txt"

reset() {
    rm -rf "$TODO_DIR"
    mkdir -p "$TODO_DIR"
    cp "$FIX"/* "$TODO_DIR"/
}

# run_both runs one flow against both binaries. Every process reopens the
# stdin file itself, so a confirm prompt in the first run never shifts the
# read offset of the second (a shared-fd redirect on the function call
# would).
run_both() {
    desc="$1"; shift
    reset
    (
        cd "$WORK"
        unset "${!TODOTXT_@}" TODO_FILE DONE_FILE REPORT_FILE TMP_FILE \
            TODO_DIR SENTENCE_DELIMITERS GTDO_CONFIG GTDO_TEST_NOW
        PATH="$WORK/bin:$PATH" TODO_TEST_TIME=$EPOCH \
            bash "$TODO_SH" -d "$WORK/todo.cfg" "$@" \
            < "$WORK/stdin" > "$WORK/sh.out" 2> "$WORK/sh.err"
    )
    shr=$?
    rm -rf "$WORK/sh.state"
    cp -r "$TODO_DIR" "$WORK/sh.state"
    reset
    (
        cd "$WORK"
        unset "${!TODOTXT_@}" TODO_FILE DONE_FILE REPORT_FILE TMP_FILE \
            TODO_DIR SENTENCE_DELIMITERS GTDO_CONFIG
        GTDO_TEST_NOW=$NOW "$WORK/gtdo" -d "$WORK/config.toml" "$@" \
            < "$WORK/stdin" > "$WORK/gd.out" 2> "$WORK/gd.err"
    )
    gdr=$?

    ok=1
    if ! cmp -s "$WORK/sh.out" "$WORK/gd.out"; then
        ok=0
        echo "stdout differs for: $desc (exit todo.sh=$shr gtdo=$gdr)"
        diff -u "$WORK/sh.out" "$WORK/gd.out" | head -20
    fi
    if ! cmp -s "$WORK/sh.err" "$WORK/gd.err"; then
        ok=0
        echo "stderr differs for: $desc (exit todo.sh=$shr gtdo=$gdr)"
        diff -u "$WORK/sh.err" "$WORK/gd.err" | head -20
    fi
    if [ "$shr" -ne "$gdr" ]; then
        ok=0
        echo "exit code differs for: $desc — todo.sh=$shr gtdo=$gdr"
    fi
    if ! diff -r "$WORK/sh.state" "$TODO_DIR" >/dev/null; then
        ok=0
        echo "file state differs for: $desc"
        diff -r "$WORK/sh.state" "$TODO_DIR" | head -20
    fi
    if [ "$ok" -eq 1 ]; then
        echo "ok: $desc"
    else
        fail=1
    fi
    rm -rf "$WORK/sh.state"
}

# stdin helpers: each check writes its own stdin file first; the
# interactive readline path of the reference todo.sh is broken on macOS
# bash 3.2 (read -i), so the argument form is used instead of stdin for
# replace (matches the t1100 sessions).
check() { : > "$WORK/stdin"; run_both "$@"; }
check_yes() { printf 'y' > "$WORK/stdin"; run_both "$@"; }
check_no() { printf 'n' > "$WORK/stdin"; run_both "$@"; }

# --- flows ---------------------------------------------------------------
# add
check "add" add "pay the rent +home @errand"
check "add alias" a "send the letter"
check "addm" addm "first line +p1 @c1
second line +p2 @c2"

# list: colors, filters, toggles, plain
check "list" list
check "list alias" ls
check "list filter" list @home
check "list OR filter" list 'pay\|water'
check "list no match" list zzz
check "list hide sigils" -@ -+ list
check "list hide priority" -P list
check "list plain" -p list
check "list color mode" -c list

# pri
check "pri" pri 3 A
check "pri replace" pri 1 C
check "pri invalid" pri 2 Q
check "pri reprioritize" -f pri 1 B
check "pri missing task" -f pri 99 A
check "listpri" listpri
check "listpri A" listpri A
check "listpri range" listpri A-C

# do
check "do" do 1
check "do already done" do 4
check "do missing" do 99

# del
check_yes "del confirmed" del 2
check_no "del declined" del 3
check "del forced" -f del 4
check "del TERM" -f del 1 "Call mom"
check "del TERM missing" -f del 1 dung
check "del missing task" -f del 99

# listcon/listproj
check "listcon" listcon
check "listcon alias" lsc
check "listcon filter" listcon home
check "listproj" listproj
check "listproj alias" lsprj

# archive/report
check "archive" archive
check "report" report
check "lsa" lsa
check "lsa filter" lsa @phone

# other mutating actions
check "append" append 1 ", please"
check "append space" append 1 "urgent"
check "prepend" prepend 1 "REMINDER"
check "depri" depri 1
check "depri none" depri 3
check "replace" replace 1 "A brand new task"
check_yes "move" move 2 dest.txt
check "move missing dest" -f move 2 nope.txt
check "move missing task" -f move 99 dest.txt
check "addto" addto inbox.txt "a note"
check "addto missing dest" addto nope.txt "a note"

if [ "$fail" -ne 0 ]; then
    echo
    echo "parity FAILED: differences found (see above)"
    exit 1
fi
echo
echo "parity OK: all flows byte-identical"

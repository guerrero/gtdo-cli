# Acciones de todo.txt-cli — Checklist de migración

Lista de las acciones de `todo.sh` (migración a Go).
Marca con `[x]` las que quieras implementar en el MVP.

## Acciones

- [x] **add** (`a`) — Añade una tarea a `todo.txt` en su propia línea. Si no se pasa texto, pide input interactivo (salvo con `-f`).
- [x] **addm** — Añade varias tareas a la vez: cada línea del input se convierte en una tarea.
- [x] **addto** — Añade una línea a cualquier archivo dentro del directorio TODO (p. ej. `inbox.txt`). El destino debe existir.
- [x] **append** (`app`) — Añade texto al final de la tarea en la línea NR. Inserta un espacio antes salvo que el texto empiece por delimitador de frase.
- [x] **archive** — Mueve las tareas hechas (`x `) de `todo.txt` a `done.txt` y elimina líneas en blanco.
- [ ] ~~**command**~~ — ~~Ejecuta el resto de argumentos usando solo builtins (sin addons).~~ **Fuera de alcance**: sin addons no tiene sentido.
- [ ] **deduplicate** — Elimina líneas duplicadas de `todo.txt`. **Fuera del MVP.**
- [x] **del** (`rm`) — Borra la tarea de la línea NR (con confirmación salvo `-f`). Con TERM opcional, borra solo esa palabra de la tarea. Con `-N`/`TODOTXT_PRESERVE_LINE_NUMBERS` deja línea en blanco.
- [x] **depri** (`dp`) — Quita la prioridad `(A)` de la(s) tarea(s) en NR [NR ...].
- [x] **do** (`done`) — Marca la(s) tarea(s) en NR [NR ...] como hechas: añade `x FECHA ` al inicio y opcionalmente las auto-archiva (según `-a`/`-A`).
- [x] **help** — Muestra ayuda de uso, opciones y acciones built-in, o de la acción pasada.
- [x] **shorthelp** — Lista el uso en una línea de todas las acciones built-in.
- [x] **list** (`ls`) — Lista tareas que contengan TERM(s) (AND lógico, OR con `\|`, exclusión con `-TERM`), ordenadas por prioridad y numeradas.
- [ ] ~~**listaddons**~~ — ~~Lista las acciones añadidas o sobreescritas en el directorio de acciones.~~ **Fuera de alcance**: no hay addons.
- [x] **listall** (`lsa`) — Igual que `list` pero sobre `todo.txt` + `done.txt` concatenados.
- [x] **listcon** (`lsc`) — Lista los contextos (`@...`) presentes en las tareas.
- [ ] **listfile** (`lf`) — Lista líneas de un archivo SRC del directorio TODO (o los nombres de archivos de texto si va sin argumentos), con filtros como `list`. **Fuera del MVP.**
- [x] **listpri** (`lsp`) — Lista tareas con prioridad (solo las PRIORITIES dadas, p. ej. `A` o `A-C`), con filtros opcionales.
- [x] **listproj** (`lsprj`) — Lista los proyectos (`+...`) presentes en las tareas.
- [x] **move** (`mv`) — Mueve la línea NR de SRC (por defecto `todo.txt`) a DEST, ambos dentro del directorio TODO.
- [x] **prepend** (`prep`) — Añade texto al principio de la tarea en la línea NR.
- [x] **pri** (`p`) — Añade (o reemplaza) la prioridad `(X)` a la tarea en la línea NR. La prioridad debe ser A-Z.
- [x] **replace** — Reemplaza la tarea de la línea NR por el texto dado.
- [x] **report** — Añade el número de tareas abiertas y hechas a `report.txt`.

## Dentro del alcance (extras)

- [x] **Configuración de colores** — El mapa de colores de `todo.cfg` (ANSI) para resaltado en `list*`.
- [x] **Bash completion** — Script de completado para bash.
- [x] **Fish completion** — Script de completado para fish.

## Notas

- **Opciones globales**: `-@ -+ -c -d CONFIG -f -h -p -P -a -A -n -N -t -T -v -vv -V`.
- **Fuera de alcance**: addons de `.todo.actions.d` (toda la funcionalidad), `command`, `deduplicate`, `listfile`.

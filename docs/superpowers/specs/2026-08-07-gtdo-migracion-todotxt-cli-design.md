# Diseño — gtdo: migración de todo.txt-cli a Go

- **Fecha**: 2026-08-07
- **Estado**: aprobado
- **Referencia**: [todo.txt-cli](https://github.com/todotxt/todo.txt-cli) (todo.sh v2.x)
- **Referencia de estructura**: [gitia](https://github.com/guerrero/gitia) (repositorio local)

## 1. Contexto y objetivos

Migrar el CLI de todo.txt (todo.sh, ~1571 líneas de bash) a un binario Go llamado **`gtdo`** con el mismo comportamiento exacto. La referencia de estructura de archivos, herramientas y convenciones es el repo local `gitia` (cobra, paquetes `internal/*`, tests txtar con go-internal, Makefile, goreleaser, man pages).

El criterio de aceptación central: **para una entrada dada, la salida (stdout, stderr, exit code, y estado de los archivos) debe ser idéntica a todo.sh byte a byte**, salvo las exclusiones declaradas en §2 y los textos de ayuda/versión propios (§6.4).

## 2. Alcance

### Dentro

- **20 acciones** con sus aliases: `add` (a), `addm`, `addto`, `append` (app), `archive`, `del` (rm), `depri` (dp), `do` (done), `help`, `shorthelp`, `list` (ls), `listall` (lsa), `listcon` (lsc), `listpri` (lsp), `listproj` (lsprj), `move` (mv), `prepend` (prep), `pri` (p), `replace`, `report`.
- Flags globales: `-@ -+ -a -A -c -d -f -h -n -N -p -P -t -T -v -V -x` (semántica idéntica a todo.sh, §6.1).
- Configuración JSON propia + variables de entorno (§5).
- Configuración de colores (§5.3).
- Completion de bash y fish (§6.6).
- Tests migrados (§7), man page, Makefile, goreleaser (§8).

### Fuera

- **Toda la funcionalidad de addons** (`.todo.actions.d`, `TODO_ACTIONS_DIR`): ejecución de addons, `listaddons`, `help`/`shorthelp` con sección de addons, `command` (sin addons no tiene sentido).
- `deduplicate` y `listfile` (fuera del MVP).
- `TODOTXT_SORT_COMMAND`, `TODOTXT_FINAL_FILTER`, `TODOTXT_DISABLE_FILTER` efectivo: son comandos shell `eval`'d, no replicables en Go. `-x` se acepta como no-op por compatibilidad de CLI (con config por defecto ya es no-op en todo.sh).
- `TODOTXT_SIGIL_BEFORE_PATTERN` / `SIGIL_VALID_PATTERN` / `SIGIL_AFTER_PATTERN`: regex POSIX BRE no traducibles 1:1 a RE2; se fijan los defaults (vacío / `.*`) en el código.
- Bash completion dinámico del todo_completion original (contextos/proyectos/tareas se cubren parcialmente con ValidArgsFunction de cobra, §6.6).

## 3. Decisiones de diseño (acordadas)

| Tema | Decisión |
|---|---|
| Nombre del binario | `gtdo` |
| Formato de config | JSON con la biblioteca estándar `encoding/json` — no se parsea bash |
| Ubicación config | `-d RUTA` / `$GTDO_CONFIG` > `~/.config/gtdo/config.json` > `/etc/gtdo/config.json` |
| Precedencia | flags CLI > env vars > JSON > defaults |
| Env vars | `TODO_DIR`, `TODO_FILE`, `DONE_FILE`, `REPORT_FILE`, `TODOTXT_*` siguen funcionando (compatibilidad con scripting) |
| Colores | Siempre emitidos salvo plain mode (`-p` o config), igual que todo.sh (no hay detección de TTY) |
| Prompts interactivos | Idénticos: `Add:`, `Append:`, `Delete '...'? (y/n)`; `-f` los salta |
| Addons | Eliminados por completo |

## 4. Arquitectura y layout

```
gtdo-cli/
├── cmd/gtdo/main.go          — entrypoint, señales, exit codes (patrón gitia)
├── internal/cli/             — árbol cobra: root + acciones, version, completions
│   └── testdata/script/*.txtar — tests de sesión black-box
├── internal/todo/            — dominio: Task, parse, filtros, sort, mutaciones, pipeline _format
├── internal/config/          — JSON (`json.go`, `loader.go`) + env vars + precedencia
├── internal/ui/              — colores ANSI, formato de salida
├── internal/exitcode/        — códigos de salida
├── tools/genman/             — generación de man page
├── man/gtdo.1                — man page generado y commiteado
├── Makefile                  — build/test/lint/man/install/release (patrón gitia)
├── .goreleaser.yaml
├── go.mod, AGENTS.md, CHANGELOG.md, LICENSE, README.md
└── ACTIONS.md                — checklist (ya existe)
```

Dependencias: `github.com/spf13/cobra`, `github.com/spf13/pflag`, `github.com/rogpeppe/go-internal` (tests), `golang.org/x/sys` (TTY si hiciera falta); la configuración usa la biblioteca estándar `encoding/json`.

## 5. Configuración

### 5.1 Resolución

Config search order: `-d PATH` / `$GTDO_CONFIG` > `~/.config/gtdo/config.json` > `/etc/gtdo/config.json`.

El cargador en `internal/config/loader.go` busca el primer archivo existente; si no existe ninguno usa los defaults. A diferencia de todo.sh no hay error fatal si falta el archivo (no hay archivo de configuración obligatorio).

### 5.2 Esquema JSON

```json
{
  "dir": "~/todo",
  "files": {
    "todo": "~/todo/todo.txt",
    "done": "~/todo/done.txt",
    "report": "~/todo/report.txt"
  },
  "behaviour": {
    "force": false,
    "preserveLineNumbers": true,
    "autoArchive": true,
    "dateOnAdd": false,
    "priorityOnAdd": "",
    "verbose": 1,
    "defaultAction": "",
    "sourceVar": "",
    "sentenceDelimiters": ",.:;"
  },
  "colors": {
    "priA": "yellow",
    "priB": "green",
    "priC": "light_blue",
    "priX": "white",
    "colorDone": "light_grey",
    "colorProject": "",
    "colorContext": "",
    "colorDate": "",
    "colorNumber": "",
    "colorMeta": "",
    "map": {"yellow": "\\033[1;33m"}
  }
}
```

Notas:
- El documento tiene las propiedades superiores `dir`, `files`, `behaviour` y `colors`; `behaviour` conserva deliberadamente la grafía británica. Los nombres compuestos usan camelCase.
- `colors` admite `priA` a `priZ`, los seis roles `color*` mostrados y `map`; sus valores pueden referenciar nombres del mapa o códigos ANSI directos.
- `$HOME` y `~` se expanden en rutas.
- Env vars: `TODO_DIR`, `TODO_FILE`, `DONE_FILE`, `REPORT_FILE`, `TODOTXT_FORCE`, `TODOTXT_PRESERVE_LINE_NUMBERS`, `TODOTXT_AUTO_ARCHIVE`, `TODOTXT_DATE_ON_ADD`, `TODOTXT_PRIORITY_ON_ADD`, `TODOTXT_VERBOSE`, `TODOTXT_DEFAULT_ACTION`, `TODOTXT_SOURCEVAR`, `TODOTXT_PLAIN`, `SENTENCE_DELIMITERS`.
- Los colores se configuran **solo por JSON** (todo.sh usa `export PRI_A=...` en bash; en gtdo las env vars de color no se soportan en el MVP).
- `internal/config/json.go` decodifica estrictamente con `encoding/json`: no se aceptan claves desconocidas, tipos incompatibles ni `null`.

### 5.3 Precedencia

Precedence: CLI flags > environment variables > JSON > defaults.

1. Flags CLI (los `OVR_*` de todo.sh): `-a/-A`, `-c/-p`, `-f`, `-n/-N`, `-t/-T`, `-v`, `-x` (no-op).
2. Env vars `TODO_*` / `TODOTXT_*`.
3. JSON.
4. Defaults de todo.sh: verbose=1, plain=0, force=0, preserve_line_numbers=1, auto_archive=1, date_on_add=0.

Semántica de `-v` replicada exactamente: si `TODOTXT_VERBOSE` env está definida manda ella; si no, `max(1, nº de -v)`. `-h` ≡ acción `shorthelp`.

## 6. Comportamiento

### 6.1 Flags

- `-@` oculta contextos (impar) / muestra (par); `-+` ídem proyectos; `-P` ídem etiquetas de prioridad (el número de apariciones alterna).
- `-c` plain=0; `-p` plain=1.
- `-d RUTA` configuración alternativa; `-f` force; `-h` → shorthelp; `-n` preserve=0; `-N` preserve=1; `-t` date_on_add=1; `-T` date_on_add=0; `-v` verbose++ ; `-V` versión (exit 0); `-x` no-op.
- Los flags se aceptan antes de la acción (getopts estilo todo.sh: `gtdo -p list`, no `gtdo list -p`). Cobra se configura para permitir flags solo antes del subcomando.

### 6.2 Pipeline de listado (`_format`)

1. **Numeración**: número de línea real del archivo, alineado a la derecha con padding a la anchura del total de líneas (`sed =` + reformateo; p. ej. 10+ tareas → ` 1`, `10`).
2. **Filtros** (`filtercommand`): cada término AND con `grep -i` (regex básica); `-TERM` → exclusión; `\|` dentro de un término → OR.
3. **Orden** (`LC_COLLATE=C sort -f -k2`): por texto de tarea case-insensitive; empates → orden original del archivo (los números van zero-padded antes de ordenar, lo que preserva el orden original).
4. **Colores** (awk de todo.sh): línea `^[0-9]+ x ` → `color_done`; `^[0-9]+ \([A-Z]\) ` → `pri_<letra>` (fallback `pri_x`); palabras: número → `color_number`, `+foo` (termina en alfanumérico) → `color_project`, `@foo` → `color_context`, fecha `(19|20)xx-xx-xx` válida → `color_date`, `key:value` → `color_meta`. `-P` elimina la etiqueta `(X)` de la salida. `-@`/`-+` eliminan los sigilos. El color de la línea se reinicia tras cada palabra coloreada (DEFAULT + color base de línea).
5. **Resumen** (verbose > 0): `--` + `PREFIX: N of M tasks shown`, donde PREFIX = basename del archivo sin extensión en mayúsculas (`TODO` para todo.txt, `DONE` para done.txt).

### 6.3 Acciones (semántica de todo.sh)

- **add/addm/addto**: limpian CR/LF; `add`/`addm` piden input interactivo si falta (`Add: `) salvo `-f`; `addto DEST` exige que el archivo exista en el TODO_DIR. `date_on_add` antepone `YYYY-MM-DD `; `priority_on_add` antepone `(X) ` (tras la fecha). Salida: `N tarea` + `TODO: N added.` (verbose>0).
- **append**: prompt `Append: ` si falta texto (salvo `-f`); espacio antes del texto salvo que empiece por delimitador de frase (`,.:;` configurable vía `SENTENCE_DELIMITERS`); escapa `\`, `|`, `&` para la sustitución sed (el efecto neto: el texto se inserta literal).
- **prepend**: idem sin espacio añadido; conserva prioridad y fecha existentes al inicio (regex `priAndDateExpr`).
- **pri**: valida A-Z; reemplaza prioridad existente conservando fecha; errores: `TODO: Invalid priority given. Must be capital A-Z.` / `TODO: No task $item.` / `TODO: $item already prioritized with (X).` (con `-f` lo re-prioriza).
- **do**: antepone `x YYYY-MM-DD ` (conservando prioridad); múltiples NR; auto-archive si `auto_archive` (mueve las líneas `x ` a done.txt; verbose: `TODO: $TODO_FILE archived.`).
- **del**: confirmación `Delete '...'? (y/n) ` salvo `-f` (respuesta `n` → `TODO: No tasks were deleted.` exit 1); con TERM borra solo el término (mensajes `TODO: 'TERM' not found; no removal done.` exit 1); `preserve_line_numbers` deja línea en blanco o compacta.
- **depri**: múltiples NR (también separados por comas); `TODO: $item no priority set.` si no tiene.
- **move**: confirmación si no es `-f`; valida destino; `TODO: No task $item in $SRC.` si no existe.
- **replace**: `TODO: No task $item.` si no existe.
- **archive**: mueve `x ` a done.txt, elimina líneas en blanco, mensajes `TODO: $TODO_FILE archived.` / `TODO: $TODO_FILE does not contain any done tasks.`.
- **list/listall/listpri/listcon/listproj**: pipeline §6.2; `listall` concatena todo.txt + done.txt; `listpri` acepta `A` o `A-C` (rango); `listcon`/`listproj` listan sigilos únicos (`sort -u`).
- **report**: escribe `N open tasks` / `M done tasks` en report.txt (fecha incluida).
- **help/shorthelp**: textos propios de gtdo (§6.4).

Los mensajes de error exactos (`die` → stderr, exit 1) se extraen de todo.sh durante la implementación; los tests los fijan byte a byte.

### 6.4 Textos de ayuda y versión

- `usage` (acción desconocida o sin acción): `Usage: gtdo [-fhpantvV] [-d todo_config] action [task_number] [task_description]` + `Try 'gtdo -h' for more information.` → stdout, exit 1.
- `shorthelp` / `-h`: lista one-line de acciones (sin sección de addons), con `gtdo` como nombre.
- `help [ACTION...]`: ayuda completa de gtdo + por acción; sin sección de addons.
- `-V`: texto de versión propio de gtdo (nombre, versión, repo), exit 0.
- El `--help` de cobra se desactiva/sobrescribe para no chocar con `-h` (que es shorthelp).

### 6.5 Otros comportamientos de todo.sh

- Crea `TODO_DIR` (mkdir -p) y los archivos todo/done/report si no existen.
- `SENTENCE_DELIMITERS` por defecto `,.:;`.
- Fechas: `date +%Y-%m-%d` local; los tests fijan `TZ=UTC`.
- `list` con `TODOTXT_SOURCEVAR` leyendo de otro archivo (solo listcon/listproj en todo.sh).
- Salida de `add`/`append`/etc. con número de línea real.

### 6.6 Completion

- Cobra `completion` (bash, fish) habilitado en el binario.
- ValidArgsFunction para completar `@contextos` y `+proyectos` (del TODO_FILE) y números de tarea donde aplique (paridad parcial con todo_completion; los tests t6xxx se portan solo en lo que cubra cobra).

## 7. Testing

### 7.1 Tests de sesión (txtar, go-internal)

Replican los `test_todo_session` de los tests shell en alcance: t1000 (add/list), t1010 (add-date), t1020/t1030 (addto), t1040 (add-priority), t1050 (todofile-override), t1100 (replace), t1200 (pri), t1250 (listpri), t1300 (ls), t1310 (listcon), t1320 (listproj), t1330 (ls-highlighting), t1340 (listescapes), t1350 (listall), t1360/t1380 (highlighting), t1400 (prepend), t1500 (do), t1600 (append), t1700 (depri), t1800 (del), t1850 (move), t1900 (archive), t1950 (report), t2000 (multiline), t2100/t2110/t2120 (help), t2200 (no-done-report-files), t0000 (config), t0001 (null), t0002 (actions/flags).

Cada caso: estado inicial de archivos (txtar) + secuencia de comandos `gtdo ...` con stdout/exit esperados. `TZ=UTC`, `HOME` aislado, sin red.

### 7.2 Unit tests

Por paquete: `internal/todo` (parse de prioridad/fecha, filtros, sort, mutaciones, pipeline), `internal/config` (resolución de rutas, precedencia, esquema JSON estricto con `encoding/json`), `internal/ui` (colores, padding, hide toggles).

### 7.3 Verificación de paridad

Durante el desarrollo: ejecutar el todo.sh real (en /tmp) y gtdo contra los mismos fixtures, comparando salidas. Los tests txtar son la garantía permanente.

## 8. Extras (patrón gitia)

- **Makefile**: `build` (ldflags con versión/commit/date), `test`, `lint` (golangci-lint), `man`, `install`, `release`/`release-dry` (goreleaser), `clean`.
- **Man page**: `tools/genman` + `man/gtdo.1` commiteado.
- **AGENTS.md** con convenciones del repo (como gitia, adaptado a gtdo).
- **CHANGELOG.md** en formato Keep a Changelog.
- **LICENSE** (MIT, igual que todo.txt-cli).
- **.goreleaser.yaml** con builds por plataforma y completions.

## 9. Criterios de aceptación

1. `go test ./...` verde.
2. Todos los tests txtar portados pasan con la misma salida que los tests shell originales.
3. Para un conjunto de fixtures, `gtdo` y `todo.sh` producen stdout/stderr/exit codes/estado de archivos idénticos (verificable con un script de comparación durante el desarrollo).
4. `make build`, `make man`, `make lint` funcionan.
5. No queda ninguna referencia a addons en el código ni en la ayuda.

## 10. Riesgos y notas

- La regex de fecha y los patrones de sigilos usan BRE en todo.sh; en Go se usan equivalentes RE2 cuidadosamente verificados contra los tests.
- `sort -f` de GNU en `LC_COLLATE=C` compara byte a byte tras lowercase; el comparador Go debe replicar empates por línea original.
- El prompt `Delete '...'? (y/n) ` usa `read -N 1` (un carácter) en bash moderno; en Go se lee un carácter con confirmación de Enter opcional — verificar contra el test t1800.
- `report` usa el formato exacto de todo.sh (verificar en implementación).

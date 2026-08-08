package config

import "strings"

// roles returns the [colors] keys the colorizer understands. The schema
// documents pri_a..pri_x; like todo.sh's PRI_<letter> every letter is valid,
// with pri_x as the fallback for letters without a color of their own.
func (c colorsTOML) roles() map[string]string {
	return map[string]string{
		"pri_a": c.PriA, "pri_b": c.PriB, "pri_c": c.PriC, "pri_d": c.PriD,
		"pri_e": c.PriE, "pri_f": c.PriF, "pri_g": c.PriG, "pri_h": c.PriH,
		"pri_i": c.PriI, "pri_j": c.PriJ, "pri_k": c.PriK, "pri_l": c.PriL,
		"pri_m": c.PriM, "pri_n": c.PriN, "pri_o": c.PriO, "pri_p": c.PriP,
		"pri_q": c.PriQ, "pri_r": c.PriR, "pri_s": c.PriS, "pri_t": c.PriT,
		"pri_u": c.PriU, "pri_v": c.PriV, "pri_w": c.PriW, "pri_x": c.PriX,
		"pri_y": c.PriY, "pri_z": c.PriZ,

		"color_done":    c.ColorDone,
		"color_project": c.ColorProject,
		"color_context": c.ColorContext,
		"color_date":    c.ColorDate,
		"color_number":  c.ColorNumber,
		"color_meta":    c.ColorMeta,
	}
}

// resolveColors computes the final ANSI code for every role. A value names a
// [colors.map] entry (case-insensitive) or is a raw ANSI string with \033
// translated to ESC; an empty value turns the color off. Plain mode turns
// every color off.
func resolveColors(c colorsTOML, plain bool) map[string]string {
	m := builtinColorMap()
	for name, code := range c.Map {
		m[strings.ToLower(name)] = code // user overrides win
	}
	roles := make(map[string]string, 32)
	for role, value := range c.roles() {
		if plain || value == "" {
			roles[role] = ""
			continue
		}
		roles[role] = resolveColor(m, value)
	}
	// default is todo.sh's DEFAULT, the reset emitted after every colored
	// word: it is not a [colors] key, defaults to \033[0m, and is
	// re-mappable through [colors.map] (t1330's customized highlighting).
	if plain {
		roles["default"] = ""
	} else {
		roles["default"] = m["default"]
	}
	return roles
}

// resolveColor resolves one [colors] value against the merged color map.
func resolveColor(m map[string]string, value string) string {
	if code, ok := m[strings.ToLower(value)]; ok {
		return unescapeColor(code)
	}
	if strings.EqualFold(value, "magenta") {
		if code, ok := m["purple"]; ok {
			return unescapeColor(code)
		}
	}
	return unescapeColor(value)
}

// unescapeColor translates the literal "\033" sequences todo.cfg uses into
// real ESC bytes; everything else passes through unchanged.
func unescapeColor(value string) string {
	return strings.ReplaceAll(value, `\033`, "\x1b")
}

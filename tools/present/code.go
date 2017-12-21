package present

import "regexp"

var PlayEnabled = false

func init() {
	Register("code", parseCode)
	Register("play", parseCode)
}

var (
	highlightRE = regexp.MustCompile(`\s+HL([a-zA-Z0-9_]+)?$`)
	hlCommentRE = regexp.MustCompile(`(.+) // HL(.*)$`)
	codeRE      = regexp.MustCompile(`\.(code|play)\s+((?:(?:-edit|-numbers)\s+)*)([^\s]+)(?:\s+(.*))?$`)
)


func parseCode(ctx *Context, sourceFile string, sourceLine int, cmd string) (Elem, error) {
	var e Elem

	return e, nil
}

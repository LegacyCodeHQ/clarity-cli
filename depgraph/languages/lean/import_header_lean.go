package lean

// leanImportHeader bounds tree-sitter's input to Lean's static import header.
// It does not extract names: the grammar still validates imports and supplies
// their identifier nodes. Declarations and extensible tactic bodies need not be
// parsed to determine file dependencies.
func leanImportHeader(source []byte) []byte {
	position := 0
	token, end := leanHeaderToken(source, position)
	if token == "module" {
		position = end
		token, end = leanHeaderToken(source, position)
	}
	if token == "prelude" {
		position = end
	}
	headerEnd := 0
	for {
		token, end = leanHeaderToken(source, position)
		if token == "public" {
			token, end = leanHeaderToken(source, end)
		}
		if token == "meta" {
			token, end = leanHeaderToken(source, end)
		}
		if token != "import" {
			break
		}
		token, end = leanHeaderToken(source, end)
		if token == "all" {
			token, end = leanHeaderToken(source, end)
		}
		if token == "" {
			break
		}
		headerEnd = end
		position = end
	}
	return source[:headerEnd]
}

func leanHeaderToken(source []byte, position int) (string, int) {
	for position < len(source) {
		switch {
		case isLeanHeaderSpace(source[position]):
			position++
		case hasLeanHeaderPrefix(source, position, "--"):
			for position < len(source) && source[position] != '\n' {
				position++
			}
		case hasLeanHeaderPrefix(source, position, "/-"):
			position += 2
			depth := 1
			for position < len(source) && depth > 0 {
				switch {
				case hasLeanHeaderPrefix(source, position, "/-"):
					depth++
					position += 2
				case hasLeanHeaderPrefix(source, position, "-/"):
					depth--
					position += 2
				default:
					position++
				}
			}
		default:
			goto token
		}
	}

token:
	start := position
	quoted := false
	for position < len(source) {
		switch {
		case !quoted && hasLeanHeaderPrefix(source, position, "«"):
			quoted = true
			position += len("«")
		case quoted && hasLeanHeaderPrefix(source, position, "»"):
			quoted = false
			position += len("»")
		case !quoted && (isLeanHeaderSpace(source[position]) ||
			hasLeanHeaderPrefix(source, position, "--") || hasLeanHeaderPrefix(source, position, "/-")):
			return string(source[start:position]), position
		default:
			position++
		}
	}
	return string(source[start:position]), position
}

func isLeanHeaderSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

func hasLeanHeaderPrefix(source []byte, position int, prefix string) bool {
	return len(source)-position >= len(prefix) && string(source[position:position+len(prefix)]) == prefix
}

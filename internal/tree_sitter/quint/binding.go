package quint

//#cgo CFLAGS: -std=c11 -fPIC
//#include "tree_sitter/parser.h"
//const TSLanguage *tree_sitter_quint(void);
import "C"

import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

func GetLanguage() *sitter.Language {
	return sitter.NewLanguage(unsafe.Pointer(C.tree_sitter_quint()))
}

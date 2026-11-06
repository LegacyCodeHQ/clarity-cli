# tree-sitter Quint

Vendored from [LegacyCodeHQ/tree-sitter-quint](https://github.com/LegacyCodeHQ/tree-sitter-quint)
at commit `f3b26e68c4f9a9b1153f7198c23c628c57b6dece`, package version 0.33.0.
The Apache-2.0 license and upstream attributions are preserved in `LICENSE` and
`NOTICE`, including the attribution for the Quint 0.33.0 recognition tables.

Sources copied from upstream:

- `src/scanner.c`
- `src/quint_033_atn.h`
- `src/quint_boundaries.h`
- `src/quint_scanner_boundary_state.h`
- `src/tree_sitter/*.h`

Upstream's parser uses ABI 15. In a temporary checkout of the pinned revision,
run `tree-sitter generate --abi 14` (CLI 0.27.1), then copy the generated
`src/parser.c` and `src/tree_sitter/*.h` into this directory. No grammar or
scanner logic changes are needed; trailing whitespace in `scanner.c` is
normalized when vendoring. The regenerated ABI 14 parser matches Clarity's
`smacker/go-tree-sitter` runtime. Do not edit the ABI constant by hand.

Normal Go builds need CGO, but do not need Quint, Node, the tree-sitter CLI,
or access to the grammar repository.

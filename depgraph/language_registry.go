package depgraph

import (
	"github.com/LegacyCodeHQ/clarity/depgraph/langsupport"
	"github.com/LegacyCodeHQ/clarity/depgraph/registry"
)

func moduleForExtension(ext string) (langsupport.Module, bool) {
	return registry.ModuleForExtension(ext)
}

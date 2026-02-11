package registry

import (
	"github.com/LegacyCodeHQ/clarity/depgraph/c"
	"github.com/LegacyCodeHQ/clarity/depgraph/cpp"
	"github.com/LegacyCodeHQ/clarity/depgraph/csharp"
	"github.com/LegacyCodeHQ/clarity/depgraph/dart"
	"github.com/LegacyCodeHQ/clarity/depgraph/golang"
	"github.com/LegacyCodeHQ/clarity/depgraph/java"
	"github.com/LegacyCodeHQ/clarity/depgraph/javascript"
	"github.com/LegacyCodeHQ/clarity/depgraph/kotlin"
	"github.com/LegacyCodeHQ/clarity/depgraph/langsupport"
	"github.com/LegacyCodeHQ/clarity/depgraph/python"
	"github.com/LegacyCodeHQ/clarity/depgraph/ruby"
	"github.com/LegacyCodeHQ/clarity/depgraph/rust"
	"github.com/LegacyCodeHQ/clarity/depgraph/swift"
	"github.com/LegacyCodeHQ/clarity/depgraph/typescript"
)

var modules = []langsupport.Module{
	c.Module{},
	cpp.Module{},
	csharp.Module{},
	dart.Module{},
	golang.Module{},
	javascript.Module{},
	java.Module{},
	kotlin.Module{},
	python.Module{},
	ruby.Module{},
	rust.Module{},
	swift.Module{},
	typescript.Module{},
}

// Modules returns supported language modules in deterministic order.
func Modules() []langsupport.Module {
	return append([]langsupport.Module(nil), modules...)
}

// ModuleForExtension returns the module registered for the provided extension.
func ModuleForExtension(ext string) (langsupport.Module, bool) {
	for _, module := range modules {
		for _, moduleExt := range module.Extensions() {
			if moduleExt == ext {
				return module, true
			}
		}
	}

	return nil, false
}

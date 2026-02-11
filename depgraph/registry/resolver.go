package registry

import "github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"

// Graph is the minimal graph contract language resolvers need during finalization.
type Graph = moduleapi.Graph

// Resolver resolves project imports for one language and can finalize graph-wide state.
type Resolver = moduleapi.Resolver

// Context contains precomputed project data shared across language resolvers.
type Context = moduleapi.Context

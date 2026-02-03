package json_test

import (
	"testing"

	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters"
	"github.com/LegacyCodeHQ/sanity/cmd/graph/formatters/json"
	"github.com/LegacyCodeHQ/sanity/parsers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyGraph_ToJSON(t *testing.T) {
	graph := parsers.DependencyGraph{
		"/project/main.dart":  {"/project/utils.dart", "/project/models/user.dart"},
		"/project/utils.dart": {},
	}

	formatter := &json.JSONFormatter{}
	jsonData, err := formatter.Format(graph, formatters.FormatOptions{})

	require.NoError(t, err)
	assert.Contains(t, jsonData, "/project/main.dart")
	assert.Contains(t, jsonData, "/project/utils.dart")
	assert.Contains(t, jsonData, "/project/models/user.dart")
}

package quint

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuintTestDefinitionAfterManyDeclarations(t *testing.T) {
	var source strings.Builder
	source.WriteString("module checks {\n")
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&source, "pure val value%d = %d\n", i, i)
	}
	source.WriteString("run finalTest = true\n}\n")
	require.True(t, IsTestFile("/project/checks.qnt", func(string) ([]byte, error) {
		return []byte(source.String()), nil
	}))
}

func TestQuintTestFilePaths(t *testing.T) {
	for _, path := range []string{"counterTest.qnt", "counterTests.qnt", "counter_test.qnt", "counter_tests.qnt", "test_counter.qnt", "/project/tests/counter.qnt", "tests/counter.qnt"} {
		require.True(t, IsTestFile(path, nil), path)
	}
	for _, path := range []string{"counter.qnt", "contest.qnt", "counterTest.go"} {
		require.False(t, IsTestFile(path, nil), path)
	}
}

func TestQuintTestDefinitions(t *testing.T) {
	for _, source := range []string{
		"module checks { run identityTest = true }",
		"module checks { action identityTest = true }",
		"module checks { pure val identityTest = true }",
		"module main { pure val x = 1 }\nmodule checks { run identityTest = true }",
	} {
		require.True(t, IsTestFile("/project/checks.qnt", func(string) ([]byte, error) {
			return []byte(source), nil
		}), source)
	}
	for _, source := range []string{
		"module main { run witness = true }",
		"module main { // run pretendTest = true\n pure val x = 1 }",
		`module main { pure val text = "identityTest" }`,
		"module main { pure def compute = { val nestedTest = true\nnestedTest } }",
	} {
		require.False(t, IsTestFile("/project/main.qnt", func(string) ([]byte, error) {
			return []byte(source), nil
		}), source)
	}
	require.False(t, IsTestFile("/project/main.qnt", func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}))
}

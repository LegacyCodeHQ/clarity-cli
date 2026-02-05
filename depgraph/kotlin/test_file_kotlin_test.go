package kotlin

import "testing"

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "kotlin test file suffix",
			filePath: "/project/src/test/kotlin/com/example/AppTest.kt",
			want:     true,
		},
		{
			name:     "kotlin tests file suffix",
			filePath: "/project/src/test/kotlin/com/example/AppTests.kt",
			want:     true,
		},
		{
			name:     "kotlin script test file suffix",
			filePath: "/project/src/test/kotlin/com/example/AppTest.kts",
			want:     true,
		},
		{
			name:     "kotlin file in test directory",
			filePath: "/project/test/Helper.kt",
			want:     true,
		},
		{
			name:     "kotlin non-test file",
			filePath: "/project/src/main/kotlin/com/example/App.kt",
			want:     false,
		},
		{
			name:     "non-kotlin file",
			filePath: "/project/src/test/com/example/App.java",
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTestFile(tc.filePath)
			if got != tc.want {
				t.Fatalf("IsTestFile(%q) = %v, want %v", tc.filePath, got, tc.want)
			}
		})
	}
}

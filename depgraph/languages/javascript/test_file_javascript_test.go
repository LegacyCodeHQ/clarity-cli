package javascript

import "testing"

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/repo/src/viewer.test.js", want: true},
		{path: "/repo/src/viewer.spec.jsx", want: true},
		{path: "/repo/src/viewer_state.test.mjs", want: true},
		{path: "/repo/src/viewer_state.spec.mjs", want: true},
		{path: "/repo/src/__tests__/viewer_state.mjs", want: true},
		{path: "/repo/src/viewer_state.mjs", want: false},
		{path: "/repo/src/viewer_state.ts", want: false},
	}

	for _, tc := range tests {
		got := IsTestFile(tc.path)
		if got != tc.want {
			t.Fatalf("IsTestFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

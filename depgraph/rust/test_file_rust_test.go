package rust

import "testing"

func TestIsTestFileWithContent(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
		want     bool
	}{
		{
			name:     "detects test attribute in tests directory",
			filePath: "/project/tests/lib.rs",
			content:  "fn main() {}\n\n#[test]\nfn it_works() {}\n",
			want:     true,
		},
		{
			name:     "detects cfg test module in tests directory",
			filePath: "/project/tests/lib.rs",
			content:  "#[cfg(test)]\nmod tests {\n    #[test]\n    fn sample() {}\n}\n",
			want:     true,
		},
		{
			name:     "non-test content in tests directory remains non-test",
			filePath: "/project/tests/lib.rs",
			content:  "fn main() {}\n",
			want:     false,
		},
		{
			name:     "test content in non-test path is ignored",
			filePath: "/project/src/lib.rs",
			content:  "#[test]\nfn it_works() {}\n",
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			contentReader := func(path string) ([]byte, error) {
				if path != tc.filePath {
					t.Fatalf("unexpected content read for %s", path)
				}
				return []byte(tc.content), nil
			}

			got := IsTestFileWithContent(tc.filePath, contentReader)
			if got != tc.want {
				t.Fatalf("IsTestFileWithContent(%q) = %v, want %v", tc.filePath, got, tc.want)
			}
		})
	}
}

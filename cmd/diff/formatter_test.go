package diff

import "testing"

func TestNewDiffFormatter_UnknownFormat(t *testing.T) {
	_, err := NewDiffFormatter("json")
	if err == nil {
		t.Fatal("expected unknown format error")
	}
}

func TestNewDiffFormatter_DOT(t *testing.T) {
	formatter, err := NewDiffFormatter("dot")
	if err != nil {
		t.Fatalf("NewDiffFormatter() error = %v", err)
	}
	if _, ok := formatter.(dotDiffFormatter); !ok {
		t.Fatalf("expected dotDiffFormatter, got %T", formatter)
	}
}

func TestNewDiffFormatter_Mermaid(t *testing.T) {
	formatter, err := NewDiffFormatter("mermaid")
	if err != nil {
		t.Fatalf("NewDiffFormatter() error = %v", err)
	}
	if _, ok := formatter.(mermaidDiffFormatter); !ok {
		t.Fatalf("expected mermaidDiffFormatter, got %T", formatter)
	}
}

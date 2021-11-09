package web

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestParseContentRange(t *testing.T) {
	tables := []struct {
		name string
		headerValue string
		expectedResult ContentRange
		expectErr bool
	}{
		{"handles satisfiable range", "bytes 0-20/30", ContentRange{"bytes", 0, 20, 30}, false},
		{"handles range without size", "bytes 10-20/*", ContentRange{"bytes", 10, 20, -1}, false},
		{"handles unsatisfiable range", "bytes */30", ContentRange{"bytes", -1, -1, 30}, false},
		{"returns null for invalid cases 1", "invalid", ContentRange{"", -1, -1, -1}, true},
		{"returns null for invalid cases 2", "bytes */*", ContentRange{"", -1, -1, -1}, true},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {

			result, err := ParseContentRange(table.headerValue)
			if table.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
			} else if !table.expectErr && err != nil {
				t.Errorf("Expected no error, got %#v", err)
			}

			if table.expectErr {
				return
			}

			if diff := cmp.Diff(table.expectedResult, result); diff != "" {
				t.Errorf("TestParseContentRange() mismatch (-want +got):\n%s", diff)
			}
		})


	}
}
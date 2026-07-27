package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPskVersion(t *testing.T) {
	tests := []struct {
		name  string
		value types.Int64
		want  int
	}{
		{name: "null uses compatibility default", value: types.Int64Null(), want: 2},
		{name: "unknown uses compatibility default", value: types.Int64Unknown(), want: 2},
		{name: "configured value", value: types.Int64Value(4), want: 4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pskVersion(test.value); got != test.want {
				t.Fatalf("pskVersion() = %d, want %d", got, test.want)
			}
		})
	}
}

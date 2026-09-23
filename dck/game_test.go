package megatwist

import (
	"testing"
)

func TestLogicalWidth(t *testing.T) {
	tests := []struct {
		name          string
		outsideWidth  int
		outsideHeight int
		want          int
	}{
		{name: "invalid", want: ContentWidth},
		{name: "native", outsideWidth: ContentWidth, outsideHeight: ContentHeight, want: ContentWidth},
		{name: "portrait", outsideWidth: 1080, outsideHeight: 2424, want: ContentWidth},
		{name: "pixel landscape", outsideWidth: 2424, outsideHeight: 1080, want: 1239},
		{name: "ultrawide cap", outsideWidth: 4000, outsideHeight: 1000, want: maxLayoutWidth},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := logicalWidth(test.outsideWidth, test.outsideHeight); got != test.want {
				t.Fatalf("logicalWidth(%d, %d) = %d, want %d", test.outsideWidth, test.outsideHeight, got, test.want)
			}
		})
	}
}

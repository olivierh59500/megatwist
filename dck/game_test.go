package megatwist

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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

func TestAppendScanlineReusesStorage(t *testing.T) {
	vertices := make([]ebiten.Vertex, 0, 4)
	indices := make([]uint16, 0, 6)
	vertices, indices = appendScanline(vertices, indices, 7, 11, 13)
	if len(vertices) != 4 || len(indices) != 6 {
		t.Fatalf("vertices, indices = %d, %d; want 4, 6", len(vertices), len(indices))
	}
	if vertices[0].DstY != 7 || vertices[0].SrcX != 11 || vertices[0].SrcY != 13 {
		t.Fatalf("first vertex = %+v", vertices[0])
	}
	if vertices[3].DstX != screenWidth || vertices[3].DstY != 8 {
		t.Fatalf("last vertex = %+v", vertices[3])
	}

	vertices = make([]ebiten.Vertex, 0, screenHeight*4)
	indices = make([]uint16, 0, screenHeight*6)
	allocations := testing.AllocsPerRun(100, func() {
		vertices = vertices[:0]
		indices = indices[:0]
		for line := 0; line < screenHeight; line++ {
			vertices, indices = appendScanline(vertices, indices, line, line%8, line%fontHeight)
		}
	})
	if allocations != 0 {
		t.Fatalf("scanline batch allocations = %.2f, want 0", allocations)
	}
}

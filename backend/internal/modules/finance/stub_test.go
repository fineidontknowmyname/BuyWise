package finance

import (
	"context"
	"testing"
	"uuid"

	"selfemi/internal/platform/provenance"
)

func TestStubStateSourceLabelsEveryValueAsStub(t *testing.T) {
	inputs, err := StubStateSource{}.Inputs(context.Background(), uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) != 3 {
		t.Fatalf("inputs = %+v", inputs)
	}
	for _, in := range inputs {
		if in.Source != provenance.Stub || in.Key == "" || in.Label == "" {
			t.Errorf("input = %+v", in)
		}
	}
}

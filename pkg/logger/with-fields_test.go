package logger

import "testing"

func TestWithFields(t *testing.T) {
	op := "create_order"

	withFields := WithFields("op", op, "orderID", "an-order-id")

	result := withFields()
	expected := []any{"op", op, "orderID", "an-order-id"}

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("At index %d: expected %v, got %v", i, expected[i], result[i])
		}
	}
}

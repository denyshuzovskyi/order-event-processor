package tests

import (
	"testing"
)

func TestInOrder(t *testing.T) {
	ExecuteScenario(t, "in_order", 0, []string{
		"483ec8f8-4864-427b-a878-ca026fd38f00",
		"483ec8f8-4864-427b-a878-ca026fd38f01",
		"483ec8f8-4864-427b-a878-ca026fd38f02",
		"483ec8f8-4864-427b-a878-ca026fd38f03",
	})
}

func TestNotInOrder(t *testing.T) {
	ExecuteScenario(t, "not_in_order", 0, []string{
		"483ec8f8-4864-427b-a878-ca026fd38f00",
		"483ec8f8-4864-427b-a878-ca026fd38f01",
		"483ec8f8-4864-427b-a878-ca026fd38f02",
		"483ec8f8-4864-427b-a878-ca026fd38f03",
	})
}

func TestNotInOrderAndInTheMiddleOfStream(t *testing.T) {
	ExecuteScenario(t, "not_in_order_middle_of_stream", 1000, []string{
		"483ec8f8-4864-427b-a878-ca026fd38f00",
		"483ec8f8-4864-427b-a878-ca026fd38f01",
		"483ec8f8-4864-427b-a878-ca026fd38f02",
		"483ec8f8-4864-427b-a878-ca026fd38f03",
	})
}

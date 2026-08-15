package notifications

import "testing"

func TestValidateListInputBounds(t *testing.T) {
	input := ListInput{Page: 0, PageSize: 1000}
	ValidateListInput(&input)
	if input.Page != 1 || input.PageSize != 50 {
		t.Fatalf("unexpected normalized input: %+v", input)
	}
}

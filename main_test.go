package main

import "testing"

func TestParseImageText(t *testing.T) {
	tests := []struct {
		command     string
		expectedNum int
		expectedText string
	}{
		{"/image1 prompt", 1, "prompt"},
		{"/image prompt", 1, "prompt"},
		{"/image42 more prompt", 42, "more prompt"},
		{"/image1234 some prompt", 1234, "some prompt"},
		{"/image-10 negative prompt", 1, "negative prompt"},
		{"/image  whitespace prompt", 1, "whitespace prompt"},
		{"/image0 zero prompt", 1, "zero prompt"},
		{"/image2 two prompt", 2, "two prompt"},
		{"/imageprompt", 1, "/imageprompt"},
		{"/image", 1, "/image"},
	}

	for _, test := range tests {
		num, text := parseImageCommand(test.command)

		if num != test.expectedNum || text != test.expectedText {
			t.Errorf("For command '%s', expected (%d, %s), but got (%d, %s)",
				test.command, test.expectedNum, test.expectedText, num, text)
		}
	}
}

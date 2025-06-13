package database

import (
	"testing"
)

func TestTruncateMessageContent(t *testing.T) {
	tests := []struct {
		name        string
		vendorLog   GenerativeUsage
		wantContent interface{}
	}{
		{
			name: "truncate string content",
			vendorLog: GenerativeUsage{
				Payload: PayloadData{
					Request: map[string]interface{}{
						"messages": []interface{}{
							map[string]interface{}{
								"role":    "user",
								"content": "This is a very long message that exceeds 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
							},
						},
					},
				},
			},
			wantContent: "This is a very long message that exceeds 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim ve",
		},
		{
			name: "truncate object content with text and image_url",
			vendorLog: GenerativeUsage{
				Payload: PayloadData{
					Request: map[string]interface{}{
						"messages": []interface{}{
							map[string]interface{}{
								"role": "user",
								"content": map[string]interface{}{
									"text": "This is a very long text that exceeds 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.",
									"image_url": map[string]interface{}{
										"url": "https://example.com/this/is/a/very/long/url/that/exceeds/200/characters/and/should/be/truncated/in/the/database/logs/lorem/ipsum/dolor/sit/amet/consectetur/adipiscing/elit/sed/do/eiusmod/tempor/incididunt/ut/labore/et/dolore/magna/aliqua/image.jpg",
									},
								},
							},
						},
					},
				},
			},
			wantContent: map[string]interface{}{
				"text": "This is a very long text that exceeds 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim venia",
				"image_url": map[string]interface{}{
					"url": "https://example.com/this/is/a/very/long/url/that/exceeds/200/characters/and/should/be/truncated/in/the/database/logs/lorem/ipsum/dolor/sit/amet/consectetur/adipiscing/elit/sed/do/eiusmod/tempor/incidi",
				},
			},
		},
		{
			name: "truncate array content with text and image_url items",
			vendorLog: GenerativeUsage{
				Payload: PayloadData{
					Request: map[string]interface{}{
						"messages": []interface{}{
							map[string]interface{}{
								"role": "user",
								"content": []interface{}{
									map[string]interface{}{
										"type": "text",
										"text": "This is a very long text that exceeds 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.",
									},
									map[string]interface{}{
										"type": "image_url",
										"image_url": map[string]interface{}{
											"url": "https://example.com/this/is/a/very/long/url/that/exceeds/200/characters/and/should/be/truncated/in/the/database/logs/lorem/ipsum/dolor/sit/amet/consectetur/adipiscing/elit/sed/do/eiusmod/tempor/incididunt/ut/labore/et/dolore/magna/aliqua/image.jpg",
										},
									},
								},
							},
						},
					},
				},
			},
			wantContent: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "This is a very long text that exceeds 200 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim venia",
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "https://example.com/this/is/a/very/long/url/that/exceeds/200/characters/and/should/be/truncated/in/the/database/logs/lorem/ipsum/dolor/sit/amet/consectetur/adipiscing/elit/sed/do/eiusmod/tempor/incidi",
					},
				},
			},
		},
		{
			name: "short content should not be truncated",
			vendorLog: GenerativeUsage{
				Payload: PayloadData{
					Request: map[string]interface{}{
						"messages": []interface{}{
							map[string]interface{}{
								"role":    "user",
								"content": "Short message",
							},
						},
					},
				},
			},
			wantContent: "Short message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			vendorLog := tt.vendorLog
			truncateMessageContent(&vendorLog)

			// Get the actual content after truncation
			messages := vendorLog.Payload.Request.(map[string]interface{})["messages"].([]interface{})
			actualContent := messages[0].(map[string]interface{})["content"]

			// Compare based on type
			switch expected := tt.wantContent.(type) {
			case string:
				if actual, ok := actualContent.(string); ok {
					if actual != expected {
						t.Errorf("String content mismatch:\nexpected: %q\nactual: %q", expected, actual)
					}
				} else {
					t.Errorf("Expected string content but got %T", actualContent)
				}

			case map[string]interface{}:
				actual, ok := actualContent.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map content but got %T", actualContent)
					return
				}
				// Check text field
				if expectedText := expected["text"].(string); actual["text"] != expectedText {
					t.Errorf("Text field mismatch:\nexpected: %q\nactual: %q", expectedText, actual["text"])
				}
				// Check image_url.url field
				if expectedURL := expected["image_url"].(map[string]interface{})["url"].(string); actual["image_url"].(map[string]interface{})["url"] != expectedURL {
					t.Errorf("Image URL mismatch:\nexpected: %q\nactual: %q", expectedURL,
						actual["image_url"].(map[string]interface{})["url"])
				}

			case []interface{}:
				actual, ok := actualContent.([]interface{})
				if !ok {
					t.Errorf("Expected array content but got %T", actualContent)
					return
				}
				// Check each item in the array
				for i, expectedItem := range expected {
					actualItem := actual[i].(map[string]interface{})
					expectedItemMap := expectedItem.(map[string]interface{})

					if actualItem["type"] != expectedItemMap["type"] {
						t.Errorf("Item %d type mismatch", i)
					}

					if expectedItemMap["type"] == "text" {
						if actualItem["text"] != expectedItemMap["text"] {
							t.Errorf("Item %d text mismatch:\nexpected: %q\nactual: %q",
								i, expectedItemMap["text"], actualItem["text"])
						}
					} else if expectedItemMap["type"] == "image_url" {
						actualURL := actualItem["image_url"].(map[string]interface{})["url"]
						expectedURL := expectedItemMap["image_url"].(map[string]interface{})["url"]
						if actualURL != expectedURL {
							t.Errorf("Item %d image URL mismatch:\nexpected: %q\nactual: %q",
								i, expectedURL, actualURL)
						}
					}
				}
			}
		})
	}
}

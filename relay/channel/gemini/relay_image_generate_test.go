package gemini

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestImageDataFromGeminiGenerateContent(t *testing.T) {
	images, err := imageDataFromGeminiGenerateContent(&dto.GeminiChatResponse{
		Candidates: []dto.GeminiChatCandidate{{
			Content: dto.GeminiChatContent{
				Parts: []dto.GeminiPart{{
					InlineData: &dto.GeminiInlineData{
						MimeType: "image/png",
						Data:     "abc123",
					},
				}},
			},
		}},
	})
	if err != nil {
		t.Fatalf("imageDataFromGeminiGenerateContent: %v", err)
	}
	if len(images) != 1 || images[0].B64Json != "abc123" {
		t.Fatalf("images = %+v", images)
	}
}

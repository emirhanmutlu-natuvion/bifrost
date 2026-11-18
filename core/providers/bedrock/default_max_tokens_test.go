package bedrock

import (
	"context"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/require"
)

func TestToBedrockChatCompletionRequest_DefaultMaxTokens(t *testing.T) {
	input := []schemas.ChatMessage{{
		Role:    schemas.ChatMessageRoleUser,
		Content: &schemas.ChatMessageContent{ContentStr: schemas.Ptr("hi")},
	}}
	tests := []struct {
		name   string
		model  string
		params *schemas.ChatParameters
		want   *int
	}{
		{
			// The CT-5551 case: an adaptive-thinking Claude with a reasoning budget
			// and no max_tokens must get the model's max output, not Bedrock's default.
			name:   "adaptive Claude without max tokens gets model max",
			model:  "eu.anthropic.claude-haiku-5-5",
			params: &schemas.ChatParameters{Reasoning: &schemas.ChatReasoning{MaxTokens: schemas.Ptr(8192)}},
			want:   schemas.Ptr(128000),
		},
		{
			name:  "Claude without params gets model max",
			model: "eu.anthropic.claude-sonnet-5-5",
			want:  schemas.Ptr(128000),
		},
		{
			name:   "explicit max tokens wins",
			model:  "eu.anthropic.claude-haiku-5-5",
			params: &schemas.ChatParameters{MaxCompletionTokens: schemas.Ptr(32000)},
			want:   schemas.Ptr(32000),
		},
		{
			name:   "non-Claude model is left to Bedrock",
			model:  "qwen.qwen3-32b-v1:0",
			params: &schemas.ChatParameters{},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
			req, err := ToBedrockChatCompletionRequest(ctx, &schemas.BifrostChatRequest{
				Model:  tt.model,
				Input:  input,
				Params: tt.params,
			})
			require.NoError(t, err)
			var got *int
			if req.InferenceConfig != nil {
				got = req.InferenceConfig.MaxTokens
			}
			require.Equal(t, tt.want, got)
		})
	}
}

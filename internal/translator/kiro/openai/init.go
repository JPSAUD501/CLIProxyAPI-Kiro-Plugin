// Package openai provides translation between OpenAI Chat Completions and Kiro formats.
package openai

import (
	"context"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

func init() {
	translator.Register(
		translator.FormatOpenAI,
		translator.FromString("kiro"),
		ConvertOpenAIRequestToKiro,
		translator.ResponseTransform{
			Stream: func(ctx context.Context, model string, originalRequest, request, response []byte, param *any) [][]byte {
				chunks := ConvertKiroStreamToOpenAI(ctx, model, originalRequest, request, response, param)
				out := make([][]byte, len(chunks))
				for i := range chunks {
					out[i] = []byte(chunks[i])
				}
				return out
			},
			NonStream: func(ctx context.Context, model string, originalRequest, request, response []byte, param *any) []byte {
				return []byte(ConvertKiroNonStreamToOpenAI(ctx, model, originalRequest, request, response, param))
			},
		},
	)
}

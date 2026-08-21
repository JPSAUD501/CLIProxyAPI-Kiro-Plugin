// Package claude provides translation between Kiro and Claude formats.
package claude

import (
	"context"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

func init() {
	translator.Register(
		translator.FormatClaude,
		translator.FromString("kiro"),
		ConvertClaudeRequestToKiro,
		translator.ResponseTransform{
			Stream: func(ctx context.Context, model string, originalRequest, request, response []byte, param *any) [][]byte {
				chunks := ConvertKiroStreamToClaude(ctx, model, originalRequest, request, response, param)
				out := make([][]byte, len(chunks))
				for i := range chunks {
					out[i] = []byte(chunks[i])
				}
				return out
			},
			NonStream: func(ctx context.Context, model string, originalRequest, request, response []byte, param *any) []byte {
				return []byte(ConvertKiroNonStreamToClaude(ctx, model, originalRequest, request, response, param))
			},
		},
	)
}

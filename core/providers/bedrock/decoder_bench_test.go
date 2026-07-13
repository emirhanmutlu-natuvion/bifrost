package bedrock

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/bytedance/sonic"
)

var bedrockStreamEventBenchmarkPayloads = map[string][]byte{
	"text":      []byte(`{"contentBlockIndex":0,"delta":{"text":"The migration plan should validate dependencies, execute the schema changes, and verify row counts before cutover."}}`),
	"reasoning": []byte(`{"contentBlockIndex":0,"delta":{"reasoningContent":{"text":"I should compare the current schema with the requested target and preserve foreign-key ordering."}}}`),
	"tool":      []byte(`{"contentBlockIndex":1,"delta":{"toolUse":{"input":"{\"query\":\"SELECT table_name FROM information_schema.tables\"}"}}}`),
	"usage":     []byte(`{"usage":{"inputTokens":25000,"outputTokens":500,"totalTokens":25500},"metrics":{"latencyMs":5000}}`),
}

func BenchmarkBedrockStreamEventDecodeCold(b *testing.B) {
	payload := bedrockStreamEventBenchmarkPayloads["text"]
	decoders := map[string]func([]byte, *BedrockStreamEvent) error{
		"sonic": func(payload []byte, event *BedrockStreamEvent) error {
			return sonic.Unmarshal(payload, event)
		},
		"encoding_json": func(payload []byte, event *BedrockStreamEvent) error {
			return json.Unmarshal(payload, event)
		},
	}

	for decoderName, decode := range decoders {
		b.Run(decoderName, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				// sync.Pool entries may survive one GC in the victim cache.
				runtime.GC()
				runtime.GC()
				var event BedrockStreamEvent
				if err := decode(payload, &event); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkBedrockStreamEventDecode(b *testing.B) {
	decoders := map[string]func([]byte, *BedrockStreamEvent) error{
		"sonic": func(payload []byte, event *BedrockStreamEvent) error {
			return sonic.Unmarshal(payload, event)
		},
		"encoding_json": func(payload []byte, event *BedrockStreamEvent) error {
			return json.Unmarshal(payload, event)
		},
	}

	for decoderName, decode := range decoders {
		for payloadName, payload := range bedrockStreamEventBenchmarkPayloads {
			b.Run(decoderName+"/"+payloadName, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var event BedrockStreamEvent
					if err := decode(payload, &event); err != nil {
						b.Fatal(err)
					}
				}
			})

			b.Run(decoderName+"/"+payloadName+"/parallel", func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						var event BedrockStreamEvent
						if err := decode(payload, &event); err != nil {
							b.Error(err)
							return
						}
					}
				})
			})
		}
	}
}

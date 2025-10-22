package genai

// GenerateContentConfig is configuration options when calling GenerateContent.
type GenerateContentConfig struct {
    // Optional tools the model may use (e.g. for function calling)
    Tools []*Tool

    // Optional maximum number of output tokens to generate
    MaxOutputTokens *int32

    // Optional randomness control
    Temperature *float32

    // Optional nucleus sampling parameter
    TopP *float32

    // Optional top‑k sampling parameter
    TopK *int32

    // Optional stop sequences (strings which, if generated, stop output)
    StopSequences []string

    // Optional seed for deterministic generation
    Seed *int64

    // Optional response MIME type (e.g. "application/json" or "text/plain")
    ResponseMIMEType *string
}

// 🚫 HAPUS atau KOMENTARI kode di bawah ini
// type Part interface {
//     toPart() Part
// }
//
// func (c *GenerateContentConfig) toPart() Part {
//     // stub: return itself or nil
//     return c
// }

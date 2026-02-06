package response

type Envelope map[string]any

// NewEnvelope creates a new Envelope instance with the provided value
func NewEnvelope[T any](data T) Envelope {
	return Envelope{
		"data": data,
	}
}

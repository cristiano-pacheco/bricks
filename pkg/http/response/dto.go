package response

type Envelope map[string]any

func NewEnvelope[T any](data T) Envelope {
	return Envelope{
		"data": data,
	}
}

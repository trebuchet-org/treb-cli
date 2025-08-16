package render

type Renderer[T any] interface {
	Render(result T) error
}

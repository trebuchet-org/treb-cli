package render

import (
	"io"

	"github.com/spf13/cobra"
)

type Renderer[T any] interface {
	Render(result T) error
}

func ProvideIO(cmd *cobra.Command) io.Writer {
	return cmd.OutOrStdout()

}

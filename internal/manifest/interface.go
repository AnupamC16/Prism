package manifest

import (
	"context"

	"github.com/anupam-chopra/prism/internal/model"
)

type GeneratorI interface {
	Generate(ctx context.Context, req *model.ManifestRequest) ([]byte, error)
}

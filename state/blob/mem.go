package blob

import (
	"context"
	"github.com/caryatid/cueball"
	"io"
)

type mem struct {
}

func NewMem(ctx context.Context) (cueball.Blob, error) {
	b := new(mem)
	return b, nil
}

func (b *mem) Save(key string, r io.Reader) error {
	return nil
}

func (b *mem) Load(key string) (io.Reader, error) {
	return nil, nil
}

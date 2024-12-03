package state

import (
	"cueball"
	"encoding/base64"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"sync"
)

var ChanSize = 1 // Configuration? Argument to NewOp?

type Op struct {
	sync.Mutex
	group   *errgroup.Group
	workers map[string]cueball.Worker
	intake  chan cueball.Worker
	init    func(*errgroup.Group, cueball.Worker)
}

func NewOp(g *errgroup.Group) (op *Op) {
	op.intake = make(chan cueball.Worker, ChanSize)
	op.group = g
	return
}

func (o *Op) Load(w cueball.Worker) {
	o.workers[w.Name()] = w
}

func (o *Op) Group() *errgroup.Group {
	return o.group
}

func (o *Op) Channel() chan cueball.Worker {
	return o.intake
}

func (o *Op) Workers() map[string]cueball.Worker {
	return o.workers
}

func unmarshal(data string, w json.Unmarshaler) error {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Debug().Err(err).Msg("failed decoding")
		return err
	}
	return json.Unmarshal(b, w)
}

func marshal(w json.Marshaler) ([]byte, error) {
	b, err := json.Marshal(w)
	if err != nil {
		log.Debug().Err(err).Msg("failed marshalling")
		return nil, err
	}
	data := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(data, b)
	return data, nil
}

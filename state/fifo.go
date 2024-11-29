package state

import(
//	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"context"
	"cueball"
	"sync"
	"io/fs"
	"syscall"
	"os"
	"bufio"
	"encoding/json"
	"encoding/base64"
)

var pre = ".cue"

func (f *Fifo) Group(ctx context.Context) *errgroup.Group {
	if f.group == nil {
		f.group, _ = errgroup.WithContext(ctx)
	}
	return f.group
}

type Fifo struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	m sync.Mutex
	in *os.File
	out *os.File
	workq chan cueball.Worker
	group *errgroup.Group
}

func NewFifo(name string, size int) (*Fifo, error) {
	var err error
	if size <= 0 {
		size = 1
	}
	if err = os.Mkdir(pre, 0700); err != nil {
		return nil, err
	}
	fname := pre + "/" + name
	s := &Fifo{workq: make(chan cueball.Worker, size)}
	if _, err := os.Stat(fname); err != nil {
		log.Debug().Err(err).Msg("making fifo")
		if err := syscall.Mkfifo(fname, 0644); err != nil {
			log.Debug().Err(err).Msg("failed making fifo")
			return nil, err
		}
	}
	// https://pubs.opengroup.org/onlinepubs/9699919799/functions/open.html#tag_16_357_03:
	// - “If O_NONBLOCK is clear, an open() for reading-only shall block the
	//   calling thread until a thread opens the file for writing. An open() for
	//   writing-only shall block the calling thread until a thread opens the file
	//   for reading.”
	// In order to unblock both open calls, we open the two ends of the FIFO
	// simultaneously in separate goroutines.
	check := make(chan error, 1)
	go func() {
		var err error
		s.in, err = os.OpenFile(fname, os.O_WRONLY, fs.ModeNamedPipe)
		if err != nil {
			check <- err
			return
		}
		check <- nil
	}()
	s.out, err = os.OpenFile(fname, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Debug().Err(err).Msg("failed opening fifo for reading")
		return nil, err
	}
	err = <- check
	if err != nil {
		log.Debug().Err(err).Msg("failed opening fifo for writing")
		return nil, err
	}
	
	return s, nil
}

func (s *Fifo) Channel() chan cueball.Worker {
	return s.workq
}

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker) error {
	d, err := marshal(w)
	if err != nil {
		return err
	}
	// TODO fixup State call
	dir := pre + "/" + w.Name() + "/" + w.State("")
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(dir + "/" + w.ID().String(), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(d, '\n')); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return f.Sync()
}

func unmarshal (data string, w cueball.Worker) error {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Debug().Err(err).Msg("failed decoding")
		return err
	}
	return json.Unmarshal(b, w)
}

func (s *Fifo) Dequeue(ctx context.Context, w cueball.Worker) {
	s.Group(ctx).Go(func () error {
		for {
			data, err := bufio.NewReader(s.out).ReadString('\n')
			if err != nil {
				log.Debug().Err(err).Msg("failed reading")
				return err
			}
			ww := w.New()
			unmarshal(data, ww)
			ww.FuncInit()
			s.workq <- ww
		}
	})
}

func marshal(w cueball.Worker) ([]byte, error) {
	b, err := json.Marshal(w)
	if err != nil {
		log.Debug().Err(err).Msg("failed marshalling")
		return nil, err
	}
	data := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(data, b)
	return data, nil
}

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.m.Lock()
	defer s.m.Unlock()
	dir := pre + "/" + w.Name() + "/" + w.State("")
	data, err := marshal(w)
	if _, err = s.in.Write(append(data, '\n')); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return s.in.Sync()
}


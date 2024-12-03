// state package for fifo
// TODO calling uuid new too often
package state

import (
	//	"github.com/rs/zerolog"
	"bufio"
	"context"
	"cueball"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"os"
	"syscall"
	"time"
)

var pre = ".cue"

type Pack struct {
	Name        string
	WorkerCodec string
}

type Fifo struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	*Op
	in  *os.File
	out *os.File
}

func NewFifo(ctx context.Context, g *errgroup.Group, name string, size int) (*Fifo, error) {
	var err error
	if size <= 0 {
		size = 1
	}
	if err = os.Mkdir(pre, 0700); err != nil {
		return nil, err
	}
	fname := pre + "/" + name
	s := new(Fifo)
	s.Op = NewOp(g)
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
	err = <-check
	if err != nil {
		log.Debug().Err(err).Msg("failed opening fifo for writing")
		return nil, err
	}

	return s, nil
}

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker) error {
	d, err := marshal(w)
	if err != nil {
		return err
	}
	dir := pre + "/" + w.Name()
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(dir+"/"+w.ID().String()+":"+
		string(time.Now().UnixNano()), os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(d, '\n')); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return f.Sync()
}

func (s *Fifo) Dequeue(ctx context.Context, w cueball.Worker) {
	s.Group().Go(func() error {
		for {
			data, err := bufio.NewReader(s.out).ReadString('\n')
			if err != nil {
				log.Debug().Err(err).Msg("failed reading")
				return err
			}
			p := new(Pack)
			if w_, ok := s.Workers()[p.Name]; ok {
				ww := w_.New()
				unmarshal(data, ww)
				ww.FuncInit()
				s.Channel() <- ww
			} else {
				s.enqueue(ctx, p.Name, p.WorkerCodec)
			}
		}
	})
}

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.Group().Go(func() error {
		s.Lock()
		defer s.Unlock()
		dir := pre + "/" + w.Name()
		dirf, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		ww := w.New() // reuse
		for _, de := range dirf {
			if de.IsDir() {
				continue
			}
			f, err := os.Open(dir + "/" + de.Name())
			if err != nil {
				return err
			}
			data, err := bufio.NewReader(f).ReadString('\n')
			if err != nil {
				return err
			}
			unmarshal(data, ww)
		}
		data, err := marshal(w)
		if _, err = s.in.Write(append(data, '\n')); err != nil {
			log.Debug().Err(err).Msg("failed writing")
			return err
		}
		return s.in.Sync()
	})
	return nil
}

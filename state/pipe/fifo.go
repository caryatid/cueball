//go:build linux

package pipe

import (
	"bufio"
	"context"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"io/fs"
	"os"
	"sync"
	"syscall"
	"time"
)

var pre = ".cue"
var sep = ":"

type fifo struct {
	wl  sync.Mutex
	rl  sync.Mutex
	in  *os.File
	out *os.File
	dir string
}

func NewFifo(ctx context.Context, name, dir string) (cueball.Pipe, error) {
	var err error
	p := new(fifo)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p.dir = dir
	fname := p.dir + "/" + name
	if _, err := os.Stat(fname); err != nil {
		if err := syscall.Mkfifo(fname, 0644); err != nil {
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
		p.in, err = os.OpenFile(fname, os.O_WRONLY, fs.ModeNamedPipe)
		if err != nil {
			check <- err
			return
		}
		check <- nil
	}()
	p.out, err = os.OpenFile(fname, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, err
	}
	err = <-check
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *fifo) Close() error {
	p.in.Close()
	p.out.Close()
	return nil
}

func (p *fifo) Enqueue(ctx context.Context, ch chan cueball.Worker) error {
	for w := range ch {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		time.Sleep(time.Millisecond * 5)
		data, err := state.Marshal(w)
		if err != nil {
			return err
		}
		pk := &state.Pack{Name: w.Name(), Codec: string(data)}
		wdata, err := state.Marshal(pk)
		if err != nil {
			return err
		}
		if err := func() error {
			p.wl.Lock()
			defer p.wl.Unlock()
			_, err := p.in.Write(append(wdata, '\n'))
			return err
		}(); err != nil {
			return err
		}
	}
	return nil
}

func (p *fifo) Dequeue(ctx context.Context, ch chan cueball.Worker) error {
	defer close(ch)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data, err := func() (string, error) {
				p.rl.Lock()
				defer p.rl.Unlock()
				return bufio.NewReader(p.out).ReadString('\n')
			}()
			if err != nil {
				return err
			}
			pk := new(state.Pack)
			if err := state.Unmarshal(data, pk); err != nil {
				return err
			}
			w := cueball.Gen(pk.Name)
			if err := state.Unmarshal(pk.Codec, w); err != nil {
				return err
			}
			ch <- w
		}
	}
	return nil
}

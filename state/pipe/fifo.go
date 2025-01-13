//go:build linux

package pipe

import (
	"bufio"
	"context"
	"github.com/caryatid/cueball"
	"io/fs"
	"os"
	"sync"
	"syscall"
)

var pre = ".cue"
var sep = ":"

type fifo struct {
	sync.Mutex
	in  *os.File
	out *os.File
	dir string
}

func NewFifo(ctx context.Context, name, dir string) (cueball.Pipe, error) {
	var err error
	p := new(fifo)
	if err = mkdir(dir); err != nil {
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

func (p *fifo) Enqueue(ctx context.Context, ch <-chan cueball.Worker) error {
	for w := range ch {
		data, err := marshal(w)
		if err != nil {
			return err
		}
		pk := &Pack{Name: w.Name(), Codec: string(data)}
		wdata, err := marshal(pk)
		if err != nil {
			return err
		}
		if _, err = p.in.Write(append(wdata, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func (p *fifo) Dequeue(ctx context.Context, ch chan<- cueball.Worker) error {
	defer close(ch)
	for {
		data, err := bufio.NewReader(p.out).ReadString('\n')
		if err != nil {
			return err
		}
		pk := new(Pack)
		if err := unmarshal(data, pk); err != nil {
			return err
		}
		w := cueball.Gen(pk.Name)
		if err := unmarshal(pk.Codec, w); err != nil {
			return err
		}
		ch <- w
	}
	return nil
}

// TODO generalize
func mkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}


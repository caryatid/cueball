//go:build linux

// state package for fifo
//
package state

import (
	"bufio"
	"context"
	"cueball"
	"fmt"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var pre = ".cue"

type Fifo struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	*Operator
	in  *os.File
	out *os.File
	dir *os.File
}

func NewFifo(ctx context.Context, name, dir string, w ...cueball.Worker) (*Fifo, error) {
	var err error
	s := new(Fifo)
	if err = mkdir(dir); err != nil {
		return nil, err
	}
	if s.dir, err = os.Open(dir); err != nil {
		return nil, err
	}
	fname := s.dir.Name() + "/" + name
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
		s.in, err = os.OpenFile(fname, os.O_WRONLY, fs.ModeNamedPipe)
		if err != nil {
			check <- err
			return
		}
		check <- nil
	}()
	s.out, err = os.OpenFile(fname, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, err
	}
	err = <-check
	if err != nil {
		return nil, err
	}
	s.Operator = NewOperator(w...)
	return s, nil
}

func mkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}

func (s *Fifo) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	fm, _ := s.filemap()
	f := fm[uuid.String()]
	return s.read(f)
}

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker) error {
	fname := fmt.Sprintf("%s/%s:%d:%s:%s", s.dir.Name(), w.ID().String(),
		time.Now().UnixNano(), w.Stage().String(), w.Name())
	d, err := marshal(w)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(d, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (s *Fifo) Dequeue(ctx context.Context) error {
	data, err := bufio.NewReader(s.out).ReadString('\n')
	if err != nil {
		return err
	}
	p := new(Pack)
	if err := unmarshal(data, p); err != nil {
		return err
	}
	w := s.Workers()[p.Name].New()
	if err := unmarshal(p.Codec, w); err != nil {
		return err
	}
	s.Work() <- w
	return nil
}

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	data, err := marshal(w)
	if err != nil {
		return err
	}
	p := &Pack{Name: w.Name(), Codec: string(data)}
	wdata, err := marshal(p)
	if err != nil {
		return err
	}
	if _, err = s.in.Write(append(wdata, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *Fifo) filemap() (map[string]string, error) {
	var curid string
	var pret int
	var err error
	fm := make(map[string]string)
	files, _ := s.dir.ReadDir(0) // TODO setup for batch
	for _, f := range files {
		ss := strings.Split(f.Name(), ":")
		if len(ss) < 4 {
			return nil, err
		}
		id, ts := ss[0], ss[1]
		curt, err := strconv.Atoi(ts)
		if err != nil {
			return nil, err
		}
		if id != curid {
			curid = id
			fm[id] = f.Name()
			continue
		} else if curt > pret {
			fm[id] = f.Name()
			pret = curt
		}
	}
	return fm, nil
}

func (s *Fifo) read(f string) (cueball.Worker, error) {
	df, err := os.Open(s.dir.Name() + "/" + f)
	if err != nil {
		return nil, err
	}
	ss := strings.Split(f, ":")
	if len(ss) < 4 {
		return nil, nil // TODO
	}
	name := ss[3]
	w := s.Workers()[name].New()
	data, err := bufio.NewReader(df).ReadString('\n')
	if err != nil {
		return nil, err
	}
	if err := unmarshal(data, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Fifo) Close() error {
	return nil
}

func (s *Fifo) LoadWork(ctx context.Context) error {
	m, err := s.filemap()
	if err != nil {
		return err
	}
	for _, f := range m {
		ss := strings.Split(f, ":")
		if len(ss) < 4 {
			return err // TODO fixit
		}
		stage := ss[2]
		if stage == "NEXT" || stage == "RETRY" || stage == "INIT" {
			w, err := s.read(f)
			if err != nil {
				return err
			}
			s.Intake() <- w
		}
	}
	return nil
}

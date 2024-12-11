// state package for fifo
// TODO calling uuid new too often
// COULD DO path handling - if more complete use path/filepath
//go:build linux
package state

import (
	"bufio"
	"context"
	"github.com/google/uuid"
	"cueball"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"fmt"
)

var pre = ".cue"

type Pack struct {
	Name  string
	Codec string
}

type Fifo struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	in  *os.File
	out *os.File
	dir *os.File
}

func NewFifo(ctx context.Context, name, dir string) (*Fifo, error) {
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
	return s, nil
}

func mkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}

// TODO handle stage
func (s *Fifo) Get(ctx context.Context, w cueball.Worker, uuid uuid.UUID) error {
	fm, _  := s.filemap()
	f := fm[uuid.String()]
	s.read(f, w)
	return nil
}

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker, st cueball.Stage) error {
	fname := fmt.Sprintf("%s/%s:%d:%s:%s", s.dir.Name(), w.ID().String(),
		time.Now().UnixNano(), st.String(), w.Name())
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

func (s *Fifo) Dequeue(ctx context.Context, w cueball.Worker) error {
	data, err := bufio.NewReader(s.out).ReadString('\n')
	if err != nil {
		return err
	}
	p := new(Pack)
	if err := unmarshal(data, p); err != nil {
		return err
	}
	if w.Name() == p.Name {
		return unmarshal(p.Codec, w)
	}
	return s.enqueue(p) // re-enqueue if not a known worker type. stop infinite loops, TTL?
}

func (s *Fifo) enqueue(p *Pack) error { // allows re-queuing a packed item
	data, err := marshal(p)
	if err != nil {
		return err
	}
	if _, err = s.in.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	data, err := marshal(w)
	if err != nil {
		return err
	}
	return s.enqueue(&Pack{Name: w.Name(), Codec: string(data)})
}

func (s *Fifo)filemap() (map[string]string, error) {
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
		curt, _ := strconv.Atoi(ts)
		if id != curid {
			curid = id
			fm[id] = f.Name()
			continue
		} else if curt > pret {
			fm[id] = f.Name()
		}
	}
	return fm, nil
}

func (s *Fifo) read(f string, w cueball.Worker) error {
	df, err := os.Open(s.dir.Name() + "/" + f)
	if err != nil {
		return err
	}
	ss := strings.Split(f, ":")
	if len(ss) < 4 {
		return nil // TODO
	}
	data, err := bufio.NewReader(df).ReadString('\n')
	if err != nil {
		return err
	}
	return unmarshal(data, w)
}

func (s *Fifo) LoadWork(ctx context.Context, w cueball.Worker, ch chan cueball.Worker) error {
	m, err :=  s.filemap()
	if err != nil {
		return err
	}
	for _, f := range m {
		ss := strings.Split(f, ":")
		if len(ss) < 4 {
			return err // TODO fixit
		}
		_, _, stage := ss[0], ss[1], ss[2]
		if stage == "NEXT" || stage == "RETRY" || stage == "INIT" {
			ww := w.New()
			if err := s.read(f, ww); err != nil {
				return err
			}
			ch <- ww
		}
	}
	return nil
}

func unmarshal(data string, w interface{}) error {
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, w)
}

func marshal(w interface{}) ([]byte, error) {
	b, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	data := make([]byte, base64.StdEncoding.EncodedLen(len(b)))
	base64.StdEncoding.Encode(data, b)
	return data, nil
}


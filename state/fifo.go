//go:build linux

package state

import (
	"bufio"
	"context"
	"fmt"
	"github.com/caryatid/cueball"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var pre = ".cue"
var sep = ":"

type Fifo struct {
	cueball.WorkerSet
	sync.Mutex
	in  *os.File
	out *os.File
	dir string
}

type execData struct {
	timestamp time.Time
	until     time.Time
	status    cueball.Status
	wname     string
}

func NewFifo(ctx context.Context, name, dir string, works ...cueball.WorkerGen) (*Fifo, error) {
	var err error
	s := new(Fifo)
	s.WorkerSet = DefaultWorkerSet(works...)
	if err = mkdir(dir); err != nil {
		return nil, err
	}
	s.dir = dir
	fname := s.dir + "/" + name
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
	go s.dequeue(ctx)
	return s, nil
}

func (s *Fifo) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	// TODO un-fubar
	fm, _ := s.filemap()
	return fm[uuid.String()], nil
}

func fnamePack(dirname string, w cueball.Worker) string {
	fmtstr := strings.Join([]string{"%s/%d", "%d", "%s", "%s"}, sep)
	return fmt.Sprintf(fmtstr, dirname, time.Now().UnixNano(),
		w.GetDefer().UnixNano(), w.Status().String(), w.Name())
}

func fnameUnpack(fname string) (*execData, error) {
	s := strings.Split(fname, sep)
	if len(s) < 4 {
		return nil, nil // TODO
	}
	st := new(cueball.Status)
	st.Scan(s[2])
	now, _ := strconv.ParseInt(s[0], 10, 64)
	until, _ := strconv.ParseInt(s[1], 10, 64)
	return &execData{time.Unix(0, now), time.Unix(0, until), *st, s[3]}, nil
}

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker) error {
	dname := s.dir + "/" + w.ID().String()
	if err := mkdir(dname); err != nil {
		return err
	}
	fname := fnamePack(dname, w)
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

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.Lock()
	defer s.Unlock()
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

func (s *Fifo) Close() error {
	return nil
}

func (s *Fifo) LoadWork(ctx context.Context) error {
	m, err := s.filemap()
	if err != nil {
		fmt.Errorf("ohh, %w", err)
		return err
	}
	for _, w := range m {
		if w.Status() == cueball.ENQUEUE {
			w.SetStatus(cueball.INFLIGHT)
			s.Persist(ctx, w)
			s.Work() <- w
		}
	}
	return nil
}

func (s *Fifo) filemap() (map[string]cueball.Worker, error) {
	fm := make(map[string]cueball.Worker)
	dir, err := os.Open(s.dir)
	defer dir.Close()
	if err != nil {
		return nil, err
	}
	ids, _ := dir.ReadDir(0)
	for _, id := range ids {
		if !id.IsDir() {
			continue
		} // TODO further validation of id
		wdir, err := os.Open(s.dir + "/" + id.Name())
		if err != nil {
			return nil, err
		}
		files, _ := wdir.ReadDir(0)
		if len(files) == 0 {
			continue
		}
		slices.SortFunc(files, func(a, b fs.DirEntry) int {
			ae, err := fnameUnpack(a.Name())
			if err != nil {
				return 0
			}
			be, err := fnameUnpack(b.Name())
			if err != nil {
				return 0
			} // NOTE: return vals flipped to make sort descending
			if ae.timestamp.Before(be.timestamp) {
				return 1
			} else if ae.timestamp.After(be.timestamp) {
				return -1
			}
			return 0
		})
		f := files[0].Name()
		df, err := os.Open(filepath.Join(s.dir, id.Name(), f))
		if err != nil {
			return nil, err
		}
		data, err := bufio.NewReader(df).ReadString('\n')
		if err != nil {
			return nil, err
		}
		fe, _ := fnameUnpack(f) // already done once; shouldn't error
		w := s.NewWorker(fe.wname)
		err = unmarshal(data, w)
		fm[id.Name()] = w
	}
	return fm, nil
}

func mkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}

func (s *Fifo) dequeue(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			data, err := bufio.NewReader(s.out).ReadString('\n')
			if err != nil {
				return err
			}
			p := new(Pack)
			if err := unmarshal(data, p); err != nil {
				return err
			}
			w := s.NewWorker(p.Name)
			if err := unmarshal(p.Codec, w); err != nil {
				return err
			}
			s.Work() <- w
		}
	}
}

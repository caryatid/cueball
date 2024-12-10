// state package for fifo
// TODO calling uuid new too often
// COULD DO path handling - if more complete use path/filepath
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
	*Op
	in  *os.File
	out *os.File
	dir *os.File
}

func NewFifo(ctx context.Context, name, dir string) (*Fifo, error) {
	var err error
	log := cueball.Lc(ctx)
	s := new(Fifo)
	s.Op = NewOp()
	if err = mkdir(dir); err != nil {
		return nil, err
	}
	if s.dir, err = os.Open(dir); err != nil {
		return nil, err
	}
	fname := s.dir.Name() + "/" + name
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

func mkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}
	return nil
}

func (s *Fifo) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	fm, _  := s.filemap()
	f := fm[uuid.String()]
	return s.read(f)
}

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	fname := fmt.Sprintf("%s/%s:%d:%s:%s", s.dir.Name(), w.ID().String(),
		time.Now().UnixNano(), w.Stage().String(), w.Name())
	d, err := marshal(w) // AWKWARD Must happen after w.ID() call
	if err != nil {
		log.Debug().Err(err).Msg("marshal failed")
		return err
	}
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Debug().Err(err).Str("filename", fname).Msg("open failed")
		return err
	}
	if _, err = f.Write(append(d, '\n')); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return f.Sync()
}

func (s *Fifo) Dequeue(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	data, err := bufio.NewReader(s.out).ReadString('\n')
	if err != nil {
		log.Debug().Err(err).Msg("failed reading")
		return err
	}
	p := new(Pack)
	if err := unmarshal(data, p); err != nil {
		log.Debug().Err(err).Msg("failed unmarshal")
		return err
	}
	if w_, ok := s.Workers()[p.Name]; ok {
		ww := w_.New()
		if err := unmarshal(p.Codec, ww); err != nil {
			log.Debug().Err(err).Msg("failed unmarshal")
			return err
		}
		ww.FuncInit()
		s.Channel() <- ww
	} else { // re-enqueue if not a known worker type. stop infinite loops, TTL?
		if err := s.enqueue(p); err != nil {
			log.Debug().Err(err).Msg("failed marshal")
			return err
		}
		log.Debug().Msg("re-enqueue")
	}
	return nil
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
	s.Lock()
	defer s.Unlock()
	log := cueball.Lc(ctx)
	data, err := marshal(w)
	if err != nil {
		log.Debug().Err(err).Msg("failed to marshal")
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

func (s *Fifo) read(f string) (cueball.Worker, error) {
	df, err := os.Open(s.dir.Name() + "/" + f)
	if err != nil {
		return nil, err
	}
	ss := strings.Split(f, ":")
	if len(ss) < 4 {
		return nil, nil
	}
	wname := ss[3]
	w, ok := s.Workers()[wname]
	if !ok {
		return nil, nil
	}
	ww := w.New()
	data, err := bufio.NewReader(df).ReadString('\n')
	err = unmarshal(data, ww)
	if err != nil {
		return nil, err
	}
	return ww, nil
}

func (s *Fifo) LoadWork(ctx context.Context, w cueball.Worker) error {
	log := cueball.Lc(ctx)
	m, err :=  s.filemap()
	if err != nil {
		return err
	}
	for _, f := range m {
		ss := strings.Split(f, ":")
		if len(ss) < 4 {
			log.Debug().Err(err).Msg("failed to split")
			return err // TODO fixit
		}
		_, _, stage := ss[0], ss[1], ss[2]
		if stage == "NEXT" || stage == "RETRY" {
			ww, err := s.read(f)
			err = s.Enqueue(ctx, ww)
			if err != nil {
				log.Debug().Err(err).Msg("failed to enqueue")
				return err
			}
			err = per(ctx, ww, cueball.RUNNING)
			if err != nil {
				log.Debug().Err(err).Msg("failed to persist")
				return err
			}
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


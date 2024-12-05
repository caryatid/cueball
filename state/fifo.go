// state package for fifo
// TODO calling uuid new too often
// COULD DO path handling - if more complete use path/filepath
package state

import (
	"bufio"
	"context"
	"cueball"
	"encoding/base64"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"os"
	"syscall"
	"time"
	"strings"
	"strconv"
	"encoding/json"
)

var pre = ".cue"

type Pack struct {
	Name        string
	Codec string
}

type Fifo struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	*Op
	in  *os.File
	out *os.File
	dir *os.File
}

func NewFifo(ctx context.Context, g *errgroup.Group, name, dir string) (*Fifo, error) {
	var err error
	log := zerolog.Ctx(ctx)
	s := new(Fifo)
	s.Op = NewOp(g)
	if err = os.Mkdir(dir, 0700); err != nil {
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

func (s *Fifo) Persist(ctx context.Context, w cueball.Worker, stage cueball.Stage) error {
	log := zerolog.Ctx(ctx)
	d, err := marshal(w)
	if err != nil {
		return err
	}
	dir := s.dir.Name() + "/" + w.Name()
	if err := os.Mkdir(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(dir+"/"+w.ID().String()+":"+
		string(time.Now().UnixNano()) + ":" + cueball.StageStr[int(stage)], os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err = f.Write(append(d, '\n')); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return f.Sync()
}

func (s *Fifo) Dequeue(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	data, err := bufio.NewReader(s.out).ReadString('\n')
	if err != nil {
		log.Debug().Err(err).Msg("failed reading")
		return err
	}
	p := new(Pack)
	if err := unmarshal(data, p); err != nil {
		return err
	}
	if w, ok := s.Workers()[p.Name]; ok {
		ww := w.New()
		if err := unmarshal(p.Codec, ww); err != nil {
			return err
		}
		ww.FuncInit()
		s.Channel() <- ww
	} else { // re-enqueue if not a known worker type. stop infinite loops, TTL?
		if err := s.enqueue(p); err != nil {
			return err
		}
	}
	return nil
}

func (s *Fifo) enqueue(p *Pack) error { // allows re-queuing a packed item
	data, err := marshal(p)
	if _, err = s.in.Write(append(data, '\n')); err != nil {
		return err
	}
	return s.in.Sync()
}

func (s *Fifo) Enqueue(ctx context.Context, w cueball.Worker) error {
	s.Lock()
	defer s.Unlock()
	data, err := marshal(w)
	if err != nil {
		return err
	}
	return s.enqueue(&Pack{Name: w.Name(), Codec: string(data)})
}

func (s *Fifo) LoadWork(ctx context.Context) error {
	for _, w := range s.Workers() {
		dir := s.dir.Name() + "/" + w.Name()
		if err := os.Mkdir(dir, 0700); err != nil {
			return err
		}
		files, _ := os.ReadDir(dir)
		var curid string
		var pret int
		var err error
		x := make(map[string]string)
		for _, f := range files {
			ss := strings.Split(f.Name(), ":")
			if len(ss) < 3 {
				// TODO fixit
				return err
			}
			id, ts, _ := ss[0], ss[1], ss[2]
			curt, _ := strconv.Atoi(ts)
			if id != curid {
				curid = id
				x[id] = f.Name()
				continue
			} else if curt > pret {
				x[id] = f.Name()
			}
		}
		for _, f := range x {
			ss := strings.Split(f, ":")
			if len(ss) < 3 {
				// TODO fixit
				return err
			}
			_, _, stage := ss[0], ss[1], ss[2]
			if stage == "NEXT" || stage == "RETRY" {
				// READ, enqueue, persist as RUNNING
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


package state

import(
//	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"cueball/worker"
	"sync"
	"io/fs"
	"syscall"
	"os"
	"bufio"
	"encoding/json"
	"encoding/base64"
)

type Fifo struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	m sync.Mutex
	in *os.File
	out *os.File
	workq chan worker.Worker
}

func NewFifo(fname string, size int) (*Fifo, error) {
	var err error
	if size <= 0 {
		size = 1
	}
	s := &Fifo{workq: make(chan worker.Worker, size)}
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

func (s *Fifo) Channel() chan worker.Worker {
	return s.workq
}

func (s *Fifo) Dequeue(w worker.Worker) {
	w.Group().Go(func () error {
		for {
			data, err := bufio.NewReader(s.out).ReadString('\n')
			if err != nil {
				log.Debug().Err(err).Msg("failed reading")
				return err
			}
			b, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				log.Debug().Err(err).Msg("failed decoding")
				return err
			}
			ww := w.New()
			err = json.Unmarshal(b, ww)
			if err != nil {
				log.Debug().Err(err).Msg("failed unmarshalling")
				return err
			}
			ww.FuncInit()
			s.workq <- ww
		}
	})
}

func (s *Fifo) Enqueue(w worker.Worker) error {
	s.m.Lock()
	defer s.m.Unlock()
	w.ID()
	b, err := json.Marshal(w)
	if err != nil {
		log.Debug().Err(err).Msg("failed marshalling")
		return err
	}
	data := base64.StdEncoding.EncodeToString(b)
	if _, err = s.in.Write([]byte(data + "\n")); err != nil {
		log.Debug().Err(err).Msg("failed writing")
		return err
	}
	return s.in.Sync()
}


package execute

import(
//	"fmt"
	"io/fs"
	"syscall"
	"os"
	"github.com/google/uuid"
	"bufio"
	"encoding/json"
	"encoding/base64"
)


type FifoState struct {
	// TODO !!! apparently Windoze does not have fifo's wtf.
	in *os.File
	out *os.File
	workq chan Worker
}

func NewFifoState(fname string, size int) (*FifoState, error) {
	var err error
	if size <= 0 {
		size = 1
	}
	s := &FifoState{workq: make(chan Worker, size)}
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
		}
		check <- nil
	}()
	s.out, err = os.Open(fname)
	if err != nil {
		return nil, err
	}
	err = <- check
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FifoState) Channel() chan Worker {
	return s.workq
}

func (s *FifoState) Dequeue(w Worker) error {
	for {
		data, err := bufio.NewReader(s.out).ReadString('\n')
		if err != nil {
			return err
		}
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return err
		}
		ww := w.New()
		err = json.Unmarshal(b, ww)
		if err != nil {
			return err
		}
		if ww.ID() == uuid.Nil {
			ww.GenID()
		}
		ww.FuncInit()
		s.workq <- ww
	}
	return nil
}

func (s *FifoState) Enqueue(w Worker) error {
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	data := base64.StdEncoding.EncodeToString(b)
	_, err = s.in.Write([]byte(data + "\n"))
	return err
}


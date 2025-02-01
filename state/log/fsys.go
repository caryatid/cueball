//go:build linux

package log

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/caryatid/cueball"
	"github.com/caryatid/cueball/state"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var pre = ".cue"
var sep = ":"

type fsys struct {
	sync.Mutex
	dir string
}

type metaData struct {
	timestamp time.Time
	until     time.Time
	status    cueball.Status
	wname     string
}

func NewFsys(ctx context.Context, dir string) (cueball.Record, error) {
	var err error
	l := new(fsys)
	if err = mkdir(dir); err != nil {
		return nil, err
	}
	l.dir = dir
	return l, nil
}

func (l *fsys) Get(ctx context.Context, uuid uuid.UUID) (cueball.Worker, error) {
	fm, _ := l.filemap()
	w, ok := fm[uuid.String()]
	if !ok {
		return nil, errors.New("get fail") // TODO make real
	}
	return w, nil
}

func fnamePack(dirname string, w cueball.Worker) string {
	fmtstr := strings.Join([]string{"%s/%d", "%d", "%s", "%s"}, sep)
	return fmt.Sprintf(fmtstr, dirname, time.Now().UnixNano(),
		w.GetDefer().UnixNano(), w.Status().String(), w.Name())
}

func fnameUnpack(fname string) (*metaData, error) {
	s := strings.Split(fname, sep)
	if len(s) < 4 {
		return nil, nil // TODO
	}
	st := new(cueball.Status)
	st.Scan(s[2])
	now, _ := strconv.ParseInt(s[0], 10, 64)
	until, _ := strconv.ParseInt(s[1], 10, 64)
	return &metaData{time.Unix(0, now), time.Unix(0, until), *st, s[3]}, nil
}

func (l *fsys) store(w cueball.Worker) error {
	l.Lock()
	defer l.Unlock()
	dname := l.dir + "/" + w.ID().String()
	if err := mkdir(dname); err != nil {
		return err
	}
	fname := fnamePack(dname, w)
	d, err := state.Marshal(w)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(d, '\n'))
	return err
}

func (l *fsys) Store(ctx context.Context, ch chan cueball.Worker) error {
	for w := range ch {
		l.store(w)
	}
	return nil
}

func (l *fsys) Scan(ctx context.Context, ch chan cueball.Worker) error {
	m, err := l.filemap()
	if err != nil {
		fmt.Errorf("ohh, %w", err)
		return err
	}
	for _, w := range m {
		if w.Status() == cueball.ENQUEUE {
			w.SetStatus(cueball.INFLIGHT)
			l.store(w)
			ch <- w
		}
	}
	return nil
}

func (l *fsys) filemap() (map[string]cueball.Worker, error) {
	fm := make(map[string]cueball.Worker)
	dir, err := os.Open(l.dir)
	defer dir.Close()
	if err != nil {
		return nil, err
	}
	ids, _ := dir.ReadDir(0)
	for _, id := range ids {
		if !id.IsDir() {
			continue
		} // TODO further validation of id
		wdir, err := os.Open(l.dir + "/" + id.Name())
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
		df, err := os.Open(filepath.Join(l.dir, id.Name(), f))
		if err != nil {
			return nil, err
		}
		data, err := bufio.NewReader(df).ReadString('\n')
		if err != nil {
			return nil, err
		}
		fe, _ := fnameUnpack(f) // already done once; shouldn't error
		w := cueball.GenWorker(fe.wname)
		err = state.Unmarshal(data, w)
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

func (l *fsys) Close() error {
	return nil
}

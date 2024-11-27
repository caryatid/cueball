package cueball

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Internal Error definitions
var (
	EndError  = errors.New("iteration complete")
	EnumError = errors.New("invalid enum value")
)

// Status enum definition
type Status int

const (
	INIT Status = iota
	ENQUEUE
	RUNNING
	RETRY
	NEXT
	DONE
	FAIL
)

var status2string = map[Status]string{
	INIT:    "INIT",
	ENQUEUE: "ENQUEUE",
	RUNNING: "RUNNING",
	RETRY:   "RETRY",
	NEXT:    "NEXT",
	DONE:    "DONE",
	FAIL:    "FAIL"}

var string2status = map[string]Status{
	"INIT":    INIT,
	"ENQUEUE": ENQUEUE,
	"RUNNING": RUNNING,
	"RETRY":   RETRY,
	"NEXT":    NEXT,
	"DONE":    DONE,
	"FAIL":    FAIL}

func (s *Status) MarshalJSON() ([]byte, error) {
	ss := status2string[*s]
	return json.Marshal(ss)
}

func (s Status) String() string {
	return status2string[s]
}

func (s *Status) UnmarshalJSON(b []byte) error {
	var ss string
	var ok bool
	if err := json.Unmarshal(b, &ss); err != nil {
		return err
	}
	*s, ok = string2status[ss]
	if !ok {
		return EnumError
	}
	return nil
}

func (s Status) Value() (driver.Value, error) {
	return status2string[s], nil
}

func (s *Status) Scan(value interface{}) error {
	if value == nil {
		*s = INIT
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = string2status[v]
	}
	return nil
}

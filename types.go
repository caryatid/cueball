package cueball

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
)

// Internal Error definitions
var (
	EnumError = errors.New("invalid enum value")
	EndError  = errors.New("iteration complete")
)

func NewError(es ...error) Error {
	e := Error{}
	e.Append(es...)
	return e
}

type Error struct {
	wraps []error
}

func (e Error) Append(es ...error) {
	e.wraps = append(e.wraps, es...)
}

func (e Error) Error() string {
	var ess []string
	for _, es := range e.wraps {
		ess = append(ess, es.Error())
	}
	return strings.Join(ess, ".")
}

func (e Error) Unwrap() []error {
	return e.wraps
}

func (s Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Error())
}

func (s Error) UnmarshalJSON(b []byte) error {
	var ss string
	if err := json.Unmarshal(b, &ss); err != nil {
		return err
	}
	s = NewError(errors.New("from unmarshal: " + ss))
	return nil
}

// Status enum definition
type Status int

const (
	ENQUEUE Status = iota
	INFLIGHT
	FAIL
	DONE
)

var status2string = map[Status]string{
	ENQUEUE:  "ENQUEUE",
	INFLIGHT: "INFLIGHT",
	FAIL:     "FAIL",
	DONE:     "DONE"}

var string2status = map[string]Status{
	"ENQUEUE":  ENQUEUE,
	"INFLIGHT": INFLIGHT,
	"FAIL":     FAIL,
	"DONE":     DONE}

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
		*s = ENQUEUE
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = string2status[v]
	}
	return nil
}

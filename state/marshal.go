package state

import (
	"encoding/json"
	"encoding/base64"
)
type Pack struct {
	Name  string
	Codec string
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


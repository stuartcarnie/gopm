package config

import (
	"encoding/json"
	"os"

	"github.com/stuartcarnie/gopm/signals"
)

// Signal holds an OS kill signal and its string form.
type Signal struct {
	S      os.Signal
	string string
}

// ParseSignal parses the signal with the given name.
func ParseSignal(str string) (Signal, error) {
	sig, err := signals.ToSignal(str)
	if err != nil {
		return Signal{}, err
	}
	return Signal{
		S:      sig,
		string: str,
	}, nil
}

func (s *Signal) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	sig, err := signals.ToSignal(str)
	if err != nil {
		return err
	}
	s.S = sig
	s.string = str
	return nil
}

func (s Signal) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.string)
}

func (s Signal) String() string {
	return s.string
}

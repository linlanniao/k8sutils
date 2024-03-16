package v2

import (
	"errors"
	"fmt"
	"regexp"
)

type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (p Parameter) Validate() error {
	// key must not be empty, and length must be between 1 and 32
	if len(p.Key) == 0 || len(p.Key) >= 32 {
		return errors.New("invalid parameter name, length must be between 1 and 32")
	}
	pattern := `^[-]{1,2}[a-zA-Z0-9_-]+$`
	if res, _ := regexp.MatchString(pattern, p.Key); !res {
		return fmt.Errorf("invalid parameter name, must match %s", pattern)
	}

	// value must not be empty
	if p.Value == "" {
		return errors.New("invalid parameter value, must not be empty")
	}

	return nil
}

type Parameters []*Parameter

func (ps Parameters) Validate() error {
	if len(ps) == 0 {
		return nil
	}
	for _, x := range ps {
		x := x
		if err := x.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (ps Parameters) IsEmpty() bool {
	return len(ps) == 0
}
func (ps Parameters) Len() int {
	return len(ps)
}

func (ps Parameters) ArgString() string {
	var s string
	for _, p := range ps {
		p := p
		s += p.Key + " " + p.Value + " "
	}
	return s
}

func (ps Parameters) Args() []string {
	args := make([]string, 0, len(ps))
	for _, p := range ps {
		p := p
		args = append(args, p.Key)
		args = append(args, p.Value)
	}
	return args
}

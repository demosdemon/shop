package secrets

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Secrets struct {
	Path  string
	value map[string]yaml.Node
}

func (s *Secrets) UnmarshalYAML(node *yaml.Node) error {
	*s = Secrets{}

	if node.Tag == "!secrets" {
		return node.Decode(&s.Path)
	}

	return node.Decode(&s.value)
}

func (s *Secrets) Get(key string) *Secret {
	v, ok := s.value[key]
	if !ok {
		return nil
	}

	sec := new(Secret)
	if err := v.Decode(sec); err == nil {
		return sec
	}

	return nil
}

type Secret struct {
	Path  string
	Value string
}

func (s *Secret) UnmarshalYAML(node *yaml.Node) error {
	*s = Secret{}

	if node.Tag == "!secret" {
		return node.Decode(&s.Path)
	}

	return node.Decode(&s.Value)
}

func (s Secret) String() string {
	if s.Path == "" {
		return s.Value
	}

	path, _ := yaml.Marshal(s.Path)
	return fmt.Sprintf("!secret %s", string(path))
}

func (s *Secret) Resolve(ctx context.Context, resolver Resolver) (string, error) {
	var err error

	if s.Path == "" {
		return s.Value, err
	}

	if s.Value == "" {
		s.Value, err = resolver.Resolve(ctx, s.Path)
	}

	return s.Value, err
}

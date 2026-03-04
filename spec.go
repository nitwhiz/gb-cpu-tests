package spec

import (
	"embed"
	"fmt"
	"iter"
	"path"

	"github.com/goccy/go-yaml"
)

const (
	ModeRead  = "read"
	ModeWrite = "write"
)

type Registers struct {
	A  uint8  `yaml:"a"`
	B  uint8  `yaml:"b"`
	C  uint8  `yaml:"c"`
	D  uint8  `yaml:"d"`
	E  uint8  `yaml:"e"`
	H  uint8  `yaml:"h"`
	L  uint8  `yaml:"l"`
	F  uint8  `yaml:"f"`
	PC uint16 `yaml:"pc"`
	SP uint16 `yaml:"sp"`
}

type RAMValue struct {
	Address uint16 `yaml:"address"`
	Value   uint8  `yaml:"value"`
}

type BusState struct {
	Address uint16 `yaml:"address"`
	Data    uint8  `yaml:"data"`
	Mode    string `yaml:"mode"`
}

type State struct {
	Registers `yaml:"registers"`
	RAM       []RAMValue `yaml:"ram"`
}

type Test struct {
	Name          string      `yaml:"name"`
	CanonicalName string      `yaml:"canonicalName,omitempty"`
	InitialState  State       `yaml:"initial"`
	FinalState    State       `yaml:"final"`
	BusCycles     []*BusState `yaml:"busCycles"`
}

type TestSuite struct {
	CanonicalName string `yaml:"canonicalName"`
	Tests         []Test `yaml:"tests"`
}

//go:embed spec/*.yaml
var specFS embed.FS

func Load(opcode uint8) (suite *TestSuite, err error) {
	bs, err := specFS.ReadFile("spec/" + fmt.Sprintf("%02x", opcode) + ".yaml")

	if err != nil {
		return nil, err
	}

	suite = &TestSuite{}

	err = yaml.Unmarshal(bs, suite)

	return
}

func LoadAll() iter.Seq2[*TestSuite, error] {
	return func(yield func(*TestSuite, error) bool) {
		entries, err := specFS.ReadDir("spec")

		if err != nil {
			yield(nil, err)
			return
		}

		for _, entry := range entries {
			var suite *TestSuite
			var bs []byte

			bs, err = specFS.ReadFile(path.Join("spec/", entry.Name()))

			if err != nil {
				yield(nil, err)
				return
			}

			suite = &TestSuite{}

			err = yaml.Unmarshal(bs, suite)

			if !yield(suite, err) {
				return
			}
		}
	}
}

package spec

import (
	"embed"
	"encoding/json"
	"fmt"
	"iter"
	"path"
	"strings"

	"github.com/goccy/go-yaml"
)

type Operand struct {
	Name      string `json:"name"`
	Immediate bool   `json:"immediate"`
}

func (o *Operand) String() string {
	if o.Immediate {
		return o.Name
	}

	return "[" + o.Name + "]"
}

type Opcode struct {
	Mnemonic string    `json:"mnemonic"`
	Operands []Operand `json:"operands"`
}

func (o *Opcode) String() string {
	if len(o.Operands) == 0 {
		return o.Mnemonic
	}

	operands := make([]string, len(o.Operands))

	for i, op := range o.Operands {
		operands[i] = op.String()
	}

	return o.Mnemonic + " " + strings.Join(operands, ", ")
}

type Opcodes struct {
	Unprefixed map[string]*Opcode `json:"unprefixed"`
	Prefixed   map[string]*Opcode `json:"cbprefixed"`
}

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

//go:embed data/opcodes.json
var opcodesBS []byte

var OpCodes Opcodes

func init() {
	if err := json.Unmarshal(opcodesBS, &OpCodes); err != nil {
		panic(err)
	}
}

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

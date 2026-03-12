package spec

import (
	"embed"
	"fmt"
	"iter"
	"path"
	"strings"

	"github.com/goccy/go-yaml"
)

type OpcodeOperand struct {
	Name      string `json:"name"`
	Bytes     int    `json:"bytes"`
	Immediate bool   `json:"immediate"`
}

func (o OpcodeOperand) String() string {
	if o.Immediate {
		return o.Name
	}

	return "[" + o.Name + "]"
}

type OpcodeFlags struct {
	Z string `json:"z"`
	N string `json:"n"`
	H string `json:"h"`
	C string `json:"c"`
}

type Opcode struct {
	Code      string          `yaml:"opcode"`
	Mnemonic  string          `yaml:"mnemonic"`
	Bytes     int             `yaml:"bytes"`
	Cycles    []int           `yaml:"cycles,flow"`
	Immediate bool            `yaml:"immediate"`
	Operands  []OpcodeOperand `yaml:"operands,omitempty"`
	Flags     OpcodeFlags     `yaml:"flags,flow"`
}

func (o Opcode) String() string {
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
	Unprefixed [256]Opcode `yaml:"unprefixed"`
	Prefixed   [256]Opcode `yaml:"prefixed"`
}

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

func (b BusState) IsRead() bool {
	return b.Mode[0] == 'r'
}

func (b BusState) IsWrite() bool {
	return b.Mode[1] == 'w'
}

func (b BusState) IsMemoryRequest() bool {
	return b.Mode[2] == 'm'
}

type State struct {
	Registers `yaml:"registers"`
	RAM       []RAMValue `yaml:"ram"`
}

type Test struct {
	Name         string      `yaml:"name"`
	InitialState State       `yaml:"initial"`
	FinalState   State       `yaml:"final"`
	BusCycles    []*BusState `yaml:"busCycles"`
}

type TestSuite struct {
	Opcode        string `yaml:"opcode"`
	CanonicalName string `yaml:"canonicalName"`
	Tests         []Test `yaml:"tests"`
}

//go:embed spec/*
var specFS embed.FS

//go:embed data/opcodes.yaml
var opcodesBS []byte

var OpcodeData Opcodes

func init() {
	if err := yaml.Unmarshal(opcodesBS, &OpcodeData); err != nil {
		panic(err)
	}
}

func Load(opcode uint8, prefixed bool) (suite *TestSuite, err error) {
	filename := fmt.Sprintf("%02x", opcode)

	if prefixed {
		filename = "cb " + filename
	}

	bs, err := specFS.ReadFile("spec/" + filename + ".yaml")

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

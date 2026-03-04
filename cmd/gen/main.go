package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	. "github.com/nitwhiz/gb-cpu-tests"
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

type OpCode struct {
	Mnemonic string    `json:"mnemonic"`
	Operands []Operand `json:"operands"`
}

func (o *OpCode) String() string {
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
	Unprefixed map[string]*OpCode `json:"unprefixed"`
	Prefixed   map[string]*OpCode `json:"cbprefixed"`
}

type V2RAMTuple struct {
	Address uint16
	Value   uint8
}

func (r *V2RAMTuple) UnmarshalJSON(data []byte) (err error) {
	var raw []json.RawMessage

	if err = json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if err = json.Unmarshal(raw[0], &r.Address); err != nil {
		return
	}

	if err = json.Unmarshal(raw[1], &r.Value); err != nil {
		return
	}

	return
}

type V2CycleBusState struct {
	Address uint16
	Data    uint8
	Mode    string
}

func (c *V2CycleBusState) UnmarshalJSON(data []byte) (err error) {
	var raw []json.RawMessage

	if err = json.Unmarshal(data, &raw); err != nil {
		return
	}

	if err = json.Unmarshal(raw[0], &c.Address); err != nil {
		return
	}

	if err = json.Unmarshal(raw[1], &c.Data); err != nil {
		return
	}

	if err = json.Unmarshal(raw[2], &c.Mode); err != nil {
		return
	}

	return
}

type V2State struct {
	A   uint8        `json:"a"`
	B   uint8        `json:"b"`
	C   uint8        `json:"c"`
	D   uint8        `json:"d"`
	E   uint8        `json:"e"`
	F   uint8        `json:"f"`
	H   uint8        `json:"h"`
	L   uint8        `json:"l"`
	PC  uint16       `json:"pc"`
	SP  uint16       `json:"sp"`
	RAM []V2RAMTuple `json:"ram"`
}

type V2Test struct {
	Name    string             `json:"name"`
	Initial V2State            `json:"initial"`
	Final   V2State            `json:"final"`
	Cycles  []*V2CycleBusState `json:"cycles"`
}

func clone() (err error) {
	cmd := exec.Command("git", "clone", "-b", "master", "https://github.com/adtennant/GameboyCPUTests.git", ".")

	cmd.Dir = "data/cpu_tests"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()

	if err != nil {
		slog.Error("unable to clone", slog.Any("err", err))
		os.Exit(1)
	}

	return
}

func update() (err error) {
	cmd := exec.Command("git", "pull")

	cmd.Dir = "data/cpu_tests"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()

	if err != nil {
		slog.Error("unable to pull", slog.Any("err", err))
		os.Exit(1)
	}

	return
}

func ensureCpuTestsDirectory() (err error) {
	stat, err := os.Stat("data/cpu_tests")

	if errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll("data/cpu_tests", 0755); err != nil {
			return
		}
	} else if err != nil {
		return
	} else if !stat.IsDir() {
		return
	}

	stat, err = os.Stat("data/cpu_tests/.git")

	if errors.Is(err, os.ErrNotExist) {
		err = clone()
	} else if err == nil {
		err = update()
	}

	return
}

func generateSpec() (err error) {
	if err = os.MkdirAll("spec", 0755); err != nil {
		return
	}

	bs, err := os.ReadFile("data/opcodes.json")

	if err != nil {
		return
	}

	var opcodes Opcodes

	if err = json.Unmarshal(bs, &opcodes); err != nil {
		return
	}

	v2Files, err := filepath.Glob("data/cpu_tests/v2/*.json")

	if err != nil {
		return
	}

	for _, v2File := range v2Files {
		baseName := filepath.Base(v2File)
		baseName = strings.TrimSuffix(baseName, ".json")

		slog.Info("processing v2 file ...", slog.String("file", v2File))

		if bs, err = os.ReadFile(v2File); err != nil {
			return
		}

		var v2Tests []V2Test

		if err = json.Unmarshal(bs, &v2Tests); err != nil {
			return
		}

		var suite TestSuite

		var op *OpCode
		isPrefixed := false

		if baseName == "cb" {
			suite.CanonicalName = "PREFIX"
			isPrefixed = true
		} else {
			suite.CanonicalName = opcodes.Unprefixed["0x"+strings.ToUpper(baseName)].String()
		}

		for _, t := range v2Tests {
			if isPrefixed {
				nameSegments := strings.SplitN(t.Name, " ", 3)
				op = opcodes.Prefixed["0x"+strings.ToUpper(nameSegments[1])]
			}

			iState := State{
				Registers: Registers{
					A:  t.Initial.A,
					B:  t.Initial.B,
					C:  t.Initial.C,
					D:  t.Initial.D,
					E:  t.Initial.E,
					F:  t.Initial.F,
					H:  t.Initial.H,
					L:  t.Initial.L,
					PC: t.Initial.PC,
					SP: t.Initial.SP,
				},
				RAM: make([]RAMValue, len(t.Initial.RAM)),
			}

			for i, ram := range t.Initial.RAM {
				iState.RAM[i] = RAMValue{
					Address: ram.Address,
					Value:   ram.Value,
				}
			}

			fState := State{
				Registers: Registers{
					A:  t.Final.A,
					B:  t.Final.B,
					C:  t.Final.C,
					D:  t.Final.D,
					E:  t.Final.E,
					F:  t.Final.F,
					H:  t.Final.H,
					L:  t.Final.L,
					PC: t.Final.PC,
					SP: t.Final.SP,
				},
				RAM: make([]RAMValue, len(t.Final.RAM)),
			}

			for i, r := range fState.RAM {
				fState.RAM[i] = RAMValue{
					Address: r.Address,
					Value:   r.Value,
				}
			}

			bus := make([]*BusState, len(t.Cycles))

			for i, b := range t.Cycles {
				if b == nil {
					continue
				}

				bus[i] = &BusState{
					Address: b.Address,
					Data:    b.Data,
					Mode:    b.Mode,
				}
			}

			test := Test{
				Name:         t.Name,
				InitialState: iState,
				FinalState:   fState,
				BusCycles:    bus,
			}

			if isPrefixed && op != nil {
				test.CanonicalName = op.String()
			}

			suite.Tests = append(suite.Tests, test)
		}

		if bs, err = yaml.Marshal(&suite); err != nil {
			return
		}

		if err = os.WriteFile(path.Join("spec/", baseName+".yaml"), bs, 0644); err != nil {
			return
		}
	}

	return
}

func main() {
	slog.Info("ensuring data/cpu_tests directory ...")

	if err := ensureCpuTestsDirectory(); err != nil {
		slog.Error("unable to ensure data/cpu_tests directory", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("generating spec files ...")

	if err := generateSpec(); err != nil {
		slog.Error("unable to generate spec files", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("done!")
}

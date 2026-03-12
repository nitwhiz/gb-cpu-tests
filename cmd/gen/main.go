package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
	. "github.com/nitwhiz/gb-cpu-tests"
)

const (
	source  = "https://github.com/SingleStepTests/sm83.git"
	workers = 6
)

type SourceOperand struct {
	Name      string `json:"name"`
	Bytes     int    `json:"bytes"`
	Immediate bool   `json:"immediate"`
}

func (o *SourceOperand) String() string {
	if o.Immediate {
		return o.Name
	}

	return "[" + o.Name + "]"
}

type SourceFlags struct {
	Z string `json:"z"`
	N string `json:"n"`
	H string `json:"h"`
	C string `json:"c"`
}

type SourceOpcode struct {
	Mnemonic  string          `json:"mnemonic"`
	Bytes     int             `json:"bytes"`
	Cycles    []int           `json:"cycles"`
	Immediate bool            `json:"immediate"`
	Operands  []SourceOperand `json:"operands"`
	Flags     SourceFlags     `json:"flags"`
}

func (o *SourceOpcode) String() string {
	if len(o.Operands) == 0 {
		return o.Mnemonic
	}

	operands := make([]string, len(o.Operands))

	for i, op := range o.Operands {
		operands[i] = op.String()
	}

	return o.Mnemonic + " " + strings.Join(operands, ", ")
}

type SourceOpcodes struct {
	Unprefixed map[string]*SourceOpcode `json:"unprefixed"`
	Prefixed   map[string]*SourceOpcode `json:"cbprefixed"`
}

type SourceRAMTuple struct {
	Address uint16
	Value   uint8
}

func (r *SourceRAMTuple) UnmarshalJSON(data []byte) (err error) {
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

type SourceCycleBusState struct {
	Address uint16
	Data    uint8
	Mode    string
}

func (c *SourceCycleBusState) UnmarshalJSON(data []byte) (err error) {
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

type SourceState struct {
	A   uint8            `json:"a"`
	B   uint8            `json:"b"`
	C   uint8            `json:"c"`
	D   uint8            `json:"d"`
	E   uint8            `json:"e"`
	F   uint8            `json:"f"`
	H   uint8            `json:"h"`
	L   uint8            `json:"l"`
	PC  uint16           `json:"pc"`
	SP  uint16           `json:"sp"`
	RAM []SourceRAMTuple `json:"ram"`
}

type SourceTest struct {
	Name    string                 `json:"name"`
	Initial SourceState            `json:"initial"`
	Final   SourceState            `json:"final"`
	Cycles  []*SourceCycleBusState `json:"cycles"`
}

type ProcessEvent struct {
	Code     string
	Op       *SourceOpcode
	Prefixed bool
}

var events = make(chan ProcessEvent)
var wg = &sync.WaitGroup{}

func workerProcess() {
	for e := range events {
		if err := generateOpCodeSpec(e.Code, e.Op, e.Prefixed); err != nil {
			slog.Error("unable to generate opcode spec", slog.Any("err", err))
			os.Exit(1)
			return
		}

		wg.Done()
	}
}

func init() {
	for i := 0; i < workers; i++ {
		go workerProcess()
	}
}

func clone() (err error) {
	cmd := exec.Command("git", "clone", source, ".")

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

func writeEmptySpecFile(code int64, op *SourceOpcode, prefixed bool) (err error) {
	baseName := fmt.Sprintf("%02x", code)

	if prefixed {
		baseName = fmt.Sprintf("cb %02x", code)
	}

	opcode := fmt.Sprintf("%02X", code)

	if prefixed {
		opcode = "CB " + opcode
	}

	suite := TestSuite{
		Opcode:        opcode,
		CanonicalName: op.String(),
	}

	var bs []byte

	if bs, err = yaml.Marshal(&suite); err != nil {
		return
	}

	slog.Info("writing empty test suite file ...", slog.String("baseName", baseName))

	err = os.WriteFile(path.Join("spec/", baseName+".yaml"), bs, 0644)

	return
}

func generateOpCodeSpec(strCode string, op *SourceOpcode, prefixed bool) (err error) {
	var code int64

	if code, err = strconv.ParseInt(strings.TrimPrefix(strCode, "0x"), 16, 64); err != nil {
		return
	}

	if code == 0xCB {
		return
	}

	baseName := fmt.Sprintf("%02x", code)

	if prefixed {
		baseName = fmt.Sprintf("cb %02x", code)
	}

	sourceFile := path.Join("data/cpu_tests/v1/", baseName+".json")

	_, err = os.Stat(sourceFile)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = writeEmptySpecFile(code, op, prefixed)
		}

		return
	}

	slog.Info("processing source file ...", slog.String("file", sourceFile))

	var bs []byte

	if bs, err = os.ReadFile(sourceFile); err != nil {
		return
	}

	var sourceTests []SourceTest

	if err = json.Unmarshal(bs, &sourceTests); err != nil {
		return
	}

	var suite TestSuite

	opcode := fmt.Sprintf("%02X", code)

	if prefixed {
		opcode = "CB " + opcode
	}

	suite.CanonicalName = op.String()
	suite.Opcode = opcode

	for _, t := range sourceTests {
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

		for i, r := range t.Final.RAM {
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

		suite.Tests = append(suite.Tests, test)
	}

	if bs, err = yaml.Marshal(&suite); err != nil {
		return
	}

	if err = os.WriteFile(path.Join("spec/", baseName+".yaml"), bs, 0644); err != nil {
		return
	}

	return
}

func generateSpecs() (err error) {
	if err = os.MkdirAll("spec", 0755); err != nil {
		return
	}

	bs, err := os.ReadFile("data/opcodes.json")

	if err != nil {
		return
	}

	var opcodes SourceOpcodes

	if err = json.Unmarshal(bs, &opcodes); err != nil {
		return
	}

	for code, op := range opcodes.Unprefixed {
		wg.Add(1)

		events <- ProcessEvent{
			Code:     code,
			Op:       op,
			Prefixed: false,
		}
	}

	for code, op := range opcodes.Prefixed {
		wg.Add(1)

		events <- ProcessEvent{
			Code:     code,
			Op:       op,
			Prefixed: true,
		}
	}

	wg.Wait()
	close(events)

	return
}

func coerceOpcode(table map[string]*SourceOpcode, code int) (op Opcode, err error) {
	inOp, ok := table[fmt.Sprintf("0x%02X", code)]

	if !ok {
		err = errors.New("opcode not found")
		return
	}

	op = Opcode{
		Code:      fmt.Sprintf("%02X", code),
		Mnemonic:  inOp.Mnemonic,
		Bytes:     inOp.Bytes,
		Cycles:    inOp.Cycles,
		Immediate: inOp.Immediate,
		Operands:  []OpcodeOperand{},
		Flags:     OpcodeFlags{},
	}

	for _, operand := range inOp.Operands {
		op.Operands = append(op.Operands, OpcodeOperand{
			Name:      operand.Name,
			Bytes:     operand.Bytes,
			Immediate: operand.Immediate,
		})
	}

	op.Flags.Z = inOp.Flags.Z
	op.Flags.N = inOp.Flags.N
	op.Flags.H = inOp.Flags.H
	op.Flags.C = inOp.Flags.C

	return
}

func generateOpcodesYAML() (err error) {
	var bs []byte

	if bs, err = os.ReadFile("data/opcodes.json"); err != nil {
		return
	}

	var inOps SourceOpcodes

	if err = json.Unmarshal(bs, &inOps); err != nil {
		return
	}

	var outOps Opcodes

	for code := range 256 {
		if outOps.Unprefixed[code], err = coerceOpcode(inOps.Unprefixed, code); err != nil {
			return
		}

		if outOps.Prefixed[code], err = coerceOpcode(inOps.Prefixed, code); err != nil {
			return
		}
	}

	if bs, err = yaml.Marshal(&outOps); err != nil {
		return
	}

	if err = os.WriteFile("data/opcodes.yaml", bs, 0644); err != nil {
		return
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

	if err := generateSpecs(); err != nil {
		slog.Error("unable to generate spec files", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("generating opcodes.yaml ...")

	if err := generateOpcodesYAML(); err != nil {
		slog.Error("unable to generate opcodes.yaml", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("done!")
}

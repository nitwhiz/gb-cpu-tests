package spec

import (
	"strings"
	"testing"
)

func TestLoad41(t *testing.T) {
	suite, err := Load(0x41, false)

	if err != nil {
		t.Fatal(err)
	}

	if suite.CanonicalName != "LD B, C" {
		t.Fatal("invalid canonical name, expected 'LD B, C', got", suite.CanonicalName)
	}

	suite, err = Load(0x41, true)

	if err != nil {
		t.Fatal(err)
	}

	if suite.CanonicalName != "BIT 0, C" {
		t.Fatal("invalid canonical name, expected 'BIT 0, C', got", suite.CanonicalName)
	}
}

func TestLoadAll(t *testing.T) {
	passedOne := false

	for suite, err := range LoadAll() {
		if err != nil {
			t.Fatal(err)
		}

		if suite == nil {
			t.Fatal("nil suite")
		}

		if suite.CanonicalName == "" {
			t.Fatal("empty suite canonical name")
		}

		if !strings.HasPrefix(suite.CanonicalName, "ILLEGAL") && len(suite.Tests) == 0 {
			t.Fatal("empty non-illegal suite")
		}

		for _, test := range suite.Tests {
			if test.Name == "" {
				t.Fatal("empty test name")
			}
		}

		passedOne = true

		t.Logf("tested suite for opcode %s '%s'", suite.Opcode, suite.CanonicalName)
	}

	if !passedOne {
		t.Fatal("no suites passed")
	}
}

func TestGetOpcode41(t *testing.T) {
	opcode := OpcodeData.Unprefixed[0x41]

	if opcode.Mnemonic != "LD" {
		t.Fatal("invalid opcode mnemonic, expected LD, got", opcode.Mnemonic)
	}

	opcode = OpcodeData.Prefixed[0x41]

	if opcode.Mnemonic != "BIT" {
		t.Fatal("invalid opcode mnemonic, expected BIT, got", opcode.Mnemonic)
	}
}

package spec

import "testing"

func TestLoad41(t *testing.T) {
	suite, err := Load(0x41)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", suite)
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

		if len(suite.Tests) == 0 {
			t.Fatal("empty suite")
		}

		for _, test := range suite.Tests {
			if test.Name == "" {
				t.Fatal("empty test name")
			}
		}

		passedOne = true
	}

	if !passedOne {
		t.Fatal("no suites passed")
	}
}

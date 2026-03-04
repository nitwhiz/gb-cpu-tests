package spec

import "testing"

func TestLoad41(t *testing.T) {
	suite, err := Load(0x41)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v", suite)
}

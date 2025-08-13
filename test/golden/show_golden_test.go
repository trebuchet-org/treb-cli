package golden

import (
	"github.com/trebuchet-org/treb-cli/test/helpers"
	"testing"
)

func TestShowCommandGolden(t *testing.T) {
	setup := [][]string{
		{"gen", "deploy", "src/Counter.sol:Counter"},
		{"run", "script/deploy/DeployCounter.s.sol"},
		{"run", "script/deploy/DeployCounter.s.sol", "--env", "label=prod", "--namespace", "production"},
	}

	tests := []GoldenTest{
		{
			Name:       "show_counter",
			SetupCmds:  setup,
			TestCmds:   [][]string{{"show", "Counter"}},
			GoldenFile: "commands/show/counter.golden",
		},
		{
			Name:      "show_with_namespace",
			SetupCmds: setup,
			TestCmds: [][]string{
				{"show", "Counter", "--namespace", "production"},
			},
			GoldenFile: "commands/show/counter_production.golden",
		},
		{
			Name: "show_proxy",
			Setup: func(t *testing.T, ctx *helpers.TrebContext) {
				// Skip proxy tests for now since proxy generation isn't implemented
				t.Skip("Proxy generation not yet implemented")
			},
			TestCmds: [][]string{
				{"show", "CounterProxy"},
			},
			GoldenFile: "commands/show/counter_proxy.golden",
		},
		{
			Name: "show_not_found",
			TestCmds: [][]string{
				{"show", "NonExistent"},
			},
			GoldenFile: "commands/show/not_found.golden",
			ExpectErr:  true,
		},
		{
			Name:      "show_by_address",
			SetupCmds: setup,
			TestCmds: [][]string{
				{"show", "0x74148047D6bDf624C94eFc07F60cEE7b6052FB29"},
			},
			GoldenFile: "commands/show/by_address.golden",
		},
	}

	RunGoldenTests(t, tests)
}

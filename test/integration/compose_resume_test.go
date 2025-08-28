package integration

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/trebuchet-org/treb-cli/test/helpers"
)

func TestComposeResume(t *testing.T) {
	tests := []IntegrationTest{
		{
			Name: "compose_with_failure_and_resume",
			PreSetup: func(t *testing.T, ctx *helpers.TestContext) {
				// Set env var to make the flaky deployment fail
				os.Setenv("FLAKY_SHOULD_FAIL", "true")
			},
			TestCmds: [][]string{
				// First command will fail
				s("compose compose-resume-test.yaml --network anvil-31337"),
			},
			ExpectErr: true,
			PostTest: func(t *testing.T, ctx *helpers.TestContext, output string) {
				assert.Contains(t, output, "Starting Step1")
				assert.Contains(t, output, "Starting Step2")
				assert.Contains(t, output, "failed") // Either "Deployment intentionally failed" or "step 'Step2' failed"
				assert.NotContains(t, output, "Starting Step3")

				// Now unset the env var for the resume test
				os.Unsetenv("FLAKY_SHOULD_FAIL")

				// Run with resume
				resumeOutput, err := ctx.TrebContext.Treb("compose", "compose-resume-test.yaml", "--network", "anvil-31337", "--resume")
				assert.NoError(t, err, "Expected resumed compose run to succeed")
				assert.Contains(t, resumeOutput, "Resuming compose from step 2 of 3")
				assert.Contains(t, resumeOutput, "🎉 Successfully orchestrated ResumeTest deployment")
			},
			OutputArtifacts: []string{
				".treb/deployments.json",
				"out/.treb/compose-compose-resume-test.json",
			},
		},
	}

	RunIntegrationTests(t, tests)
}

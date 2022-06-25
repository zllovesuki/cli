package cli

import (
	"testing"
)

// TestVersionOneTwentyOneRegression tests a regression that was merged between versions 1.20.0 and 1.21.0
// The included app.Run line worked in 1.20.0, and then was broken in 1.21.0.
// Relevant PR: https://github.com/urfave/cli/pull/872
func TestVersionOneTwentyOneRegression(t *testing.T) {
	testData := []struct {
		testCase       string
		appRunInput    []string
		skipArgReorder bool
	}{
		{
			testCase:    "with_dash_dash",
			appRunInput: []string{"cli", "command", "--flagone", "flagvalue", "--", "docker", "image", "ls", "--no-trunc"},
		},
		{
			testCase:       "with_dash_dash_and_skip_reorder",
			appRunInput:    []string{"cli", "command", "--flagone", "flagvalue", "--", "docker", "image", "ls", "--no-trunc"},
			skipArgReorder: true,
		},
		{
			testCase:    "without_dash_dash",
			appRunInput: []string{"cli", "command", "--flagone", "flagvalue", "docker", "image", "ls", "--no-trunc"},
		},
		{
			testCase:       "without_dash_dash_and_skip_reorder",
			appRunInput:    []string{"cli", "command", "--flagone", "flagvalue", "docker", "image", "ls", "--no-trunc"},
			skipArgReorder: true,
		},
	}
	for _, test := range testData {
		t.Run(test.testCase, func(t *testing.T) {
			// setup
			app := NewApp()
			app.Commands = []Command{{
				Name:           "command",
				SkipArgReorder: test.skipArgReorder,
				Flags: []Flag{
					StringFlag{
						Name: "flagone",
					},
				},
				Action: func(c *Context) error { return nil },
			}}

			// logic under test
			err := app.Run(test.appRunInput)

			// assertions
			if err != nil {
				t.Errorf("did not expected an error, but there was one: %s", err)
			}
		})
	}
}

// TestSkipArgReorderKnownFlags checks for a bug identified in
// https://github.com/urfave/cli/issues/1424
func TestSkipArgReorderKnownFlags(t *testing.T) {
	for _, tc := range []struct {
		name           string
		argv           []string
		expectedArgs   []string
		skipArgReorder bool
	}{
		{
			name:         "with_dash_dash",
			argv:         []string{"1424", "test", "--", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs: []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
		},
		{
			name:         "with_dash_dash_including_bool",
			argv:         []string{"1424", "test", "-t", "--", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs: []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
		},
		{
			name:           "with_dash_dash_and_skip_reorder",
			argv:           []string{"1424", "test", "--", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs:   []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			skipArgReorder: true,
		},
		{
			name:           "with_dash_dash_including_bool_and_skip_reorder",
			argv:           []string{"1424", "test", "-t", "--", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs:   []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			skipArgReorder: true,
		},
		{
			name:         "without_dash_dash",
			argv:         []string{"1424", "test", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs: []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
		},
		{
			name:         "without_dash_dash_including_bool",
			argv:         []string{"1424", "test", "-t", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs: []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
		},
		{
			name:           "without_dash_dash_and_skip_reorder",
			argv:           []string{"1424", "test", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs:   []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			skipArgReorder: true,
		},
		{
			name:           "without_dash_dash_including_bool_and_skip_reorder",
			argv:           []string{"1424", "test", "-t", "zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			expectedArgs:   []string{"zeta", "--beta", "theta", "--alpha", "ori", "--nori", "oin", "--gloin", "-t"},
			skipArgReorder: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := NewApp()
			app.Commands = []Command{{
				Name:           "test",
				SkipArgReorder: tc.skipArgReorder,
				Flags: []Flag{
					BoolFlag{Name: "t"},
				},
				Action: func(c *Context) error {
					expect(t, c.Args(), Args(tc.expectedArgs))
					return nil
				},
			}}

			expect(t, app.Run(tc.argv), nil)
		})
	}
}

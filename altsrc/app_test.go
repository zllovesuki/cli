package altsrc_test

import (
	"fmt"
	"log"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func ExampleApp_Run_sliceDestinations() {
	testInt := int(0)
	appState := map[string]bool{}

	testFloat64Slice := cli.NewFloat64Slice()
	testIntSlice := cli.NewIntSlice()
	testInt64Slice := cli.NewInt64Slice()
	testStringSlice := cli.NewStringSlice()

	flags := []cli.Flag{
		altsrc.NewFloat64SliceFlag(&cli.Float64SliceFlag{
			Name:        "float64-slice",
			Destination: testFloat64Slice,
			Value:       testFloat64Slice,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "int",
			Destination: &testInt,
			Value:       testInt,
		}),
		altsrc.NewIntSliceFlag(&cli.IntSliceFlag{
			Name:        "int-slice",
			Destination: testIntSlice,
			Value:       testIntSlice,
		}),
		altsrc.NewInt64SliceFlag(&cli.Int64SliceFlag{
			Name:        "int64-slice",
			Destination: testInt64Slice,
			Value:       testInt64Slice,
		}),
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name:        "string-slice",
			Destination: testStringSlice,
			Value:       testStringSlice,
		}),
		&cli.StringFlag{Name: "load"},
	}

	app := &cli.App{
		Action: func(cCtx *cli.Context) error {
			fmt.Printf("load is-set=%v context=%v\n", cCtx.IsSet("load"), cCtx.String("load"))

			fmt.Printf("float64-slice is-set=%v value=%+v context=%+v\n", cCtx.IsSet("float64-slice"), testFloat64Slice.Value(), cCtx.Float64Slice("float64-slice"))
			fmt.Printf("int is-set=%v value=%v context=%v\n", cCtx.IsSet("int"), testInt, cCtx.Int("int"))
			fmt.Printf("int-slice is-set=%v value=%v context=%v\n", cCtx.IsSet("int-slice"), testIntSlice.Value(), cCtx.IntSlice("int-slice"))
			fmt.Printf("int64-slice is-set=%v value=%v context=%v\n", cCtx.IsSet("int64-slice"), testInt64Slice.Value(), cCtx.Int64Slice("int64-slice"))
			fmt.Printf("string-slice is-set=%v value=%v context=%v\n", cCtx.IsSet("string-slice"), testStringSlice.Value(), cCtx.StringSlice("string-slice"))

			appState["ran"] = true
			return nil
		},
		Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("load")),
		Flags:  flags,
	}

	if err := app.Run([]string{"", "--load", "../testdata/slice-destinations.yaml"}); err != nil {
		log.Fatal(err)
	}

	if _, ok := appState["ran"]; !ok {
		log.Fatal("app did not run")
	}

	// Output:
	// float64-slice is-set=true value=[0.1 2.3 4567.89]
	// int is-set=true value=1 context=1
	// int-slice is-set=true value=[1 2 3] context=[1 2 3]
	// int64-slice is-set=true value=[10000000 999 1312] context=[10000000 999 1312]
	// string-slice is-set=true value=[a b c] context=[a b c]
}

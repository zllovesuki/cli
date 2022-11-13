package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3/internal/argh"
)

var (
	errCoerce = errors.New("coerce error")
)

// Context is a type that is passed through to
// each Handler action in a cli application. Context
// can be used to retrieve context-specific args and
// parsed command-line options.
type Context struct {
	context.Context

	App     *App
	Command *Command

	shellComplete bool
	parent        *Context
	flagSet       *argh.CommandConfig
}

type coerceFunc func(string) (any, error)

// NewContext creates a new context. For use in when invoking an App or Command action.
func NewContext(app *App, flagSet *argh.CommandConfig, parentCtx *Context) *Context {
	cCtx := &Context{App: app, flagSet: flagSet, parent: parentCtx}

	if parentCtx != nil {
		cCtx.Context = parentCtx.Context
		cCtx.shellComplete = parentCtx.shellComplete
	}

	cCtx.Command = &Command{}

	if cCtx.Context == nil {
		cCtx.Context = context.Background()
	}

	return cCtx
}

// NumFlags returns the number of flags set
func (cCtx *Context) NumFlags() int {
	return cCtx.flagSet.NFlag()
}

// Set sets a context flag to a value.
func (cCtx *Context) Set(name, value string) error {
	if _, flCfg := cCtx.lookupFlagSet(name); flCfg != nil {
		return flCfg.Set(value)
	}

	return fmt.Errorf("no such flag -%s", name)
}

// IsSet determines if the flag was actually set
func (cCtx *Context) IsSet(name string) bool {
	flCfg, ok := cCtx.flagSet.GetFlagConfig(name)
	return ok && flCfg.Node != nil
}

// LocalFlagNames returns a slice of flag names used in this context.
func (cCtx *Context) LocalFlagNames() []string {
	if cCtx.flagSet == nil {
		return []string{}
	}

	var names []string
	cCtx.flagSet.Visit(makeFlagNameVisitor(&names))
	// Check the flags which have been set via env or file
	if cCtx.Command != nil && cCtx.Command.Flags != nil {
		for _, f := range cCtx.Command.Flags {
			if f.IsSet() {
				names = append(names, f.Names()...)
			}
		}
	}

	// Sort out the duplicates since flag could be set via multiple
	// paths
	m := map[string]struct{}{}
	var unames []string
	for _, name := range names {
		if _, ok := m[name]; !ok {
			m[name] = struct{}{}
			unames = append(unames, name)
		}
	}

	return unames
}

// FlagNames returns a slice of flag names used by the this context and all of
// its parent contexts.
func (cCtx *Context) FlagNames() []string {
	var names []string
	for _, pCtx := range cCtx.Lineage() {
		names = append(names, pCtx.LocalFlagNames()...)
	}
	return names
}

// Lineage returns *this* context and all of its ancestor contexts in order from
// child to parent
func (cCtx *Context) Lineage() []*Context {
	var lineage []*Context

	for cur := cCtx; cur != nil; cur = cur.parent {
		lineage = append(lineage, cur)
	}

	return lineage
}

// Count returns the num of occurences of this flag
func (cCtx *Context) Count(name string) int {
	_, flCfg := cCtx.lookupFlagSet(name)
	if flCfg == nil || flCfg.Node == nil {
		return 0
	}

	return len(flCfg.Node.Values)
}

// Value returns the value of the flag corresponding to `name`
func (cCtx *Context) Value(name string) interface{} {
	_, flCfg := cCtx.lookupFlagSet(name)
	if flCfg == nil || flCfg.Node == nil {
		return nil
	}

	if vals := flCfg.Values(); len(vals) > 0 {
		return vals[0]
	}

	return nil
}

// Args returns the command line arguments associated with the context.
func (cCtx *Context) Args() Args {
	ret := args(cCtx.flagSet.Args())
	return &ret
}

// NArg returns the number of the command line arguments.
func (cCtx *Context) NArg() int {
	return cCtx.Args().Len()
}

func (cCtx *Context) lookupFlag(name string) Flag {
	for _, c := range cCtx.Lineage() {
		if c.Command == nil {
			continue
		}

		for _, f := range c.Command.Flags {
			for _, n := range f.Names() {
				if n == name {
					return f
				}
			}
		}
	}

	if cCtx.App != nil {
		for _, f := range cCtx.App.Flags {
			for _, n := range f.Names() {
				if n == name {
					return f
				}
			}
		}
	}

	return nil
}

func (cCtx *Context) lookupFlagSet(name string) (*argh.CommandConfig, *argh.FlagConfig) {
	for _, c := range cCtx.Lineage() {
		if c.flagSet == nil {
			continue
		}

		if flCfg := c.flagSet.Lookup(name); flCfg != nil {
			return c.flagSet, flCfg
		}
	}
	cCtx.onInvalidFlag(name)
	return nil, nil
}

func (cCtx *Context) checkRequiredFlags(flags []Flag) requiredFlagsErr {
	var missingFlags []string
	for _, f := range flags {
		if rf, ok := f.(RequiredFlag); ok && rf.IsRequired() {
			var flagPresent bool
			var flagName string

			for _, key := range f.Names() {
				flagName = key

				if cCtx.IsSet(strings.TrimSpace(key)) {
					flagPresent = true
				}
			}

			if !flagPresent && flagName != "" {
				missingFlags = append(missingFlags, flagName)
			}
		}
	}

	if len(missingFlags) != 0 {
		return &errRequiredFlags{missingFlags: missingFlags}
	}

	return nil
}

func (cCtx *Context) onInvalidFlag(name string) {
	for cCtx != nil {
		if cCtx.App != nil && cCtx.App.InvalidFlagAccessHandler != nil {
			cCtx.App.InvalidFlagAccessHandler(cCtx, name)
			break
		}
		cCtx = cCtx.parent
	}
}

func (cCtx *Context) lookupValue(flCfg *argh.FlagConfig, name string, f coerceFunc) (any, error) {
	if v, ok := flCfg.LookupValue(); ok {
		return f(v)
	}

	missErr := fmt.Errorf("no value with key %q", name)

	for c := cCtx; c.parent != nil; c = c.parent {
		if c.Command == nil {
			continue
		}

		for _, fl := range c.Command.Flags {
			if gvfl, ok := fl.(getValueAsAnyFlag); ok {
				for _, flName := range fl.Names() {
					if flName == name {
						return gvfl.getValueAsAny()
					}
				}
			}
		}
	}

	return nil, missErr
}

func makeFlagNameVisitor(names *[]string) func(*argh.CommandFlag) {
	return func(f *argh.CommandFlag) {
		nameParts := strings.Split(f.Name, ",")
		name := strings.TrimSpace(nameParts[0])

		for _, part := range nameParts {
			part = strings.TrimSpace(part)
			if len(part) > len(name) {
				name = part
			}
		}

		if name != "" {
			*names = append(*names, name)
		}
	}
}

package argh

import (
	"fmt"
	"sort"
	"sync"
)

const (
	OneOrMoreValue  NValue = -2
	ZeroOrMoreValue NValue = -1
	ZeroValue       NValue = 0
)

var (
	zeroValuePtr = func() *NValue {
		v := ZeroValue
		return &v
	}()
)

type NValue int

func (nv NValue) Required() bool {
	if nv == OneOrMoreValue {
		return true
	}

	return int(nv) >= 1
}

func (nv NValue) Contains(i int) bool {
	tracef("NValue.Contains(%v)", i)

	if i < int(ZeroValue) {
		return false
	}

	if nv == OneOrMoreValue || nv == ZeroOrMoreValue {
		return true
	}

	return int(nv) > i
}

type ParserConfig struct {
	Prog *CommandConfig

	ScannerConfig *ScannerConfig
}

type ParserOption func(*ParserConfig)

func NewParserConfig(opts ...ParserOption) *ParserConfig {
	pCfg := &ParserConfig{}

	for _, opt := range opts {
		if opt != nil {
			opt(pCfg)
		}
	}

	if pCfg.Prog == nil {
		pCfg.Prog = NewCommandConfig()
	}

	if pCfg.ScannerConfig == nil {
		pCfg.ScannerConfig = POSIXyScannerConfig
	}

	return pCfg
}

type CommandConfig struct {
	NValue     NValue
	ValueNames []string
	Flags      *Flags
	Commands   *Commands
	Node       *CommandFlag
}

func NewCommandConfig() *CommandConfig {
	cCfg := &CommandConfig{}
	cCfg.init()

	return cCfg
}

func (cCfg *CommandConfig) init() {
	if cCfg.ValueNames == nil {
		cCfg.ValueNames = []string{}
	}

	if cCfg.Flags == nil {
		cCfg.Flags = &Flags{}
	}

	if cCfg.Commands == nil {
		cCfg.Commands = &Commands{}
	}
}

// NFlag works like flag.FlagSet.NFlag
func (cCfg *CommandConfig) NFlag() int {
	n := 0

	for _, flCfg := range cCfg.Flags.Map {
		if flCfg.Node != nil {
			n++
		}
	}

	return n
}

// Visit works like flag.FlagSet.Visit
func (cCfg *CommandConfig) Visit(f func(*CommandFlag)) {
	names := make([]string, len(cCfg.Flags.Map))

	i := 0
	for name := range cCfg.Flags.Map {
		names[i] = name
		i++
	}

	sort.Strings(names)

	for _, name := range names {
		if flCfg, ok := cCfg.Flags.GetShallow(name); ok && flCfg.Node != nil {
			f(flCfg.Node)
		}
	}
}

// Args is like flag.FlagSet.Args
func (cCfg *CommandConfig) Args() []string {
	ret := []string{}

	if cCfg.Node == nil {
		return ret
	}

	for _, node := range cCfg.Node.Nodes {
		if p, ok := node.(*PassthroughArgs); ok {
			for _, ptNode := range p.Nodes {
				if idn, ok := ptNode.(*Ident); ok {
					ret = append(ret, idn.Literal)
				}
			}
		}
	}

	return ret
}

// Lookup is like flag.FlagSet.Lookup
func (cCfg *CommandConfig) Lookup(name string) *FlagConfig {
	if flCfg, ok := cCfg.Flags.GetShallow(name); ok {
		return flCfg
	}

	return nil
}

// Set is like flag.FlagSet.Set
func (cCfg *CommandConfig) Set(name, value string) error {
	cCfg.SetFlagConfig(
		name,
		&FlagConfig{
			Node: &CommandFlag{
				Name:   name,
				Values: map[string]string{name: value},
			},
		},
	)

	return nil
}

func (cCfg *CommandConfig) GetCommandConfig(name string) (CommandConfig, bool) {
	tracef("CommandConfig.GetCommandConfig(%q)", name)

	if cCfg.Commands == nil {
		cCfg.Commands = &Commands{Map: map[string]CommandConfig{}}
	}

	return cCfg.Commands.Get(name)
}

func (cCfg *CommandConfig) GetFlagConfig(name string) (*FlagConfig, bool) {
	tracef("CommandConfig.GetFlagConfig(%q)", name)

	if cCfg.Flags == nil {
		cCfg.Flags = &Flags{Map: map[string]FlagConfig{}}
	}

	return cCfg.Flags.Get(name)
}

func (cCfg *CommandConfig) SetFlagConfig(name string, flCfg *FlagConfig) {
	tracef("CommandConfig.SetFlagConfig(%q, ...)", name)

	if cCfg.Flags == nil {
		cCfg.Flags = &Flags{Map: map[string]FlagConfig{}}
	}

	cCfg.Flags.Set(name, flCfg)
}

func (cCfg *CommandConfig) SetDefaultFlagConfig(name string, flCfg *FlagConfig) {
	tracef("CommandConfig.SetDefaultFlagConfig(%q, ...)", name)

	if cCfg.Flags == nil {
		cCfg.Flags = &Flags{Map: map[string]FlagConfig{}}
	}

	cCfg.Flags.SetDefault(name, flCfg)
}

type FlagConfig struct {
	NValue     NValue
	Persist    bool
	ValueNames []string
	Node       *CommandFlag
}

func (flCfg *FlagConfig) Set(value string) error {
	if flCfg.Node == nil {
		if len(flCfg.ValueNames) == 0 {
			return fmt.Errorf("cannot set value of flag without name: %w", Error)
		}

		flCfg.Node = &CommandFlag{
			Name: flCfg.ValueNames[0],
		}
	}

	flCfg.Node.Values[flCfg.Node.Name] = value

	return nil
}

func (flCfg *FlagConfig) Name() string {
	if flCfg.Node == nil {
		return ""
	}

	return flCfg.Node.Name
}

// Value is like flag.Value.String (see alias)
func (flCfg *FlagConfig) Value() string {
	if vals := flCfg.Values(); len(vals) > 0 {
		return vals[0]
	}

	return ""
}

func (flCfg *FlagConfig) String() string { return flCfg.Value() }

func (flCfg *FlagConfig) Values() []string {
	if flCfg.Node == nil {
		return []string{}
	}

	values := make([]string, len(flCfg.Node.Values))

	i := 0
	for _, value := range flCfg.Node.Values {
		values[i] = value
		i++
	}

	return values
}

type Flags struct {
	Parent *Flags
	Map    map[string]FlagConfig

	Automatic bool

	m sync.Mutex
}

func (fl *Flags) GetShallow(name string) (*FlagConfig, bool) {
	if fl.Map == nil {
		fl.Map = map[string]FlagConfig{}
	}

	flCfg, ok := fl.Map[name]
	return &flCfg, ok
}

func (fl *Flags) Get(name string) (*FlagConfig, bool) {
	tracef("Flags.Get(%q)", name)

	if fl.Map == nil {
		fl.Map = map[string]FlagConfig{}
	}

	flCfg, ok := fl.Map[name]
	if !ok {
		if fl.Automatic {
			return &FlagConfig{}, true
		}

		if fl.Parent != nil {
			v, ok := fl.Parent.Get(name)
			return v, ok && v.Persist
		}
	}

	return &flCfg, ok
}

func (fl *Flags) Set(name string, flCfg *FlagConfig) {
	tracef("Flags.Set(%q, ...)", name)

	fl.m.Lock()
	defer fl.m.Unlock()

	if fl.Map == nil {
		fl.Map = map[string]FlagConfig{}
	}

	fl.Map[name] = *flCfg
}

func (fl *Flags) SetDefault(name string, flCfg *FlagConfig) {
	tracef("Flags.SetDefault(%q, ...)", name)

	fl.m.Lock()
	defer fl.m.Unlock()

	if fl.Map == nil {
		fl.Map = map[string]FlagConfig{}
	}

	if _, ok := fl.Map[name]; !ok {
		fl.Map[name] = *flCfg
	}
}

type Commands struct {
	Map map[string]CommandConfig
}

func (cmd *Commands) Get(name string) (CommandConfig, bool) {
	tracef("Commands.Get(%q)", name)

	if cmd.Map == nil {
		cmd.Map = map[string]CommandConfig{}
	}

	cmdCfg, ok := cmd.Map[name]
	return cmdCfg, ok
}

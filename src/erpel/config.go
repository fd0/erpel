package erpel

import (
	"gopkg.in/gcfg.v1"

	"github.com/fd0/probe"
)

// ConfigFile reads configuration data for erpel from a config file.
type ConfigFile struct {
	Global struct {
		RulesDir string
		Prefix   string
	}

	Aliases map[string]string
}

// LoadConfigFile unmarshals the configuration contained in the file.
func LoadConfigFile(filename string) (*ConfigFile, error) {
	cfg := ConfigFile{}
	err := gcfg.ReadFileInto(&cfg, filename)
	if err != nil {
		return nil, probe.Trace(err, filename)
	}

	// err = hcl.Unmarshal(buf, cfg)
	// if err != nil {
	// 	return nil, probe.Trace(err, string(buf))
	// }

	// cfg.Prefix = strings.TrimSpace(cfg.Prefix)
	// if cfg.Prefix != "" {
	// 	// add begining-of-line matching if not already present
	// 	if cfg.Prefix[0] != '^' {
	// 		cfg.Prefix = "^" + cfg.Prefix
	// 	}

	// 	r, err := regexp.Compile(cfg.Prefix)
	// 	if err != nil {
	// 		return nil, probe.Trace(err, cfg.Prefix)
	// 	}

	// 	cfg.prefix = r
	// } else {
	// 	cfg.prefix = regexp.MustCompile("^")
	// }

	return &cfg, nil
}

// LoadAllRules loads all rules from files in dir.
// func LoadAllRules(cfg *Config) (rules []*regexp.Regexp, err error) {
// 	pattern := filepath.Join(cfg.RulesDir, "*")
// 	matches, err := filepath.Glob(pattern)
// 	if err != nil {
// 		return nil, probe.Trace(err, pattern)
// 	}

// 	for _, file := range matches {
// 		r, err := LoadRules(cfg, file)
// 		if err != nil {
// 			return nil, probe.Trace(err, file)
// 		}

// 		rules = append(rules, r...)
// 	}

// 	return rules, nil
// }

// LoadRules unmarshals a rules file.
// func LoadRules(cfg *Config, filename string) (rules []*regexp.Regexp, err error) {
// 	f, err := os.Open(filename)
// 	if err != nil {
// 		return nil, probe.Trace(err, filename)
// 	}

// 	defer func() {
// 		e := probe.Trace(f.Close())
// 		if err == nil {
// 			err = e
// 		}
// 	}()

// 	sc := bufio.NewScanner(f)
// 	currentLine := 0

// 	for sc.Scan() {
// 		currentLine++
// 		line := strings.TrimSpace(sc.Text())

// 		// filter out comments and empty lines
// 		if len(line) == 0 || line[0] == '#' {
// 			continue
// 		}

// 		// add beginning-of-line matching if not already present
// 		if line[0] != '^' {
// 			line = "^" + line
// 		}

// 		// replace aliases
// 		// for _, alias := range cfg.Aliases {
// 		// 	line = strings.Replace(line, alias.Name, alias.Regex, -1)
// 		// }

// 		r, err := regexp.Compile(line)
// 		if err != nil {
// 			return nil, probe.Trace(err, filename, currentLine, sc.Text())
// 		}

// 		rules = append(rules, r)
// 	}

// 	return rules, nil
// }

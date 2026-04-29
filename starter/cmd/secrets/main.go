package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	cfgpkg "github.com/go-sum/foundry/pkg/config"
	"gopkg.in/yaml.v3"
)

type composeFile struct {
	Secrets map[string]any `yaml:"secrets"`
}

func run(composeFiles []string, envFile, outDir string) error {
	// Collect secret names from all compose files.
	seen := make(map[string]struct{})
	for _, path := range composeFiles {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("open %s: %w", path, err)
		}
		var cf composeFile
		if err := yaml.NewDecoder(f).Decode(&cf); err != nil {
			f.Close() //nolint:errcheck
			return fmt.Errorf("parse %s: %w", path, err)
		}
		f.Close() //nolint:errcheck
		for name := range cf.Secrets {
			seen[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	// Parse .env file.
	env := make(map[string]string)
	ef, err := os.Open(envFile)
	if err != nil {
		return fmt.Errorf("open %s: %w", envFile, err)
	}
	defer ef.Close() //nolint:errcheck

	scanner := bufio.NewScanner(ef)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := line[idx+1:]
		env[key] = val
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", envFile, err)
	}
	cfgpkg.ExtractDSNComponents(env, seen)

	if err := os.MkdirAll(outDir, 0700); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	var missing []string
	written := 0
	for _, name := range names {
		val, ok := env[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "warning: secret %q not found in %s\n", name, envFile)
			missing = append(missing, name)
			continue
		}
		dest := outDir + "/" + name
		if err := os.WriteFile(dest, []byte(val), 0600); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
		written++
	}

	fmt.Printf("wrote %d secret file(s) to %s\n", written, outDir)

	if len(missing) > 0 {
		return fmt.Errorf("%d secret(s) missing from %s: %s", len(missing), envFile, strings.Join(missing, ", "))
	}
	return nil
}

func main() {
	envFile := flag.String("env", ".env", "path to .env file")
	outDir := flag.String("dir", ".secrets", "directory to write secret files into")
	flag.Parse()

	composeFiles := flag.Args()
	if len(composeFiles) == 0 {
		composeFiles = []string{"docker-compose.data.yml", "docker-compose.yml"}
	}

	if err := run(composeFiles, *envFile, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

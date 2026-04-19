package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/yaml"

	"github.com/khvedela/kubediag/internal/cli"
	"github.com/khvedela/kubediag/internal/config"
)

func newConfigCmd(_ *viper.Viper) *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "View or initialize kubediag configuration",
	}
	c.AddCommand(newConfigViewCmd(), newConfigInitCmd(), newConfigPathCmd())
	return c
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Print the resolved configuration with provenance",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := cli.Get(cmd)
			cfg := opts.Config
			cfg.Provenance = nil // exclude from YAML body; print separately

			b, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "# resolved configuration")
			fmt.Fprint(cmd.OutOrStdout(), string(b))

			fmt.Fprintln(cmd.OutOrStdout(), "\n# provenance (which source supplied each key)")
			prov := opts.Config.Provenance
			keys := make([]string, 0, len(prov))
			for k := range prov {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "#   %-24s %s\n", k, prov[k])
			}
			return nil
		},
	}
}

func newConfigInitCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Write a commented config template to the default location",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := config.DefaultPath()
			if path == "" {
				return fmt.Errorf("could not determine config directory (set $XDG_CONFIG_HOME or $HOME)")
			}
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s already exists; pass --force to overwrite", path)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(config.Template()), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			return nil
		},
	}
	c.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	return c
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the default config file path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := config.DefaultPath()
			if p == "" {
				return fmt.Errorf("could not determine config directory")
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
			return nil
		},
	}
}

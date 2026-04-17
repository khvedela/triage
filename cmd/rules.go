package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/khvedela/triage/internal/findings"
	"github.com/khvedela/triage/internal/rules"
)

func newRulesCmd() *cobra.Command {
	c := &cobra.Command{Use: "rules", Short: "List and explain built-in rules"}
	c.AddCommand(newRulesListCmd(), newRulesExplainCmd())
	return c
}

func newRulesListCmd() *cobra.Command {
	var category, severity string
	c := &cobra.Command{
		Use:   "list",
		Short: "List all built-in rules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSEVERITY\tCATEGORY\tSCOPES\tTITLE")
			for _, r := range rules.All() {
				m := r.Meta()
				if category != "" && !strings.EqualFold(string(m.Category), category) {
					continue
				}
				if severity != "" && !strings.EqualFold(string(m.Severity), severity) {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					m.ID, m.Severity, m.Category, formatScopes(m.Scopes), m.Title)
			}
			return w.Flush()
		},
	}
	c.Flags().StringVar(&category, "category", "", "filter by category")
	c.Flags().StringVar(&severity, "severity", "", "filter by severity")
	return c
}

func newRulesExplainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "explain <rule-id>",
		Short: "Print full documentation for a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			r := rules.Get(id)
			if r == nil {
				return fmt.Errorf("rule %q not found; try `triage rules list`", id)
			}
			m := r.Meta()
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "%s — %s\n", m.ID, m.Title)
			fmt.Fprintf(w, "  category:   %s\n", m.Category)
			fmt.Fprintf(w, "  severity:   %s (default)\n", m.Severity)
			fmt.Fprintf(w, "  scopes:     %s\n", formatScopes(m.Scopes))
			fmt.Fprintf(w, "  priority:   %d\n", m.Priority)
			if m.Description != "" {
				fmt.Fprintf(w, "\n%s\n", m.Description)
			}
			if len(m.DocsLinks) > 0 {
				fmt.Fprintln(w, "\nDocs:")
				for _, l := range m.DocsLinks {
					fmt.Fprintf(w, "  - %s\n", l)
				}
			}
			return nil
		},
	}
}

func formatScopes(scopes []findings.TargetKind) string {
	s := make([]string, len(scopes))
	for i, x := range scopes {
		s[i] = string(x)
	}
	return strings.Join(s, ",")
}

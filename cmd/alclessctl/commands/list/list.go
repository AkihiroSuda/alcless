package list

import (
	"encoding/json"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/AkihiroSuda/alcless/pkg/store"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "list",
		Aliases:               []string{"ls"},
		Short:                 "List instances",
		Args:                  cobra.NoArgs,
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.Bool("json", false, "jsonify output")
	flags.BoolP("quiet", "q", false, "only show names")
	return cmd
}

func action(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	stdout := cmd.OutOrStdout()
	flags := cmd.Flags()
	flagJson, err := flags.GetBool("json")
	if err != nil {
		return err
	}
	flagQuiet, err := flags.GetBool("quiet")
	if err != nil {
		return err
	}
	if flagJson && flagQuiet {
		return errors.New("option --json conflicts with option --quiet")
	}
	insts, err := store.Instances(ctx)
	if err != nil {
		return err
	}
	switch {
	case flagJson:
		// single JSON object per line (similar to `limactl ls --quiet`)
		enc := json.NewEncoder(stdout)
		enc.SetEscapeHTML(false)
		for _, b := range insts {
			if err = enc.Encode(b); err != nil {
				return err
			}
		}
	case flagQuiet:
		for _, b := range insts {
			if _, err = fmt.Fprintln(stdout, b.Name); err != nil {
				return err
			}
		}
	default:
		w := tabwriter.NewWriter(stdout, 4, 8, 4, ' ', 0)
		fmt.Fprintln(w, "NAME\tUSER")
		for _, b := range insts {
			if _, err = fmt.Fprintf(w, "%s\t%s\n", b.Name, b.User); err != nil {
				return err
			}
		}
		if err = w.Flush(); err != nil {
			return err
		}
	}

	return nil
}

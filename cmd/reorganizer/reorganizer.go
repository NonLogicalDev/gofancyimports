package reorganizer

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	gofancyimports "github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/internal/diff"
)

type ArgHook func(cmd *cobra.Command) func(cmd *cobra.Command, args []string)

type Config struct {
	commandName string
	argHook     ArgHook
}

type Option func(c *Config)

func WithArgHook(hook ArgHook) Option {
	return func(c *Config) {
		c.argHook = hook
	}
}

func WithCommandName(name string) Option {
	return func(c *Config) {
		c.commandName = name
	}
}

func RunCmd(ctx context.Context, rewriter gofancyimports.ImportOrganizer, opts ...Option) error {
	cfg := &Config{
		commandName: "go-fancy-imports",
	}
	for _, o := range opts {
		o(cfg)
	}

	cmd := &cobra.Command{
		Use: cfg.commandName,
		Args: cobra.MinimumNArgs(1),
	}
	cmdFlags := cmd.PersistentFlags()

	var (
		flagWrite = cmdFlags.
			BoolP("write", "w", false, "write the file back?")
		showDiff  = cmdFlags.
			BoolP("diff", "d", false, "print diff")
		//localPrefixes = cmdFlags.
		//	StringArrayP("local", "l", nil, "list of local prefixes")
	)

	var execHook func(cmd *cobra.Command, args []string)
	if cfg.argHook != nil {
		execHook = cfg.argHook(cmd)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		for _, sourcePath := range args {
			if execHook != nil {
				execHook(cmd, args)
			}

			src, err := os.ReadFile(sourcePath)
			if err != nil { return err }

			var newSrc []byte

			newSrc, err = gofancyimports.RewriteImports(sourcePath, src, rewriter)
			if err != nil { return err }

			// Print diff.
			if *showDiff {
				diffOut, err := diff.Diff("", src, newSrc)
				if err != nil { return err }

				diffOut, err = diff.ReplaceTempFilename(diffOut, sourcePath)
				if err != nil { return err }

				fmt.Println(string(diffOut))
				continue
			}

			// Print source.
			if !*flagWrite {
				fmt.Println(">>> ", sourcePath)
				fmt.Println(string(newSrc))
				continue
			}

			// Write back.
			err = os.WriteFile(sourcePath, newSrc, 0x666)
			if err != nil {
				return err
			}

			fmt.Println("Written:", sourcePath)
		}

		return nil
	}

	return cmd.ExecuteContext(ctx)
}

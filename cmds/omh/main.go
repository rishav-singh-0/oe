package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	omh "github.com/rishav-singh-0/oe/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/thlib/go-timezone-local/tzlocal"
	cli "github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "omh",
		Version:               "0.0.1",
		Description:           "Command line tool to export Obsidian Vault to Hugo",
		Authors:               []any{"Rishav Singh <rsh04613@gmail.com>"},
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "obsidian-root",
				Aliases:  []string{"O"},
				Required: true,
				Value:    "vault",
				Usage:    "Path to root of Obsidian Vault",
			},
			&cli.StringFlag{
				Name:     "hugo-root",
				Aliases:  []string{"H"},
				Required: true,
				Usage:    "Path to root of Hugo setup",
			},
			&cli.StringFlag{
				Name:    "sub-path",
				Aliases: []string{"p"},
				Usage:   "Sub-path used in Hugo setup below content and static",
				Value:   "posts",
			},
			&cli.StringSliceFlag{
				Name:    "include-tag",
				Aliases: []string{"i"},
				Usage:   "Tag to include (accept list - accepts all, if unset)",
			},
			&cli.StringSliceFlag{
				Name:    "exclude-tag",
				Aliases: []string{"e"},
				Usage:   "Tag to exclude (reject list - reject none, if unset)",
			},
			&cli.StringSliceFlag{
				Name:    "publish-field",
				Aliases: []string{"P"},
				Usage:   "Only include Field/Frontmatter (include all if unset)",
			},
			&cli.StringSliceFlag{
				Name:    "front-matter",
				Aliases: []string{"F"},
				Usage:   "Additional Front Matter, added to all generated Hugo pages, in the form `key:value`",
			},
			&cli.StringFlag{
				Name:    "tags-key",
				Aliases: []string{"t"},
				Usage:   "Name of Front Matter attribute to use for tags (so that taxonomy in Hugo can be used)",
				Value:   "tags",
			},
			&cli.BoolFlag{
				Name:    "recursive",
				Aliases: []string{"R"},
				Usage:   "Whether to recurse the Obsidian Root directory (or not and then ignore sub directories..)",
			},
			&cli.StringFlag{
				Name:    "time-zone",
				Aliases: []string{"z"},
				Usage:   "The time zone all output dates should have",
				Value:   loadTimeZone(),
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"D"},
				Usage:   "Enable debug logs",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {

			if cmd.Bool("debug") {
				log.SetLevel(log.DebugLevel)
				// Print all arguments
				for i, v := range cmd.FlagNames() {
					fmt.Printf("%d-%s %#v\n", i, v, cmd.Value(v))
				}
			}

			timeZone, err := time.LoadLocation(cmd.String("time-zone"))
			if err != nil {
				return fmt.Errorf("failed to parse time zone: %w", err)
			}
			omh.TimeZone = timeZone

			recurse := cmd.Bool("recursive")
			directory, err := omh.LoadObsidianDirectory(cmd.String("obsidian-root"), createFilter(cmd), recurse)
			if err != nil {
				return err
			}

			// is there additional front matter?
			addFrontMatter := make(map[string]interface{})
			for _, matter := range cmd.StringSlice("front-matter") {
				kv := strings.SplitN(matter, ":", 2)
				addFrontMatter[kv[0]] = kv[1]
			}

			converter := &omh.Converter{
				ObsidianRoot: directory,
				HugoRoot:     cmd.String("hugo-root"),
				SubPath:      cmd.String("sub-path"),
				FrontMatter:  addFrontMatter,
				ConvertName: func(name string) (link string) {
					return omh.Sanitize(strcase.ToKebab(name))
				},
				TagsKey: cmd.String("tags-key"),
			}

			return converter.Run()
		},
		Arguments: cli.AnyArguments,
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func createFilter(c *cli.Command) omh.ObsidianFilter {
	filters := make([]omh.ObsidianFilter, 0)

	// Filter notes to include only those with tags matching the "include-tag" list
	if includes := c.StringSlice("include-tag"); len(includes) > 0 {
		// Convert the list of included tags into a map for quick lookup
		included := strsToBoolMap(includes)

		// Append the filter that checks if any tag in the note matches the included tags
		filters = append(filters, func(note omh.ObsidianNote) bool {
			// Iterate over each tag in the note's front matter
			for _, tag := range note.FrontMatter.Strings("tags") {
				// If the tag exists in the included map, keep the note
				if included[tag] {
					return true
				}
			}
			// Exclude the note if none of its tags match the included list
			return false
		})
	}

	if fields := c.StringSlice("publish-field"); len(fields) > 0 {
		filters = append(filters, func(note omh.ObsidianNote) bool {
			for _, field := range fields {
				if note.FrontMatter.Has(field) {
					return true
				}
			}
			return false
		})
	}

	// Filter notes to exclude those with tags matching the "exclude-tag" list
	if excludes := c.StringSlice("exclude-tag"); len(excludes) > 0 {
		// Convert the list of excluded tags into a map for quick lookup
		excluded := strsToBoolMap(excludes)

		// Append the filter that checks if any tag in the note matches the excluded tags
		filters = append(filters, func(note omh.ObsidianNote) bool {
			// Iterate over each tag in the note's front matter
			for _, tag := range note.FrontMatter.Strings("tags") {
				// If the tag exists in the excluded map, discard the note
				if excluded[tag] {
					return false
				}
			}
			// Keep the note if none of its tags match the excluded list
			return true
		})
	}

	if len(filters) == 0 {
		return nil
	}

	return func(note omh.ObsidianNote) bool {
		for _, f := range filters {
			if !f(note) {
				return false
			}
		}
		return true
	}
}

func loadTimeZone() string {
	tz, err := tzlocal.RuntimeTZ()
	if err != nil {
		return "UTC"
	}
	return tz
}

func strsToBoolMap(strs []string) map[string]bool {
	r := make(map[string]bool)
	for _, str := range strs {
		r[str] = true
	}
	return r
}

func todo() {
	fmt.Println("Obsidian Meets Hugo")
	fmt.Println("  Command line tool to export (partial) Obsidian Vault to Hugo")
	fmt.Println("Input:")
	fmt.Println("  - Obsidian Directory: Path to root of Obsidian Vault")
	fmt.Println("  - Hugo Directory: Path to root of Hugo setup")
	fmt.Println("    - Sub-Path, default `obsidian`, used in `content/<sub-path>` and `static/<sub-path>`")
	fmt.Println("  - Optional Tag include list")
	fmt.Println("  - Optional Tag exclude list")
	fmt.Println("Execution:")
	fmt.Println("  - Find all Markdown files in Obsidian Directory and Subdirectories")
	fmt.Println("    - Copy and Transform from Obsidian Note into Hugo Page in `<hugo-root>/content/<sub-path>`")
	fmt.Println("      - Make file name snake-case")
	fmt.Println("      - Replace all internal links, so that they work in Hugo (point to snake case, respective sub-path in content)")
	fmt.Println("      - Replace all internal references to non-Markdown files with appropriate Markdown")
	fmt.Println("  - Find all none-Markdown files and copy them to `<hugo-root>/static/<sub-path>")
}

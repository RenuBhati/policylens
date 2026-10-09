package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"policylens/sdk"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, stderr io.Writer) int {
	f := flag.NewFlagSet("policylens", flag.ContinueOnError)
	f.SetOutput(stderr)
	url := f.String("url", "http://127.0.0.1:8080", "API base URL")
	asJSON := f.Bool("json", false, "print JSON output")
	if err := f.Parse(args); err != nil {
		return 2
	}
	rest := f.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "Usage: policylens [-url URL] [-json] policies | ask QUESTION | check FILE")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()
	client := sdk.New(*url)
	printJSON := func(value any) { enc := json.NewEncoder(out); enc.SetIndent("", "  "); _ = enc.Encode(value) }
	switch rest[0] {
	case "policies":
		if len(rest) != 1 {
			return 2
		}
		items, err := client.Policies(ctx)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			printJSON(items)
		} else {
			for _, p := range items {
				fmt.Fprintf(out, "%-32s %s\n", p.ID, p.Title)
			}
		}
	case "ask":
		if len(rest) < 2 {
			fmt.Fprintln(stderr, "Provide a question.")
			return 2
		}
		answer, err := client.Ask(ctx, strings.Join(rest[1:], " "), "")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			printJSON(answer)
		} else {
			fmt.Fprintf(out, "Mode: %s\n\n%s\n\nSources: %s\n", answer.Mode, answer.Text, strings.Join(answer.Citations, ", "))
		}
	case "check":
		if len(rest) != 2 {
			fmt.Fprintln(stderr, "Provide one YAML file.")
			return 2
		}
		file, err := os.Open(rest[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, (64<<10)+1))
		if err != nil || len(data) > 64<<10 {
			fmt.Fprintln(stderr, "Cannot read manifest or file exceeds 64 KiB.")
			return 1
		}
		result, err := client.Check(ctx, string(data))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if *asJSON {
			printJSON(result)
		} else {
			fmt.Fprintf(out, "%s · %s\n", result.Engine, result.Resource)
			for _, r := range result.Results {
				fmt.Fprintf(out, "%-7s %-32s %s\n", strings.ToUpper(r.Status), r.PolicyID, r.Message)
			}
		}
		if result.Summary.Fail > 0 || result.Summary.Error > 0 || result.Summary.Skipped > 0 {
			return 1
		}
	default:
		fmt.Fprintln(stderr, "Unknown command:", rest[0])
		return 2
	}
	return 0
}

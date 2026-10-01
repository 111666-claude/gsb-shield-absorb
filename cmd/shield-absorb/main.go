// Command shield-absorb 跑护盾吸收样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/shield"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("shield-absorb", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "overflow", "overflow / expire / priority / negative")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "overflow":
		absorber := shield.NewAbsorber()
		absorber.Add("p", shield.Shield{ID: "a", Amount: 30, ExpireMs: 1000, Priority: 2})
		absorber.Add("p", shield.Shield{ID: "b", Amount: 30, ExpireMs: 1000, Priority: 1})
		absorbed, remaining := absorber.Apply("p", 100, 0)
		fmt.Fprintf(stdout, "absorbed=%.0f remaining=%.0f\n", absorbed, remaining)
	case "expire":
		absorber := shield.NewAbsorber()
		absorber.Add("p", shield.Shield{ID: "a", Amount: 50, ExpireMs: 1000, Priority: 1})
		absorbed, remaining := absorber.Apply("p", 20, 5000)
		fmt.Fprintf(stdout, "absorbed=%.0f remaining=%.0f\n", absorbed, remaining)
	case "priority":
		absorber := shield.NewAbsorber()
		absorber.Add("p", shield.Shield{ID: "a", Amount: 10, ExpireMs: 1000, Priority: 2})
		absorber.Add("p", shield.Shield{ID: "b", Amount: 50, ExpireMs: 1000, Priority: 1})
		absorber.Apply("p", 30, 0)
		fmt.Fprintf(stdout, "a=%.0f b=%.0f\n", absorber.AmountOf("p", "a"), absorber.AmountOf("p", "b"))
	case "negative":
		absorber := shield.NewAbsorber()
		absorber.Add("p", shield.Shield{ID: "a", Amount: 30, ExpireMs: 1000, Priority: 1})
		absorber.Apply("p", 1000, 0)
		fmt.Fprintf(stdout, "a=%.0f\n", absorber.AmountOf("p", "a"))
	default:
		fmt.Fprintln(stderr, "需要 --sample overflow|expire|priority|negative")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

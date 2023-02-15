package main

import (
	"fmt"
	"github.com/ZenLiuCN/fn"
	. "github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strings"
)

var (
	Version = "1.0.0"
)

func main() {
	AppHelpTemplate = `{{template "helpNameTemplate" .}} .ver {{.Version}}
usage: {{.HelpName}} {{if .VisibleFlags}}[options] {{end}}{{if .ArgsUsage}}{{.ArgsUsage}}{{end}}
options:{{template "visibleFlagTemplate" .}}
about:
   {{template "descriptionTemplate" .}}
`
	app := App{
		HideHelpCommand: true,
		Suggest:         true,
		Version:         Version,
		Name:            strings.TrimSuffix(filepath.Base(fn.Panic1(os.Executable())), ".exe"),
		Usage:           "The directory clean tool",
		ArgsUsage:       "<directory> [directories...]",
		Description: `Cleaner a tool clean directories which may contains caches or temporary files.
THOSE files '.gitignore .cleanignore' are treat as configs,global/user config can be exists under $HOME/.cleaner, {ExecPath}/.cleaner,
config priority: .cleanignore>.gitignore>USER>GLOBAL,all config are using same syntax and mean as .gitignore;
`,
		Flags: FlagsByName{
			&BoolFlag{Name: "exec", Aliases: strs("e"), Usage: "execute to filesystem"},
			&BoolFlag{Name: "debug", Aliases: strs("d"), Usage: "print debug info"},
			&PathFlag{Name: "output", Aliases: strs("o"), Usage: "output list to `file`"},
			&PathFlag{Name: "input", Aliases: strs("i"), Usage: "use input list `file`"},
			&StringSliceFlag{Name: "extra", Aliases: strs("x"), Usage: "extra lists in gitignore `pattern` form, has highest priority."},
		},
		Action: func(c *Context) (err error) {
			dir := c.Args().Slice()
			if len(dir) == 0 {
				return ShowAppHelp(c)
			}
			dbg := c.Bool("d")
			in := c.Path("i")
			p := c.Bool("p")
			if len(in) > 0 && c.Bool("p") {
				b, e := os.ReadFile(in)
				if e != nil {
					return e
				}
				purify(dbg, strings.Split(string(b), "\n"))
				return nil
			}
			out := c.Path("o")
			var o *os.File
			if out != "" {
				o, err = os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
				if err != nil {
					return
				}
			}
			conf := Load(dbg)
			{
				ek := c.StringSlice("x")
				if len(ek) > 0 {
					conf.Append(ek)
				}
			}
			(&conf).Debug = dbg
			var r []string
			for _, s := range dir {
				s, err = filepath.Abs(s)
				if err != nil {
					return err
				}
				r = append(r, conf.Matches(s)...)
			}
			if o != nil {
				for _, s := range r {
					_, _ = o.WriteString(s)
					_, _ = o.Write([]byte{'\n'})
				}
				fn.Panic(o.Close())
				o = nil
			}
			if p {
				purify(dbg, r)
				if out == "" {
					fmt.Printf("cleaned %d entries", len(r))
				}
			} else if out == "" {
				fmt.Println("clean targets:")
				for _, s := range r {
					fmt.Println(s)
				}
			}
			return
		},
	}
	fn.Panic(app.Run(os.Args))
}

//go:inline
func strs(s ...string) []string {
	return s
}
func purify(debug bool, list []string) {
	for _, s := range list {
		if debug {
			fmt.Println("purify " + s)
		}
		err := os.RemoveAll(s)
		if debug && err != nil {
			fmt.Println("purify " + s + " with error: " + err.Error())
		}
	}
}

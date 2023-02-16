package main

import (
	"errors"
	"fmt"
	"github.com/ZenLiuCN/fn"
	. "github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	name := strings.TrimSuffix(filepath.Base(fn.Panic1(os.Executable())), ".exe")
	app := App{
		HideHelpCommand: true,
		Suggest:         true,
		Version:         Version,
		Name:            name,
		Usage:           "The directory clean tool",
		ArgsUsage:       "<directory> [directories...]",
		Description: name + ` a tool clean directories which may contains caches or temporary files.
THOSE files '.gitignore .cleanignore' are treat as configs,GLOBAL/USER config can be exists under $HOME/.cleanignore, {ExecPath}/.cleanignore.
GLOBAL/USER config only effect on final level which means 'abc/efg' will no effect in GLOBAL/USER config;
PRIORITY: USER>GLOBAL>.cleanignore>.gitignore,all config are using same syntax and mean as .gitignore;
Note: .cleanignore's ignored files (without '!' prefix) means to be cleaned.
`,
		Flags: FlagsByName{
			&BoolFlag{Name: "exec", Aliases: strs("e"), Usage: "execute to filesystem"},
			&UintFlag{Name: "debug", Aliases: strs("d"), Usage: "print debug: 0 none, 1 match only, 2 match and keep only, else all "},
			&BoolFlag{Name: "hit", Aliases: strs("t"), Usage: "with hit info,can't use with -e -i"},
			&PathFlag{Name: "output", Aliases: strs("o"), Usage: "output list to `file`"},
			&PathFlag{Name: "input", Aliases: strs("i"), Usage: "use input list `file`"},
			&StringSliceFlag{Name: "extra", Aliases: strs("x"), Usage: "extra lists in gitignore `pattern` form, has highest priority."},
		},
		Action: func(c *Context) (err error) {
			dir := c.Args().Slice()
			if len(dir) == 0 {
				return ShowAppHelp(c)
			}
			dbg := c.Uint("d")
			hit := c.Bool("t")
			in := c.Path("i")
			p := c.Bool("p")
			if hit && (in != "" || p) {
				return errors.New("hit can not use with -e | -p | -i")
			}
			if len(in) > 0 && p {
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
			be := time.Now()
			defer func() {
				ed := time.Now()
				fmt.Println("End: ", ed.Format("15:01:05"))
				fmt.Printf("Time: %d seconds\n", ed.Unix()-be.Unix())
			}()
			fmt.Println("Begin: ", be.Format("15:01:05"))
			for _, s := range dir {
				s, err = filepath.Abs(s)
				if err != nil {
					return err
				}
				fmt.Println("Scan: " + s)
				x := conf.Matches(s, hit)
				fmt.Printf("Found %d entries\n", len(x))
				if o != nil && len(x) > 0 {
					for _, e := range x {
						_, _ = o.WriteString(e)
						_, _ = o.Write([]byte{'\n'})
					}
				}
				r = append(r, x...)
			}
			if o != nil {
				fn.Panic(o.Close())
				o = nil
			}
			if p {
				purify(dbg, r)
				if out == "" {
					fmt.Printf("Cleaned: %d entries", len(r))
				}
			} else if out == "" {
				fmt.Println("Clean entries:")
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
func purify(debug uint, list []string) {
	for _, s := range list {
		if debug != 0 {
			fmt.Println("purify " + s)
		}
		err := os.RemoveAll(s)
		if debug != 0 && err != nil {
			fmt.Println("purify " + s + " with error: " + err.Error())
		}
	}
}

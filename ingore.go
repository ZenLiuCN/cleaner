package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ZenLiuCN/fn"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Ignore struct {
	Path      string //config file path
	IgnoreDir []*regexp.Regexp
	Ignore    []*regexp.Regexp
	KeepDir   []*regexp.Regexp
	Keep      []*regexp.Regexp
	Debug     bool
}

// NewIgnore parse .ignore file. if any error happened will panic
func NewIgnore(path string) Ignore {
	b := fn.Panic1(os.ReadFile(path))
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Split(bufio.ScanLines)
	i := new(Ignore)
	i.Path = path
	for sc.Scan() {
		txt := sc.Text()
		switch {
		case txt[0] == '#':
		case len(txt) == 0:
		default:
			dir := txt[len(txt)-1] == '/' || (txt[len(txt)-1] == '*' && txt[len(txt)-2] == '*')
			switch {
			case txt[0] == '!':
				if dir {
					i.KeepDir = append(i.KeepDir, Compile(txt[1:]))
				} else {
					i.Keep = append(i.Keep, Compile(txt[1:]))
				}
			case txt[0] == '\\' && txt[1] == '!':
				if dir {
					i.IgnoreDir = append(i.IgnoreDir, Compile(txt[1:]))
				} else {
					i.Ignore = append(i.Ignore, Compile(txt[1:]))
				}
			default:
				if dir {
					i.IgnoreDir = append(i.IgnoreDir, Compile(txt))
				} else {
					i.Ignore = append(i.Ignore, Compile(txt))
				}
			}
		}
	}
	fn.Panic(sc.Err())
	return *i
}

// Matches returns ignored file and directories or none ignored (with
// Reverse=true).this will auto discovery sub ignore files. this should only call
// once for it won't keep original state.
func (i Ignore) Matches(root string) (r []string) {
	root = fn.Panic1(filepath.Abs(root))
	if !os.IsPathSeparator(root[len(root)-1]) {
		root = root + string(filepath.Separator)
	}
	var ignores []Ignore
	var last []string
	addLocal := func(path, rel string, git bool) {
		if len(last) == 0 {
			last = append(last, rel)
		} else if last[len(last)-1] != rel {
			last = append(last, rel)
		} else {
			li := ignores[len(ignores)-1]
			if git {
				li.Merge(NewIgnore(path), false)
			} else {
				li.Merge(NewIgnore(path), true)
			}
			return
		}
		if git {
			ignores = append(ignores, NewIgnore(path))
		} else {
			ignores = append(ignores, NewIgnore(path))
		}
	}
	checkLocal := func(rel string) {
		if len(last) == 0 {
			return
		}
		if strings.HasPrefix(rel, last[len(last)-1]) {
			return
		}
		ignores = ignores[:len(ignores)-1]
		if len(ignores) == 0 {
			last = nil
		} else {
			last = last[:len(last)-1]
		}
	}
	findLocal := func(path string) {
		ig := filepath.Join(path, ".gitignore")
		_, err := os.Stat(ig)
		if err == nil {
			if i.Debug {
				fmt.Println("found local config: " + ig)
			}
			addLocal(ig, strings.TrimPrefix(path, root), true)

		}
		ig = filepath.Join(path, ".cleanignore")
		_, err = os.Stat(ig)
		if err == nil {
			if i.Debug {
				fmt.Println("found local config: " + ig)
			}
			addLocal(ig, strings.TrimPrefix(path, root), false)

		}
	}
	findLocal(root)
	var lastPath string
	fn.Panic(filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		rel := strings.TrimPrefix(path, root)
		if rel == "" {
			return nil
		}
		if lastPath == "" && info.IsDir() {
			lastPath = path
		} else if info.IsDir() && lastPath != path {
			findLocal(path)
		}
		checkLocal(rel)
		if err != nil {
			if info.IsDir() {
				_, _ = fmt.Fprintf(os.Stderr, "reading %s fail,skip", path)
				return filepath.SkipDir
			}
			return err
		}
		if i.clean(rel, info.IsDir()) {
			r = append(r, path)
			if info.IsDir() {
				return filepath.SkipDir
			}
		} else {
			for j := 0; j < len(ignores); j++ {
				if ignores[j].clean(rel, info.IsDir()) {
					r = append(r, path)
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

		return nil
	}))
	return

}

// Merge another ig, if higher, ig will have higher order
func (i *Ignore) Merge(ig Ignore, higher bool) {
	if higher {
		i.Ignore = append(ig.Ignore, i.Ignore...)
		i.IgnoreDir = append(ig.IgnoreDir, i.IgnoreDir...)
		i.KeepDir = append(ig.KeepDir, i.KeepDir...)
		i.Keep = append(ig.Keep, i.Keep...)
	} else {
		i.KeepDir = append(i.KeepDir, ig.KeepDir...)
		i.IgnoreDir = append(i.IgnoreDir, ig.IgnoreDir...)
		i.Keep = append(i.Keep, ig.Keep...)
		i.Ignore = append(i.Ignore, ig.Ignore...)
	}
}

func (i Ignore) clean(rel string, dir bool) bool {
	if dir {
		for _, r := range i.KeepDir {
			if r.MatchString(rel) {
				if i.Debug {
					fmt.Printf("keep: %s by %s  in %s\n", rel, r.String(), i.Path)
				}
				return false
			}
		}
		for _, r := range i.IgnoreDir {
			if r.MatchString(rel) {
				if i.Debug {
					fmt.Printf("match: %s by %s  in %s\n", rel, r.String(), i.Path)
				}
				return true
			}
		}
	} else {
		for _, r := range i.Keep {
			if r.MatchString(rel) {
				if i.Debug {
					fmt.Printf("keep: %s by %s  in %s\n", rel, r.String(), i.Path)
				}
				return false
			}
		}
		for _, r := range i.Ignore {
			if r.MatchString(rel) {
				if i.Debug {
					fmt.Printf("match: %s by %s  in %s\n", rel, r.String(), i.Path)
				}
				return true
			}
		}
	}
	if i.Debug {
		fmt.Printf("ignore: %s\n", rel)
	}
	return false
}

// Append extra higher priorities patterns which not
func (i *Ignore) Append(extra []string) {
	var kd []*regexp.Regexp
	var k []*regexp.Regexp
	var id []*regexp.Regexp
	var ix []*regexp.Regexp
	for _, txt := range extra {
		txt = strings.TrimSpace(txt)
		switch {
		case txt[0] == '#':
		case len(txt) == 0:
		default:
			dir := txt[len(txt)-1] == '/' || (txt[len(txt)-1] == '*' && txt[len(txt)-2] == '*')
			switch {
			case txt[0] == '!':
				if dir {
					kd = append(kd, Compile(txt[1:]))
				} else {
					k = append(k, Compile(txt[1:]))
				}
			case txt[0] == '\\' && txt[1] == '!':
				if dir {
					id = append(id, Compile(txt[1:]))
				} else {
					ix = append(ix, Compile(txt[1:]))
				}
			default:
				if dir {
					id = append(id, Compile(txt))
				} else {
					ix = append(ix, Compile(txt))
				}
			}
		}
	}
	i.KeepDir = append(kd, i.KeepDir...)
	i.Keep = append(k, i.Keep...)
	i.IgnoreDir = append(id, i.IgnoreDir...)
	i.Ignore = append(ix, i.Ignore...)
}

// Compile compile a string git ignore pattern to regexp
func Compile(ig string) *regexp.Regexp {
	p := new(strings.Builder)
	n := len(ig) - 1
	for i := 0; i < len(ig); i++ {
		c := ig[i]
		switch {
		case c == '*' && i != n && ig[i+1] == '*':
			p.WriteString(".*")
		case c == '*':
			p.WriteString("[^/]*")
		case c == '.':
			p.WriteString("\\.")
		case c == '\\':
			p.WriteByte(ig[i+1])
			i++
		case c == '(':
			p.WriteString("\\(")
		case c == ')':
			p.WriteString("\\)")
		case c == '?':
			p.WriteString(".?")
		default:
			p.WriteByte(c)
		}
	}
	return regexp.MustCompile(p.String())
}

// Load from global and user config
func Load(debug bool) (r Ignore) {
	g := false
	str, _ := os.Executable()
	str, _ = filepath.EvalSymlinks(str)
	if str != "" {
		global := filepath.Join(filepath.Dir(str), ".cleaner")
		_, err := os.Stat(global)
		if err == nil {
			if debug {
				fmt.Println("found Global config at " + global)
			}
			r = NewIgnore(global)
			g = true
		}
	} else if debug {
		fmt.Println("not found Global config at " + str)
	}
	str, _ = os.UserHomeDir()
	if str == "" {
		if debug {
			fmt.Println("not found user config at " + str)
		}
		return
	}
	user := filepath.Join(str, ".cleaner")
	_, err := os.Stat(user)
	if err == nil {
		if debug {
			fmt.Println("found user config at " + user)
		}
		if !g {
			r = NewIgnore(user)
			return
		}
		x := NewIgnore(user)
		(&r).Merge(x, true)
	}
	return
}

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

type Local struct {
	ignores []Ignore
	root    string
}

func (s *Local) Init(root string) {
	s.root = root
}
func (s *Local) last() (string, *Ignore) {
	if len(s.ignores) > 0 {
		return s.ignores[len(s.ignores)-1].Rel, &s.ignores[len(s.ignores)-1]
	}
	return "", nil
}

// find local ignores
func (s *Local) find(debug uint, path string) {
	ig := filepath.Join(path, ".gitignore")
	_, err := os.Stat(ig)
	if err == nil {
		if debug > 0 {
			fmt.Println("Local config: " + ig)
		}
		s.AddLocal(ig, true)
	}
	ig = filepath.Join(path, ".cleanignore")
	_, err = os.Stat(ig)
	if err == nil {
		if debug > 0 {
			fmt.Println("Local config: " + ig)
		}
		s.AddLocal(ig, false)

	}
}

// AddLocal add local ignores
func (s *Local) AddLocal(path string, git bool) {
	rel := strings.TrimPrefix(filepath.ToSlash(filepath.Dir(path))+"/", s.root)
	if len(s.ignores) != 0 {
		pre, li := s.last()
		if pre == rel {
			if git {
				li.Merge(NewIgnore(path), false)
			} else {
				li.Merge(NewIgnore(path), true)
			}
			return
		}
	}
	i := NewIgnore(path)
	i.Rel = rel
	s.ignores = append(s.ignores, i)
}

// Check path change and pop local ignores
func (s *Local) Check(debug uint, path string, isDir bool) {
	rel := strings.TrimPrefix(filepath.ToSlash(path), s.root)
	if isDir {
		rel += "/"
	}
	if len(s.ignores) != 0 {
		n := len(s.ignores) - 1
		var i int
		for i = n; i >= 0; i-- {
			dir := s.ignores[i].Rel
			if rel == dir {
				if i < n {
					fmt.Printf("pop: %s =>%s\n", dir, path)
					s.ignores = s.ignores[:i+1]
				}
				return
			}
			if strings.HasPrefix(rel, dir) {
				break
			}
		}
		if i != -1 {
			if i < n {
				s.ignores = s.ignores[:i+1]
			}
			if isDir {
				s.find(debug, path)
			}
			return
		}
	}
	if isDir {
		s.find(debug, path)
	}

}

// Matches check if matched by local ignores
func (s *Local) Matches(path, rel string, isDir, hit bool) (r string, k bool) {
	var t bool
	var h string
	for j := len(s.ignores) - 1; j >= 0; j-- {
		if !strings.HasPrefix(rel, s.ignores[j].Rel) {
			return
		}
		rr := strings.TrimPrefix(rel, s.ignores[j].Rel)
		if t, k, h = s.ignores[j].clean(rr, isDir, hit); t {
			if hit {
				r = path + "\t" + h
				return
			} else {
				r = path
				return
			}
		} else if k {
			return
		}
	}
	return
}

type Ignore struct {
	Path      string //config file path
	Rel       string //context relative path used by Local
	IgnoreDir []*regexp.Regexp
	Ignore    []*regexp.Regexp
	KeepDir   []*regexp.Regexp
	Keep      []*regexp.Regexp
	Debug     uint
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
		txt = strings.TrimSpace(txt)
		switch {
		case len(txt) == 0:
		case txt[0] == '#':
		default:
			if txt[len(txt)-1] == '\\' {
				txt += " "
			}
			dir := txt[len(txt)-1] == '/' || (txt[len(txt)-1] == '*' && txt[len(txt)-2] == '*')
			switch {
			case txt[0] == '!':
				if len(txt) > 1 {
					if dir {
						i.KeepDir = append(i.KeepDir, Compile(txt[1:]))
					} else {
						i.Keep = append(i.Keep, Compile(txt[1:]))
					}
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
func (i Ignore) Matches(root string, hit bool) (r []string) {
	root = fn.Panic1(filepath.Abs(root))
	if !os.IsPathSeparator(root[len(root)-1]) {
		root = root + string(filepath.Separator)
	}
	local := new(Local)
	local.Init(filepath.ToSlash(root))
	local.Check(i.Debug, root, true)
	fn.Panic(filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		rel := strings.TrimPrefix(path, root)
		if rel == "" {
			return nil
		}
		local.Check(i.Debug, path, info.IsDir())
		if err != nil {
			if info.IsDir() {
				_, _ = fmt.Fprintf(os.Stderr, "reading %s fail,skip", path)
				return filepath.SkipDir
			}
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			rel += "/"
			path += string(filepath.Separator)
		}
		var t, k bool
		var h string
		h = filepath.Base(rel)
		if info.IsDir() {
			h += "/"
		}
		if t, k, h = i.clean(h, info.IsDir(), hit); t {
			if hit {
				r = append(r, path+"\t"+h)
			} else {
				r = append(r, path)
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
		} else if !k {
			if h, k = local.Matches(path, rel, info.IsDir(), hit); h != "" {
				if !k {
					r = append(r, h)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if k && info.IsDir() {
			return filepath.SkipDir
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

func (i Ignore) clean(rel string, dir bool, hit bool) (c, k bool, s string) {
	if dir {
		for _, r := range i.KeepDir {
			if r.MatchString(rel) {
				if i.Debug > 1 {
					fmt.Printf("[D]keep: %s by %s  in %s\n", rel, r.String(), i.Path)
				}
				k = true
				return
			}
		}
		for _, r := range i.IgnoreDir {
			if r.MatchString(rel) {
				if i.Debug > 0 {
					fmt.Printf("[D]match: %s by %s  in %s\n", rel, r.String(), i.Path)
				}
				if hit {
					s = fmt.Sprintf("%s [I]\t%s", i.Path, r)
				}
				c = true
				return
			}
		}
	} else {
		for _, r := range i.Keep {
			if r.MatchString(rel) {
				if i.Debug > 1 {
					fmt.Printf("[F]keep: %s\t%s\t%s\n", rel, i.Path, r.String())
				}
				k = true
				return
			}
		}
		for _, r := range i.Ignore {
			if r.MatchString(rel) {
				if i.Debug > 0 {
					fmt.Printf("[F]match: %s\t%s\t%s\n", rel, i.Path, r.String())
				}
				if hit {
					s = fmt.Sprintf("%s [I]\t%s", i.Path, r)
				}
				c = true
				return
			}
		}
	}
	if i.Debug > 2 {
		fmt.Printf("ignore: %s\n", rel)
	}
	return
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
	ig = strings.TrimSpace(ig)
	if ig[len(ig)-1] == '\\' {
		ig += " "
	}
	p := new(strings.Builder)
	p.Grow(len(ig))
	n := len(ig) - 1
	sb := false
	if len(ig) >= 2 && ig[:2] != "**" {
		p.WriteRune('^')
	}
	var c byte
	for i := 0; i < len(ig); i++ {
		c = ig[i]
		switch {
		//glob
		case c == '[' && !sb:
			sb = true
		case c == ']' && sb:
			sb = false
		//translate
		case c == '*' && i < n-1 && ig[i+1] == '*' && ig[i+2] == '/':
			p.WriteString(".*")
			i += 2
		case c == '*' && i != n && ig[i+1] == '*':
			p.WriteString(".*")
			i += 1
		case c == '*':
			p.WriteString("[^/]*")
		case c == '\\':
			p.WriteByte(ig[i+1])
			i++
		case c == '!' && sb: //unix glob
			p.WriteString("^")
		case c == '?':
			p.WriteString(".")
		//escape
		case c == '+' && !sb:
			p.WriteString("\\+")

		//other escape
		case c == '.':
			p.WriteString("\\.")
		case c == '|':
			p.WriteString("\\|")
		case c == '$':
			p.WriteString("\\$")
		case c == '^':
			p.WriteString("\\^")
		case c == '{':
			p.WriteString("\\{")
		case c == '}':
			p.WriteString("\\}")
		case c == '(':
			p.WriteString("\\(")
		case c == ')':
			p.WriteString("\\)")

		default:
			p.WriteByte(c)
		}
	}
	if c != '/' {
		p.WriteRune('$') //end the file
	}
	return regexp.MustCompile(p.String())
}

// Load from global and user config
func Load(debug uint) (r Ignore) {
	g := false
	str, _ := os.Executable()
	str, _ = filepath.EvalSymlinks(str)
	if str != "" {
		global := filepath.Join(filepath.Dir(str), ".cleanignore")
		_, err := os.Stat(global)
		if err == nil {
			if debug > 0 {
				fmt.Println("GLOBAL config: " + global)
			}
			r = NewIgnore(global)
			g = true
		}
	}
	str, _ = os.UserHomeDir()
	if str == "" {
		return
	}
	user := filepath.Join(str, ".cleanignore")
	_, err := os.Stat(user)
	if err == nil {
		if debug > 0 {
			fmt.Println("USER config: " + user)
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

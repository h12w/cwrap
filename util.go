// Copyright 2014, Hǎiliàng Wáng. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cwrap

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unsafe"
)

var (
	MachineSize = int(unsafe.Sizeof(uintptr(0)))
)

func p(v ...interface{}) {
	fmt.Println(v...)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func gofmt(file string) error {
	p("go fmt:")
	defer p("")
	return newCmd("go", "fmt", file).exec()
}

func fp(w io.Writer, v ...interface{}) {
	fmt.Fprint(w, v...)
	fmt.Fprintln(w)
}

func fpn(w io.Writer, v ...interface{}) {
	fmt.Fprint(w, v...)
}

func trimPrefix(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}

func join(a []string, sep string) string {
	return strings.Join(a, sep)
}

func joins(a ...string) string {
	return strings.Join(a, "")
}

func sprint(v ...interface{}) string {
	return fmt.Sprint(v...)
}

func snakeToCamel(s string) string {
	ss := strings.Split(s, "_")
	for i := range ss {
		s := ss[i]
		if strings.ToUpper(s) == s {
			s = strings.ToLower(s)
		}
		ss[i] = strings.Title(s)
	}
	return join(ss, "")
}

func upperName(s string, re *regexp.Regexp) string {
	if re != nil {
		m := re.FindStringSubmatch(s)
		if len(m) > 1 && len(m[1]) > 2 {
			s = m[1]
		}
	}
	if len(s) > 3 {
		s = trimSuffix(s, "_t")
	}
	return snakeToCamel(s)
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func snakeToLowerCamel(s string) string {
	if len(s) <= 1 {
		return s
	}
	s = snakeToCamel(s)
	return strings.ToLower(s[:1]) + s[1:]
}

func spaceToSnake(s string) string {
	return strings.Replace(s, " ", "_", -1)
}

func typeSizeAssert(ctype, gotype string) string {
	return fmt.Sprintf(
		`	if s1, s2 := unsafe.Sizeof(%[1]s), unsafe.Sizeof(%[2]s); s1 != s2 {
			panic(fmt.Errorf("Size of %[1]s is %%d, expected %%d as Go's %[2]s.", s1, s2))
		}
	`, ctype, gotype)
}

type SSet struct {
	m map[string]struct{}
}

func NewSSet() SSet {
	return SSet{make(map[string]struct{})}
}

func (m *SSet) Len() int {
	return len(m.m)
}

func (m *SSet) IsNil() bool {
	return m.m == nil
}

func (m *SSet) Add(ss ...string) {
	for _, s := range ss {
		m.m[s] = struct{}{}
	}
}

func (m *SSet) Del(s string) {
	delete(m.m, s)
}

func (m *SSet) Has(s string) bool {
	if m.m == nil {
		return false
	}
	_, has := m.m[s]
	return has
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func replace(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func trimPreSuffix(s, fix string) string {
	return strings.TrimSuffix(strings.TrimPrefix(s, fix), fix)
}

func trimSuffix(s, suffix string) string {
	return strings.TrimSuffix(s, suffix)
}

func remove(s, substr string) string {
	return strings.Replace(s, substr, "", -1)
}

func atoi(s string) int {
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func fileExists(file string) bool {
	f, err := os.Open(file)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

type cmd struct {
	*exec.Cmd
}

func newCmd(name string, arg ...string) cmd {
	return cmd{exec.Command(name, arg...)}
}

func (c cmd) read(visit func(r io.Reader) error) error {
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := c.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()
	if err := c.Start(); err != nil {
		return err
	}
	go io.Copy(os.Stderr, stderr)
	if err := visit(stdout); err != nil {
		return err
	}
	if err := c.Wait(); err != nil {
		return err
	}
	return nil
}

func (c cmd) exec() error {
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()
	stderr, err := c.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()
	if err := c.Start(); err != nil {
		return err
	}
	go io.Copy(os.Stderr, stderr)
	go io.Copy(os.Stdout, stdout)
	if err := c.Wait(); err != nil {
		return err
	}
	return nil

}

func writeToString(write func(w io.Writer)) string {
	var buf bytes.Buffer
	write(&buf)
	return buf.String()
}

func pt(v interface{}) {
	p(reflect.TypeOf(v))
}

func hex(i int, length int) string {
	if i < 0 {
		return strconv.FormatInt(int64(i), 10)
	}
	s := strings.ToUpper(strconv.FormatInt(int64(i), 16))
	if len(s)+2 < length {
		s = strings.Repeat("0", length-len(s)-2) + s
	}
	return "0x" + s
}

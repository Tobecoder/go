// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package present

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"net/url"
	"log"
)

var (
	parsers = make(map[string]ParseFunc)
	funcs   = template.FuncMap{}
)

type ParseFunc func(ctx *Context, fileName string, lineNumber int, inputLine string) (Elem, error)

type ParseMode int

const (
	// If set, parse only the title and subtitle.
	TitlesOnly ParseMode = 1
)

// Template returns an empty template with the action functions in its FuncMap.
func Template() *template.Template {
	return template.New("").Funcs(funcs)
}

func Register(name string, parser ParseFunc) {
	if len(name) == 0 || name[0] == ';' {
		panic("bad name in Register: " + name)
	}
	parsers["."+name] = parser
}

type Doc struct {
	Title      string
	Subtitle   string
	Time       time.Time
	Authors    []Author
	Tags       []string
	TitleNotes []string
	Sections   []Section
}

// Author represents the person who wrote and/or is presenting the document.
type Author struct {
	Elem []Elem
}

// Section represents a section of a document (such as a presentation slide)
// comprising a title and a list of elements.
type Section struct {
	Number  []int
	Title   string
	Elem    []Elem   //元素
	Notes   []string //区块备注
	Classes []string
	Styles  []string
}

func (s Section) TemplateName() string { return "section" }

type Elem interface {
	TemplateName() string
}

type Context struct {
	ReadFile func(filename string) ([]byte, error)
}

func Parse(r io.Reader, name string, mode ParseMode) (*Doc, error) {
	ctx := Context{ReadFile: ioutil.ReadFile}
	return ctx.Parse(r, name, mode)
}

func (ctx *Context) Parse(r io.Reader, name string, mode ParseMode) (*Doc, error) {
	doc := new(Doc)
	lines, err := readLines(r)
	if err != nil {
		return doc, err
	}
	for i := lines.line; i < len(lines.text); i++ {
		if strings.HasPrefix(lines.text[i], "*") {
			break
		}
		if isSpeakerNote(lines.text[i]) {
			doc.TitleNotes = append(doc.TitleNotes, lines.text[i][2:])
		}
	}
	err = parseHeader(doc, lines)
	if err != nil {
		return nil, err
	}

	if mode&TitlesOnly != 0 {
		return doc, nil
	}

	// Authors
	if doc.Authors, err = parseAuthors(lines); err != nil {
		return nil, err
	}

	// Sections
	if doc.Sections, err = parseSections(ctx, name, lines, []int{}); err != nil {
		return nil ,err
	}

	return doc, nil
}

func parseAuthors(lines *Lines) (authors []Author, err error) {
	if _, ok := lines.nextNonEmpty(); !ok {
		return nil, errors.New("unexpected EOF")
	}
	lines.back()

	var a *Author
	for {
		text, ok := lines.next()
		if !ok {
			return nil, errors.New("unexpected EOF")
		}

		// If we find a section heading, we're done.
		if strings.HasPrefix(text, "* ") {
			lines.back()
			break
		}

		if isSpeakerNote(text) {
			continue
		}

		// If we encounter a blank we're done with this author.
		if a != nil && len(text) == 0 {
			authors = append(authors, *a)
			a = nil
			continue
		}
		if a == nil {
			a = new(Author)
		}

		// Parse the line. Those that
		// - begin with @ are twitter names,
		// - contain slashes are links, or
		// - contain an @ symbol are an email address.
		// The rest is just text.
		var el Elem
		switch {
		case strings.HasPrefix(text, "@"):
			el = parseURL("http://twitter.com/" + text[1:])
		case strings.Contains(text, ":"):
			el = parseURL(text)
		case strings.Contains(text, "@"):
			el = parseURL("mailto:" + text)
		}
		if l, ok := el.(Link); ok {
			l.Label = text
			el = l
		}
		if el == nil {
			el = Text{Lines: []string{text}}
		}
		a.Elem = append(a.Elem, el)
	}
	if a != nil {
		authors = append(authors, *a)
	}
	return authors, nil
}

func parseURL(text string) Elem {
	u, err := url.Parse(text)
	if err != nil {
		log.Printf("Parse(%q): %v", text, err)
		return nil
	}
	return Link{URL: u}
}

func parseSections(ctx *Context, name string, lines *Lines, number []int) ([]Section, error) {
	var sections []Section
	for i := 1; ; i++ {
		// Next non-empty line is title.
		text, ok := lines.nextNonEmpty()
		for ok && text == "" {
			text, ok = lines.next()
		}
		if !ok {
			break
		}
		prefix := strings.Repeat("*", len(number)+1)
		if !strings.HasPrefix(text, prefix+" ") {
			lines.back()
			break
		}
		section := Section{
			Number: append(append([]int{}, number...), i),
			Title:  text[len(prefix)+1:],
		}
		//取出下一非空行
		text, ok = lines.nextNonEmpty()
		//只要行不是"*"开头的
		for ok && !lesserHeading(text, prefix) {
			var e Elem
			r, _ := utf8.DecodeRuneInString(text)
			switch {
			case unicode.IsSpace(r):
				first := strings.IndexFunc(text, func(r rune) bool {
					return !unicode.IsSpace(r)
				})
				if first < 0 {
					break
				}
				indent := text[:first]
				var s []string
				for ok && (strings.HasPrefix(text, indent) || text == "") {
					if text != "" {
						text = text[first:]
					}
					s = append(s, text)
					text, ok = lines.next()
				}
				lines.back()
				pre := strings.Join(s, "\n")
				pre = strings.Replace(pre, "\t", "    ", -1) // browsers treat tabs badly
				pre = strings.TrimRightFunc(pre, unicode.IsSpace)
				e = Text{Lines: []string{pre}, Pre: true}
			case strings.HasPrefix(text, "- "):
				var b []string
				for ok && strings.HasPrefix(text, "- ") {
					b = append(b, text[2:])
					text, ok = lines.next()
				}
				lines.back()
				e = List{Bullet: b}
			case isSpeakerNote(text):
				section.Notes = append(section.Notes, text[2:])
			case strings.HasPrefix(text, prefix+"* "):
				lines.back()
				subsecs, err := parseSections(ctx, name, lines, section.Number)
				if err != nil {
					return nil, err
				}
				for _, ss := range subsecs {
					section.Elem = append(section.Elem, ss)
				}
			case strings.HasPrefix(text, "."):
				args := strings.Fields(text)
				if args[0] == ".background" {
					section.Classes = append(section.Classes, "background")
					section.Styles = append(section.Styles, "background-image: url('"+args[1]+"')")
					break
				}
				parser := parsers[args[0]]
				if parser == nil {
					return nil, fmt.Errorf("%s:%d: unknown command %q\n", name, lines.line, text)
				}
				t, err := parser(ctx, name, lines.line, text)
				if err != nil {
					return nil, err
				}
				e = t
			default:
				var l []string
				for ok && strings.TrimSpace(text) != "" {
					if text[0] == '.' { // Command breaks text block.
						lines.back()
						break
					}
					if strings.HasPrefix(text, `\.`) { // Backslash escapes initial period.
						text = text[1:]
					}
					l = append(l, text)
					text, ok = lines.next()
				}
				if len(l) > 0 {
					e = Text{Lines: l}
				}
			}
			if e != nil {
				section.Elem = append(section.Elem, e)
			}
			text, ok = lines.nextNonEmpty()
		}
		if isHeading.MatchString(text) {
			lines.back()
		}
		sections = append(sections, section)
	}
	return sections, nil
}

type List struct {
	Bullet []string
}

func (l List) TemplateName() string { return "list" }

type Text struct {
	Lines []string
	Pre   bool
}

func (t Text) TemplateName() string { return "text" }

var isHeading = regexp.MustCompile(`^\*+ `)

func lesserHeading(text, prefix string) bool {
	return isHeading.MatchString(text) && !strings.HasPrefix(text, prefix+"*")
}

func parseHeader(doc *Doc, lines *Lines) error {
	var ok bool
	doc.Title, ok = lines.nextNonEmpty()
	if !ok {
		return fmt.Errorf("unexpected EOF; expected title")
	}
	//逐行读取，空行跳出，isSpeakerNote跳过，处理有前缀的"Tags:"，解析时间，最后处理子标题
	for {
		text, ok := lines.next()
		if !ok {
			return errors.New("unexpect EOF")
		}
		if len(text) == 0 {
			break
		}
		if isSpeakerNote(text) {
			continue
		}
		const tagPrefix = "Tags:"
		if strings.HasPrefix(text, tagPrefix) {
			tags := strings.Split(text[len(tagPrefix):], ",")
			for i, _ := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
			doc.Tags = append(doc.Tags, tags...)
		} else if t, ok := parseTime(text); ok {
			doc.Time = t
		} else if doc.Subtitle == "" {
			doc.Subtitle = text
		} else {
			return fmt.Errorf("unexpected header line: %q", text)
		}
	}
	return nil
}

func parseTime(s string) (t time.Time, ok bool) {
	t, err := time.Parse("15:04 2 Jan 2006", s)
	if err == nil {
		return t, true
	}
	t, err = time.Parse("2 Jan 2006", s)
	if err == nil {
		// at 11am UTC it is the same date everywhere
		t = t.Add(time.Hour * 11)
		return t, true
	}
	return
}

type Lines struct {
	line int // 0 indexed, so has 1-indexed number of last line returned
	text []string
}

func (l *Lines) back() {
	l.line--
}

func (l *Lines) nextNonEmpty() (text string, ok bool) {
	for {
		text, ok = l.next()
		if !ok {
			return
		}
		if len(text) > 0 {
			break
		}
	}
	return
}

func (l *Lines) next() (text string, ok bool) {
	for {
		current := l.line
		l.line++
		if current >= len(l.text) {
			//这里的返回参数不能省略，text可能包含'#'
			return "", false
		}
		text = l.text[current]
		// '#'这个开头的是注释
		if len(text) == 0 || text[0] != '#' {
			ok = true
			break
		}
	}
	return
}

func readLines(r io.Reader) (*Lines, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &Lines{0, lines}, nil
}

// renderElem implements the elem template function, used to render
// sub-templates.
func renderElem() string {
	return ""
}

func init() {
	funcs["elem"] = renderElem
}

func isSpeakerNote(s string) bool {
	return strings.HasPrefix(s, ": ")
}

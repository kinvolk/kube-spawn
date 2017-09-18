package script

import (
	"bytes"
	"fmt"
	"text/template"
)

func render(tmpl string, opts interface{}) *bytes.Buffer {
	var buf = new(bytes.Buffer)
	t := template.Must(template.New(fmt.Sprintf("%T", opts)).Parse(tmpl))
	if err := t.Execute(buf, opts); err != nil {
		return nil
	}
	return buf
}

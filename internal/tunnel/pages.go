package tunnel

import (
	_ "embed"
	"html/template"
)

//go:embed server_down.html
var serverDownHTML string

var serverDownTmpl = template.Must(template.New("server_down").Parse(serverDownHTML))

type serverDownData struct {
	Port  int
	Error string
}

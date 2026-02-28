package proxy

import (
	_ "embed"
	"html/template"
)

//go:embed upstream_down.html
var upstreamDownHTML string

var upstreamDownTmpl = template.Must(template.New("upstream_down").Parse(upstreamDownHTML))

type upstreamDownData struct {
	Host string
	Port int
}

// Command mailgraph is an RRDtool-based mail statistics grapher for Postfix and other MTAs.
package main

import (
	_ "embed"

	"mailgraph/cmd"
)

//go:embed web/static/mailgraph.css
var mailgraphCSS []byte

func main() {
	cmd.SetMailgraphCSS(mailgraphCSS)
	cmd.Execute()
}
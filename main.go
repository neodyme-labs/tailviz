package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/tailscale/hujson"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
	tailscale "tailscale.com/client/tailscale"
)

func matchName(
	a string,
	names []string,
	ignoreWildcard bool,
) ([]string, error) {
	res := []string{}
	if ignoreWildcard && a == "*" {
		return res, nil
	}

	aRegex := strings.Replace(a, "*", ".*", -1)
	aRegex = "^" + strings.Replace(aRegex, ":.*", "(:\\*)?", -1) + "$"

	r, err := regexp.Compile(aRegex)
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		if r.MatchString(name) {
			res = append(res, name)
		}
	}

	return res, nil
}

func matchNodes(
	array []string,
	nodes map[string]*cgraph.Node,
	ignoreWildcard bool,
) (map[string]*cgraph.Node, error) {
	res := make(map[string]*cgraph.Node)
	for _, a := range array {
		matches, err := matchName(a, maps.Keys(nodes), ignoreWildcard)
		if err != nil {
			return nil, err
		}

		if len(matches) == 0 && a != "*" {
			log.Printf("WARNING: No match for \"%s\"\n", a)
		}

		for _, m := range matches {
			res[m] = nodes[m]
		}
	}

	return res, nil
}

func main() {
	Autogroups := []string{
		"autogroup:internet",
		"autogroup:self",
		"autogroup:owner",
		"autogroup:admin",
		"autogroup:member",
		"autogroup:tagged",
		"autogroup:auditor",
		"autogroup:billing",
		"autogroup:it",
		"autogroup:network",
		"autogroup:shared",
		"autogroup:nonroot",
	}

	app := &cli.App{
		Name:  "tailviz",
		Usage: "Visualize you tailnet acls",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:      "input",
				Aliases:   []string{"i"},
				Usage:     "Tailnet hujson file to visualize",
				Required:  true,
				TakesFile: true,
			},
			&cli.PathFlag{
				Name:      "output",
				Aliases:   []string{"o"},
				Usage:     "Output file, depending on the extension, the format is chosen. (Supported extensions: dot, svg, png, jpg)",
				Required:  true,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:     "layout",
				Aliases:  []string{"l"},
				Usage:    "Specify the layout to use. (Supported layouts: circo, dot, fdp, neato, osage, patchwork, sfdp, twopi)",
				Required: false,
				Value:    "dot",
			},
			&cli.BoolFlag{
				Name:     "ignore-wildcard",
				Usage:    "Do not render wildcard edges",
				Required: false,
				Value:    false,
			},
		},
		Action: func(cCtx *cli.Context) error {
			inputPath := cCtx.Path("input")
			outputPath := cCtx.Path("output")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				return err
			}

			b, err := hujson.Standardize(input)
			if err != nil {
				return err
			}

			var aclDetails tailscale.ACLDetails
			if err = json.Unmarshal(b, &aclDetails); err != nil {
				return err
			}

			g := graphviz.New()
			g.SetLayout(graphviz.Layout(cCtx.String("layout")))
			graph, err := g.Graph(graphviz.Directed)
			if err != nil {
				return err
			}
			defer func() {
				if err := graph.Close(); err != nil {
					log.Fatal(err)
				}
				g.Close()
			}()

			nodeNames := slices.Concat(
				Autogroups,
				[]string{"*"},
				maps.Keys(aclDetails.TagOwners),
				maps.Keys(aclDetails.Groups),
				maps.Keys(aclDetails.Hosts),
			)

			for _, members := range aclDetails.Groups {
				nodeNames = slices.Concat(nodeNames, members)
			}

			for _, acl := range aclDetails.ACLs {
				for _, n := range slices.Concat(acl.Src, acl.Dst, acl.Users, acl.Ports) {
					res, err := matchName(n, nodeNames, false)
					if err != nil {
						return err
					}

					if len(res) == 0 {
						nodeNames = append(nodeNames, n)
					}
				}
			}

			nodes := make(map[string]*cgraph.Node)

			for _, name := range nodeNames {
				node, err := graph.CreateNode(name)
				if err != nil {
					return err
				}

				nodes[name] = node
			}

			wildcardNode := nodes["*"]
			wildcardNode.SetColor("red")
			wildcardNode.SetShape(cgraph.PolygonShape)

			for group, members := range aclDetails.Groups {
				groupNode := nodes[group]

				for _, member := range members {
					memberNode := nodes[member]

					e, err := graph.CreateEdge("", memberNode, groupNode)
					if err != nil {
						return err
					}

					e.SetStyle(cgraph.DashedEdgeStyle)
					e.SetColor("blue")
				}
			}

			for name, host := range aclDetails.Hosts {
				node := nodes[name]

				hostNode := nodes[host]

				e, err := graph.CreateEdge("", hostNode, node)
				if err != nil {
					return err
				}

				e.SetStyle(cgraph.DashedEdgeStyle)
				e.SetColor("blue")
			}

			ignoreWildcard := cCtx.Bool("ignore-wildcard")
			for _, acl := range aclDetails.ACLs {
				if acl.Action != "accept" {
					return errors.New("Action has to be accepts")
				}

				from, err := matchNodes(slices.Concat(acl.Src, acl.Users), nodes, ignoreWildcard)
				if err != nil {
					return err
				}

				to, err := matchNodes(slices.Concat(acl.Dst, acl.Ports), nodes, ignoreWildcard)
				if err != nil {
					return err
				}

				for _, f := range from {
					for _, t := range to {
						_, err := graph.CreateEdge("", f, t)
						if err != nil {
							return err
						}
					}
				}
			}

			format := graphviz.Format(filepath.Ext(outputPath)[1:])
			var buf bytes.Buffer
			if err := g.Render(graph, format, &buf); err != nil {
				return err
			}

			if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

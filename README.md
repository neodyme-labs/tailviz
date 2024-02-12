# tailviz

tailviz is tool to visualize your tailnet [acl](https://tailscale.com/kb/1018/acls) config in a graph.
This can be helpful to better understand which devices can communicate whith which devices. 

## Features

- Show all tags, users, ips and group relations
- Support dot, svg, png and jpg output
- Support specifying the layout to render the graph
- Option to ignore wildcard edges

## Usage

This tool is written using go 1.22. This is the only requirement.

Once installed:
```bash
git clone https://github.com/neodyme-labs/tailviz.git
cd tailviz
go run main.go --input <input_path> --output <output_path>
```

```bash
go run main.go -h
NAME:
   tailviz - Visualize you tailnet acls

USAGE:
   tailviz [global options] command [command options] 

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --input value, -i value   Tailnet hujson file to visualize
   --output value, -o value  Output file, depending on the extension, the format is chosen. (Supported extensions: dot, svg, png, jpg)
   --layout value, -l value  Specify the layout to use. (Supported layouts: circo, dot, fdp, neato, osage, patchwork, sfdp, twopi) (default: "dot")
   --ignore-wildcard         Do not render wildcard edges (default: false)
   --help, -h                show help

```

## Disclaimer

This is not an official Tailscale or Tailscale Inc. project.
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	Version        = "0.4.0"
	SingBoxVersion = "0.14.0"
)

type options struct {
	workingDir string
	host       string
	port       int
	version    bool
	command    string
}

func Execute() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printUsage()
		os.Exit(1)
	}

	if opts.version {
		printVersion()
		return
	}

	switch opts.command {
	case "serve":
		if opts.workingDir == "" {
			fmt.Fprintln(os.Stderr, "error: -d is required")
			printUsage()
			os.Exit(1)
		}
		if err := serveCmd(opts.workingDir, opts.host, opts.port); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "version":
		printVersion()
	case "":
		printUsage()
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", opts.command)
		os.Exit(1)
	}
}

func parseArgs(args []string) (*options, error) {
	opts := &options{host: "127.0.0.1", port: 8080}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-d", "--dir":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag %s requires a value", arg)
			}
			i++
			opts.workingDir = args[i]
		case "--listen":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag %s requires a value", arg)
			}
			i++
			opts.host = args[i]
		case "-p", "--port":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("flag %s requires a value", arg)
			}
			i++
			port, err := strconv.Atoi(args[i])
			if err != nil || port <= 0 || port > 65535 {
				return nil, fmt.Errorf("invalid port: %s", args[i])
			}
			opts.port = port
		case "-v", "--version":
			opts.version = true
		case "serve", "version", "help":
			if opts.command != "" && opts.command != arg {
				return nil, fmt.Errorf("multiple commands: %s and %s", opts.command, arg)
			}
			opts.command = arg
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			return nil, fmt.Errorf("unknown command: %s", arg)
		}
	}
	return opts, nil
}

func printVersion() {
	fmt.Printf("version: %s\nsupported sing-box version: %s\n", Version, SingBoxVersion)
}

func printUsage() {
	fmt.Print(`usage: sbctl [flags] <command>

commands:
  serve    start webui server
  version  print version
  help     print this help

flags:
  -d <dir>        working directory (required for serve)
  --listen <host> listen address (default 127.0.0.1, use 0.0.0.0 for LAN access)
  -p <port>       webui port (default 8080)
  -v              print version
`)
}

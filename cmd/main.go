/*===========================================================================*\
 *           MIT License Copyright (c) 2022 Kris Nóva <kris@nivenly.com>     *
 *                                                                           *
 *                ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓                *
 *                ┃   ███╗   ██╗ ██████╗ ██╗   ██╗ █████╗   ┃                *
 *                ┃   ████╗  ██║██╔═████╗██║   ██║██╔══██╗  ┃                *
 *                ┃   ██╔██╗ ██║██║██╔██║██║   ██║███████║  ┃                *
 *                ┃   ██║╚██╗██║████╔╝██║╚██╗ ██╔╝██╔══██║  ┃                *
 *                ┃   ██║ ╚████║╚██████╔╝ ╚████╔╝ ██║  ██║  ┃                *
 *                ┃   ╚═╝  ╚═══╝ ╚═════╝   ╚═══╝  ╚═╝  ╚═╝  ┃                *
 *                ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛                *
 *                                                                           *
 *                       This machine kills fascists.                        *
 *                                                                           *
\*===========================================================================*/

package main

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"time"

	modcontainer "github.com/kris-nova/xpid/pkg/modules/container"

	modebpf "github.com/kris-nova/xpid/pkg/modules/ebpf"

	filter "github.com/kris-nova/xpid/pkg/filters"

	Raw "github.com/kris-nova/xpid/pkg/encoders/raw"

	"github.com/kris-nova/xpid/pkg/encoders/json"

	v1 "github.com/kris-nova/xpid/pkg/api/v1"

	modproc "github.com/kris-nova/xpid/pkg/modules/proc"

	"github.com/kris-nova/xpid/pkg/procx"

	"github.com/kris-nova/xpid"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var cfg = &AppOptions{}

type AppOptions struct {
	Verbose bool
	Fast    bool

	// Encoders
	Output string

	Hidden  bool
	Threads bool

	Probe bool

	// Modules
	All  bool
	Proc bool

	// Containers
	Container bool
}

const (
	ExitCode_PermissionDenied int = 99
)

func main() {
	/* Change version to -V */
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "The version of the program.",
	}
	app := &cli.App{
		Name:     xpid.Name,
		Version:  xpid.Version,
		Compiled: time.Now(),
		Authors: []*cli.Author{
			&cli.Author{
				Name:  xpid.AuthorName,
				Email: xpid.AuthorEmail,
			},
		},
		Copyright: xpid.Copyright,
		HelpName:  xpid.Copyright,
		Usage:     "Linux Process Discovery. Like nmap, but for pids.",
		UsageText: `xpid [flags] -o [output] <query>

Investigate pid 123 and write the report to out.txt
	xpid 123 > out.txt

Find all container processes on a system 
	# Looks for /proc/[pid]/ns/cgroup != /proc/1/ns/cgroup 
	xpid -c <query>

Find all processes running with eBPF programs at runtime.
	# Looks for /proc/[pid]/fdinfo and correlates to /sys/fs/bpf
	xpid --ebpf <query>

Find all processes between specific values
	xpid <flags> +100      # Search pids up to 100
	xpid <flags> 100-2000  # Search pids between 100-2000 
	xpid <flags> 65000+    # Search pids 65000 or above

Find all "hidden" processes on a system
	# Looks for chdir, opendir, and dent in /proc
	xpid -x <query>

Find all possible pids on a system, and investigate each one (slow). The --all flag is default.
	xpid > out.txt 

Investigate all pids from 0 to 1000 and write the report to out.json
	xpid -o json 0-1000 > out.json

`,
		Commands: []*cli.Command{
			&cli.Command{},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Destination: &cfg.Verbose,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o", "out"},
				Destination: &cfg.Output,
			},
			// Modules should have capital single letter flags!
			&cli.BoolFlag{
				Name:        "all",
				Aliases:     []string{"A"},
				Destination: &cfg.All,
				Value:       false,
			},
			&cli.BoolFlag{
				Name:        "fast",
				Aliases:     []string{"f"},
				Destination: &cfg.Fast,
				Value:       true,
			},
			&cli.BoolFlag{
				Name:        "probe",
				Aliases:     []string{"bpf", "ebpf", "b"},
				Destination: &cfg.Probe,
				Value:       false,
			},
			&cli.BoolFlag{
				Name:        "hidden",
				Aliases:     []string{"x"},
				Destination: &cfg.Hidden,
				Value:       false,
			},
			&cli.BoolFlag{
				Name:        "threads",
				Aliases:     []string{"t", "thread"},
				Destination: &cfg.Threads,
				Value:       false,
			},
			&cli.BoolFlag{
				Name:        "proc",
				Aliases:     []string{"P"},
				Destination: &cfg.Proc,
				Value:       false,
			},
			&cli.BoolFlag{
				Name:        "container",
				Aliases:     []string{"c", "containers"},
				Destination: &cfg.Container,
				Value:       false,
			},
		},
		EnableBashCompletion: false,
		HideHelp:             false,
		HideVersion:          true,
		Before: func(c *cli.Context) error {
			Preloader()
			return nil
		},
		After: func(c *cli.Context) error {
			// Destruct
			return nil
		},
		Action: func(c *cli.Context) error {
			var pids []*v1.Process
			query := c.Args().Get(0)
			if query == "" {
				max := procx.MaxPid()
				if max == -1 {
					return fmt.Errorf("unable to read from /proc")
				}
				query = fmt.Sprintf("1-%d", max)
			}

			// Initialize the explorer based on flags
			pids = procx.PIDQuery(query)
			if pids == nil {
				return fmt.Errorf("invalid pid query: %s", query)
			}
			logrus.Infof("Query : %s\n", query)
			x := procx.NewProcessExplorer(pids)

			// Fast
			x.SetFast(cfg.Fast)

			// Encoder
			var encoder procx.ProcessExplorerEncoder
			switch cfg.Output {
			case "json":
				encoder = json.NewJSONEncoder()
				break
			case "raw":
				encoder = Raw.NewRawEncoder()
				break
			case "color":
			default:
				rawcolor := Raw.NewRawEncoder()
				rawcolor.SetFormat(Raw.ColorFormatter)
				encoder = rawcolor
			}

			// Filters
			encoder.AddFilter(filter.RetainOnlyNamed)
			if cfg.Hidden {
				encoder.AddFilter(filter.RetainOnlyHidden)
			}
			if !cfg.Threads {
				encoder.AddFilter(filter.RejectThreads)
			}
			if cfg.Container {
				x.AddModule(modcontainer.NewContainerModule())
				encoder.AddFilter(filter.RetainOnlyContainers)
			}

			// Set encoder after filters are applied

			if !cfg.Probe && !cfg.Proc {
				cfg.All = true
			}
			if cfg.All {
				cfg.Proc = true
				//cfg.Probe = true
			}
			if cfg.Proc {
				pmod := modproc.NewProcModule()
				x.AddModule(pmod)
			}
			if cfg.Probe {
				// Also proc for names and meta
				pmod := modproc.NewProcModule()
				x.AddModule(pmod)
				bpfmod := modebpf.NewEBPFModule()
				x.AddModule(bpfmod)
				encoder.AddFilter(filter.RetainOnlyEBPF)
			}
			// Execute
			x.SetEncoder(encoder)
			x.SetWriter(os.Stdout)
			return x.Execute()
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		logrus.Errorf("execution error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// Preloader will run for ALL commands, and is used
// to initalize the runtime environments of the program.
func Preloader() {
	/* Flag parsing */
	if cfg.Verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}

	if cfg.Container {
		if !isuid(0) {
			logrus.Errorf("Permission denied.")
			os.Exit(ExitCode_PermissionDenied)
		}
	}
}

func isuid(check int) bool {
	u, _ := user.Current()
	if u == nil {
		return false
	}
	i, _ := strconv.Atoi(u.Uid)
	return check == i
}

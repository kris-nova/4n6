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
	"time"

	"github.com/kris-nova/xpid"
	"github.com/kris-nova/xpid/pkg/service"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var cfg = &AppOptions{}

type AppOptions struct {
	verbose bool
}

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
		Usage:     "Linux Process Discovery.",
		UsageText: `xpid [flags] -o [output]

Investigate pid 123 and write the report to out.txt
	xpid -p 123 > out.txt

Investigate pid 123 using the "--proc" module only.
	xpid --proc -p 123 > out.txt

`,
		Commands: []*cli.Command{
			&cli.Command{},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Destination: &cfg.verbose,
			},
		},
		EnableBashCompletion: false,
		HideHelp:             false,
		HideVersion:          true,
		Before: func(c *cli.Context) error {
			Preloader()
			fmt.Fprintf(c.App.Writer, xpid.Banner())
			return nil
		},
		After: func(c *cli.Context) error {
			// Destruct
			return nil
		},
		Action: func(c *cli.Context) error {

			//
			novaObject := service.NewNova()
			return novaObject.Run()
			//

		},
	}
	app.Run(os.Args)
}

// Preloader will run for ALL commands, and is used
// to initalize the runtime environments of the program.
func Preloader() {
	/* Flag parsing */
	if cfg.verbose {
		logrus.SetLevel(logrus.InfoLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}
}

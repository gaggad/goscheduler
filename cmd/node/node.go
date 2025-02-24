// Command goscheduler-node
package main

import (
	"os"
	"runtime"
	"strings"

	"github.com/gaggad/goscheduler/internal/modules/rpc/auth"
	"github.com/gaggad/goscheduler/internal/modules/rpc/server"
	"github.com/gaggad/goscheduler/internal/modules/utils"
	"github.com/gaggad/goscheduler/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	AppVersion, BuildDate, GitCommit string
)

func main() {
	app := &cli.App{
		Name:    "goscheduler-node",
		Usage:   "goscheduler node service",
		Version: AppVersion,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "allow-root",
				Usage: "allow running as root user",
			},
			&cli.StringFlag{
				Name:    "server-addr",
				Aliases: []string{"s"},
				Value:   "0.0.0.0:5921",
				Usage:   "server address to listen on",
			},
			&cli.BoolFlag{
				Name:  "enable-tls",
				Usage: "enable TLS",
			},
			&cli.StringFlag{
				Name:  "ca-file",
				Usage: "path to CA file",
			},
			&cli.StringFlag{
				Name:  "cert-file",
				Usage: "path to cert file",
			},
			&cli.StringFlag{
				Name:  "key-file",
				Usage: "path to key file",
			},
			&cli.StringFlag{
				Name:  "log-level",
				Value: "info",
				Usage: "log level (debug|info|warn|error|fatal)",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	level, err := logrus.ParseLevel(c.String("log-level"))
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	if c.Bool("version") {
		util.PrintAppVersion(AppVersion, GitCommit, BuildDate)
		return nil
	}

	enableTLS := c.Bool("enable-tls")
	if enableTLS {
		caFile := c.String("ca-file")
		certFile := c.String("cert-file")
		keyFile := c.String("key-file")

		if !utils.FileExist(caFile) {
			return cli.Exit("failed to read ca cert file: "+caFile, 1)
		}
		if !utils.FileExist(certFile) {
			return cli.Exit("failed to read server cert file: "+certFile, 1)
		}
		if !utils.FileExist(keyFile) {
			return cli.Exit("failed to read server key file: "+keyFile, 1)
		}
	}

	certificate := auth.Certificate{
		CAFile:   strings.TrimSpace(c.String("ca-file")),
		CertFile: strings.TrimSpace(c.String("cert-file")),
		KeyFile:  strings.TrimSpace(c.String("key-file")),
	}

	if runtime.GOOS != "windows" && os.Getuid() == 0 && !c.Bool("allow-root") {
		return cli.Exit("Do not run goscheduler-node as root user", 1)
	}

	server.Start(c.String("server-addr"), enableTLS, certificate)
	return nil
}

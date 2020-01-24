// Copyright 2017 Sigurd Hogsbro
// NATSCAT
// =======

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	cli "github.com/urfave/cli/v2"
)

var (
	appName   = "stancat"
	listen    bool
	verbose   bool
	buffered  bool
	serverURL string
	subject   string
	message   string
	clusterID string
	clientID  string
)

func cmdLine(c *cli.Context) error {
	verbose = c.Bool("verbose")
	listen = c.Bool("listen")
	buffered = c.Bool("buffered")

	if !listen && c.NArg() > 0 {
		message = strings.Join(c.Args().Slice(), " ")
		buffered = true
	}

	if subject == "" {
		err := cli.ShowCommandHelp(c, "")
		if err != nil {
			log.Printf("%v\n", err)
		}
		log.Fatalf("%s: Must specify subject string\n", appName)
	}

	if !listen {
		if strings.ContainsAny(subject, "*>") {
			log.Fatalf("%s: Cannot specify wildcard subject when publishing\n", appName)
		}
	}

	return nil
}

func cat() {
	nc, err := nats.Connect(serverURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	sc, err := stan.Connect(clusterID, clientID, stan.NatsConn(nc))
	if err != nil {
		log.Fatal(err)
	}

	if verbose {
		log.Printf("%s: Connected to %s\n", appName, nc.ConnectedUrl())
	}

	switch {
	case listen:
		// Listening for messages
		if verbose {
			log.Printf("%s: Listening on [%s], buffered %v\n", appName, subject, buffered)
		}
		_, err = sc.Subscribe(subject, func(m *stan.Msg) {
			if verbose {
				log.Println(m.Subject, string(m.Data))
				log.Printf("[%s] %s\n", m.Subject, string(m.Data))
			}
			if buffered {
				// print the message followed by CR/LF
				fmt.Println(string(m.Data))
			} else {
				// Write the binary message body to stdout
				buf := bytes.NewBuffer(m.Data)
				_, err = buf.WriteTo(os.Stdout)
				if err != nil {
					log.Fatal(err)
				}
			}
		})
		if err != nil {
			log.Fatal(err)
		}
		select {}

	case message != "":
		// Publish specified message
		err = sc.Publish(subject, []byte(message))
		if err != nil {
			log.Fatal(err)
		}
		if verbose {
			log.Printf("%s: [%s] Wrote '%s'\n", appName, subject, message)
		}

	case message == "":
		// Publish message(s) from stdin
		count := 0
		if buffered {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				err = sc.Publish(subject, []byte(line))
				if err != nil {
					log.Fatal(err)
				}
				count++
			}
			if verbose {
				log.Printf("%s: [%s] Wrote %d lines\n", appName, subject, count)
			}
		} else {
			bytes, _ := ioutil.ReadAll(os.Stdin)
			count = len(bytes)
			err = sc.Publish(subject, bytes)
			if err != nil {
				log.Fatal(err)
			}
			if verbose {
				log.Printf("%s: [%s] Wrote %d bytes\n", appName, subject, count)
			}
		}
	}

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Log to stderr without timestamp
	log.SetFlags(0)

	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version, V",
		Usage: "print the version",
	}
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s: version %s\n", c.App.Name, c.App.Version)
		os.Exit(1)
	}
	hp := cli.HelpPrinter
	cli.HelpPrinter = func(w io.Writer, templ string, data interface{}) {
		hp(w, templ, data)
		os.Exit(1)
	}

	app := &cli.App{
		Name:      appName,
		Usage:     "cat to/from STAN subject",
		UsageText: "stancat [global options] topic [message to post]",
		Compiled:  time.Now(),
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Eelco Cramer",
				Email: "eelco@servicelab.org",
			},
		},
		Version: "0.1",
		Action:  cmdLine,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "cluster_id",
				Usage:       "Cluster ID",
				Value:       "test-cluster",
				Destination: &clusterID,
			},
			&cli.StringFlag{
				Name:        "client_id",
				Aliases:     []string{"cid", "c"},
				Value:       "stancat",
				Destination: &clientID,
			},
			&cli.BoolFlag{
				Name:        "buffered",
				Aliases:     []string{"b"},
				Usage:       "read/write messages in buffered mode, terminated by CR/LF",
				Destination: &buffered,
			},
			&cli.StringFlag{
				Name:        "message",
				Aliases:     []string{"m"},
				Usage:       "message to publish",
				Value:       "",
				Destination: &message,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Usage:       "verbose logging",
				Destination: &verbose,
			},
			&cli.BoolFlag{
				Name:        "listen",
				Aliases:     []string{"l"},
				Usage:       "listen for messages",
				Destination: &listen,
			},
			&cli.StringFlag{
				Name:        "subject",
				Aliases:     []string{"s"},
				Value:       "",
				Usage:       "[Required] NATS subject ('*' and '>' wildcards only valid when listening)",
				Destination: &subject,
			},
			&cli.StringFlag{
				Name:        "server",
				Aliases:     []string{"S"},
				Value:       nats.DefaultURL,
				Usage:       "NATS server URL(s), comma-separated",
				EnvVars:     []string{"NATS"},
				Destination: &serverURL,
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	cat()
}

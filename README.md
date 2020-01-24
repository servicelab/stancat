# STANCAT

A [Go](http://golang.org) tool to send/receive input to [STAN messaging system](https://docs.nats.io/nats-streaming-concepts/intro), modeled on Unix `netcat` and `cat`.

Based upon [natscat](https://github.com/shogsbro/natscat).

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT)

## Installation
```bash
go install github.com/servicelab/stancat
```

## Basic Usage

### Sending to a STAN subject
```bash
# Send a message to subject 'test'
stancat -s test -m "Test message"

# Send contents of a file to subject 'test'
stancat -s test <README.md
```

### Listening on a STAN subject
```bash
# Listen to a specific subject, writing messages in buffered mode (CRLF appended)
stancat -l -s test -b

# Listen to all subjects
stancat -l -s '>'
```

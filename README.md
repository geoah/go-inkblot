# Inkblot

This is a proof of concept identity server for [ink](protocol.ink).

## Deployment

[![Deploy](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy)

## CLI

    go run main.go --port=9000 --hostname=localhost --id=test
    go run main.go --port=9001 --hostname=localhost --id=test1 --ids="http://localhost:9000"
    go run main.go --port=9002 --hostname=localhost --id=test2 --ids="http://localhost:9000"

    add http://localhost:9001
    // test.local > received request
    send test.local message hello world!
    // cannot send to test.local, pending approval
    // test.local > accepted your request
    // test.local > sent message geiaaaa...
    send test.local message hello world!
    // sending as wdfsauj
    // test.local > received message wdfsauj, "test"

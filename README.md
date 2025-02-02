# reconfy

[![Go Report Card](https://goreportcard.com/badge/github.com/andrewheberle/reconfy?logo=go&style=flat-square)](https://goreportcard.com/report/github.com/andrewheberle/reconfy)

This can be used to trigger a webhook when a file changes.

In addition environment variables may be substituted within the file.

## Command Line Options

* `--input`: Input file(s) to watch. Use multiple times or seperate using commas (or both)
* `--output`: Output file for environment variable substitutions (optional)
* `--webhook-url`: URL for webhook on reload (default "http://localhost:8080")
* `--webhook-method`: Method for webhook on reload (default "POST")
* `--debug`: Enable debug logging (default "false")

All command line options may be specified as environment variables in the form of `RECONFY_<option>` such as `RECONFY_WEBHOOK_URL="http://localhost:8080/reload"`

## Watching Multiple Files

It is possible to specify the `--input` option multiple times or provide a comma seperated list of files to watch. 

All provided files will be watched, but only the first will have environment variable substitution performed. 

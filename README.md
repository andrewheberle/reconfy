# reconfy

[![Go Report Card](https://goreportcard.com/badge/github.com/andrewheberle/reconfy?logo=go&style=flat-square)](https://goreportcard.com/report/github.com/andrewheberle/reconfy)

This can be used to trigger a webhook when a file changes.

In addition environment variables may be substituted within the file using Kubernetes syntax.

## Command Line Options

* `--input`: Input file to watch
* `--output`: Output file for environment variable substitutions (optional)
* `--webhook`: URL for webhook on reload (default "http://localhost:8080")
* `--watch-dirs`: Additional directories to watch for changes (optional)
* `--metrics-listen`: Listen address for metrics (optional)
* `--metrics-path`: Path for Prometheus metrics (default "/metrics")
* `--ignore-missing`: Ignore missing environment variables when performing substitutions (default "false")

All command line options may be specified as environment variables in the form of `RECONFY_<option>` such as `RECONFY_WEBHOOK="http://localhost:8080/reload"` or `RECONFY_METRICS_LISTEN=":8080"`.

## Watching Multiple Directories

It is possible to specify the `--watch-dirs` option multiple times or provide a comma seperated list of additional directories to watch for changes. 

All provided directories will be watched for changes and trigger reloads, but no environment variable substitution is performed on these files. 

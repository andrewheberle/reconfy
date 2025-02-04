# reconfy

[![Go Report Card](https://goreportcard.com/badge/github.com/andrewheberle/reconfy?logo=go&style=flat-square)](https://goreportcard.com/report/github.com/andrewheberle/reconfy)

This can be used to trigger a webhook when a file changes.

In addition environment variables may be substituted within the file using Kubernetes syntax. For example, given the input file below:

```yaml
this: is a static value
whileThis: $(FROM_THIS_ENV_VAR)
```

Would result in the following output:

```yaml
this: is a static value
whileThis: came from an env var
```

Assuming the following command was executed:

```sh
env FROM_THIS_ENV_VAR="came from an env var" reconfy --input ./input.yml --output ./output.yml
```

## Command Line Options

* `--input`: Input file to watch
* `--output`: Output file for environment variable substitutions (optional)
* `--webhook`: URL for webhook on reload (default "http://localhost:8080")
* `--watchdirs`: Additional directories to watch for changes (optional)
* `--metrics.listen`: Listen address for metrics (optional)
* `--metrics.path`: Path for Prometheus metrics (default "/metrics")
* `--ignoremissing`: Ignore missing environment variables when performing substitutions (default "false")
* `--config`: Configuration file to load (supports multiple reloaders)

All command line options may be specified as environment variables in the form of `RECONFY_<option>` such as `RECONFY_WEBHOOK="http://localhost:8080/reload"` or `RECONFY_METRICS_LISTEN=":8080"`.

## Watching Multiple Directories

It is possible to specify the `--watch-dirs` option multiple times or provide a comma seperated list of additional directories to watch for changes. 

All provided directories will be watched for changes and trigger reloads, but no environment variable substitution is performed on these files. 

## Using a configuration file

A configuration file may be specified which allows the use of multiple reloaders using the following syntax:

```yaml
reloaders:
  - name: first
    input: examples/input1.yml
    output: examples/output1.yml
    webhook: http://localhost:8080
  - name: second
    input: examples/input2.yml
    output: examples/output2.yml
    webhook: http://localhost:8081
```

The `name` is optional however it is recommended as this is added to log entries for that reloader.

It is important to ensure that the input and output locations for multiple reloaders do not overlap as this would cause an infinite loop of webhook triggers.

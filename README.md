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

Please note that the substitution of variables can be anywhere in a file or any type, not just the values in a YAML document.

## Container

```sh
docker run -e FROM_THIS_ENV_VAR="some value" -v /path/to/config:/config gcr.io/andrewheberle/reconfy:v0.5.4 --input /config/input.yml --output /config/output.yml 
```

## Command Line Options

* `--config`: Configuration file to load (supports multiple reloaders)
* `--ignoremissing`: Ignore missing environment variables when performing substitutions
* `--input`: Input file to watch
* `--metrics.listen`: Listen address for metrics
* `--metrics.path`: Path for Prometheus metrics (default "/metrics")
* `--output`: Output file for environment variable substitutions
* `--watchdirs`: Additional directories to watch for changes
* `--webhook`: URL for webhook on reload (default "http://localhost:8080")

All command line options may be specified as environment variables in the form of `RECONFY_<option>` such as `RECONFY_WEBHOOK="http://localhost:8080/reload"` or `RECONFY_METRICS_LISTEN=":8080"`.

## Watching Multiple Directories

It is possible to specify the `--watchdirs` option multiple times or provide a comma seperated list of additional directories to watch for changes. 

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
    watchdirs:
      - examples/dir1
      - examples/dir2
    webhook: http://localhost:8081
```

The `name` is optional for a single reloader however it is recommended as this is added to log entries for that reloader and also added as the `reloader` label to that reloaders metrics (if enabled). When using multiple reloaders the `name` must be unique.

It is important to ensure that the input and output locations for multiple reloaders do not overlap as this would cause an infinite loop of webhook triggers.

## Metrics

If the `--metrics.listen` flag is provided, Prometheus metrics are exposed at `/metrics` (by default).

This web service also exposes a `/-/healthy` endpoint to check if the service is up and running when enabled.

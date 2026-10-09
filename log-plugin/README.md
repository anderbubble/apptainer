# Log Plugin

This logging plugin writes an audit log entry to syslog for every `apptainer`
command, except `apptainer plugin ...` commands. Entries are written before
the command runs, so failed and rejected commands are logged too. Multiple
outputs can be configured in a YAML configuration file, each with its own
template for the log entries.

The plugin was ported from the SingularityCE `log-plugin`. Its configuration
and template files work unchanged with this plugin.

## Building and installing

A Go plugin only loads into an `apptainer` binary built from the same source
tree, with the same Go toolchain, and with plugin support enabled (the
default for source builds, but not with `mconfig --reproducible`).

For developers, a `Makefile` is provided to compile the plugin, and install
or uninstall it along with the default config and template files. Apptainer
must already be installed from this source tree, which must still be in
place.

```sh
# Build the plugin
make

# Install the plugin, config.yml and templates
sudo make install

# Uninstall the plugin
sudo make uninstall

# Clean up build files
make clean
```

## Configuration

Plugin configuration is placed in the `plugins/apptainer.org/log-plugin`
subdirectory of Apptainer's configuration directory, e.g.:

```sh
/usr/local/etc/apptainer/plugins/apptainer.org/log-plugin/
```

`config.yml` lists the outputs:

```yaml
output:
- name: System logger
  type: syslog
  tag: apptainer
  facility: user
  template: default.tpl
```

Each output has:

- `name`: a name used in warning messages.
- `type`: the output type. Only `syslog` is supported.
- `template`: a template file, relative to the configuration directory.
- `tag` (optional): the syslog tag. Defaults to `apptainer`.
- `facility` (optional): the syslog facility name, such as `user`, `daemon`,
  `auth`, `authpriv`, or `local0` to `local7`. Defaults to `user`. An output
  with an unknown facility is skipped with a warning.

Each output writes one message per command, at severity `notice`. Nothing is
written if the template output is empty. Problems with the configuration
only print a warning, even with `--quiet` or `--silent`; they never stop the
command from running.

`config.yml`, `default.tpl` and `mpi-rank0-only.tpl` are copied to the
plugin configuration directory by `make install`. `mpi-rank0-only.tpl` only
logs from rank 0 of an Open MPI or PMI job.

The example `config.yml` and `default.tpl` generate log messages in the
format:

<!-- markdownlint-disable line-length -->
```sh
Oct 09 15:02:18 host1 apptainer[2505827]: uid=1000 gid=1000 user="alice" group="alice" exe="/usr/bin/apptainer" version="1.5.4-1.el9" cwd="/home/alice/test" command="run" args=["alpine_latest.sif"] flags=["userns=\"true\""] image="/home/alice/test/alpine_latest.sif" sif-uuid="46bff61b-7129-4707-9d54-c36c7fb09c04" LOGNAME="alice"
```
<!-- markdownlint-enable line-length -->

## Templates

Templates are Go `text/template` files. The following fields are available.
Environment variables that are not set render as empty strings. `Flags` only
lists flags given on the command line; flags set through `APPTAINER_*`
environment variables are not included, so log those from `Env`.

```go
// UID is the numeric user id of the account that ran apptainer
UID int
// GID is the numeric group id of the account that ran apptainer
GID int
// User is the text user name of the account that ran apptainer
User string
// Group is the text group name of the account that ran apptainer
Group string
// Executable is the full path to the apptainer executable that was run
Executable string
// Version is the version number of the apptainer executable that was run
Version string
// CWD is the current working directory
CWD string
// Command is the top level command passed to apptainer, e.g. run or build
Command string
// Args is a list of arguments passed to the command
Args []string
// Flags is a list of flags set on the command line
Flags []string
// Image is the path / URL of the container image, if applicable
Image string
// SIFUUID is the unique identifier (UUID) of the SIF image, if applicable
SIFUUID string
// Env gives access to log arbitrary environment variables.
// Map of NAME -> VALUE
Env map[string]string
```

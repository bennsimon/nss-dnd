nss-dnd
=========

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)  [![Go](https://github.com/bennsimon/nss-dnd/actions/workflows/go.yaml/badge.svg?branch=main)](https://github.com/bennsimon/nss-dnd/actions/workflows/go.yaml)

Implementation of a nss module in Go with flexibility on host matching. It currently supports static (includes
wildcard support), cname, and external query(api) matching.

## Install & run from source

- Builds the shared objects and module.
- Copies it to the `/lib/$(shell uname -m)-linux-gnu` directory.
- Add `dnd` to your `/etc/nsswitch.conf` file in the `hosts:` line (after `files`).

````shell
make install
````

## Build (shared objects and module) only

- Builds the shared objects and module.

````shell
make build
````

## Configuration

The module uses a set of rules to map `hostname` to `IP`. These rules are stored in a yaml file. The yaml file should be
copied located at `/etc/nss_dnd_rules.yaml` by default or update the location with `NSS_DND_CONFIG_FILE_PATH` linux
environment variable. See example [nss_dnd_rules.yaml.example](nss_dnd_rules.yaml.example).

Logs are written to syslog.

Built out of curiosity!

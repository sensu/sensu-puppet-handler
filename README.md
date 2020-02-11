# Sensu Puppet Keepalive Handler

- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Handler definition](#handler-definition)
  - [Check definition](#check-definition)
- [Installation from source and
  contributing](#installation-from-source-and-contributing)

## Overview

The [Sensu Puppet Keepalive Handler][0] is a [Sensu Event Handler][3] that will
delete an entity with a failing keepalive check when its corresponding
[Puppet][2] node no longer exists or is deregistered.

## Usage examples

Help:

```
Usage:
  sensu-puppet-handler [flags]
  sensu-puppet-handler [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
      --cacert string              path to the site's Puppet CA certificate PEM file
      --cert string                path to the SSL certificate PEM file signed by your site's Puppet CA
  -e, --endpoint string            the PuppetDB API endpoint (URL). If an API path is not specified, /pdb/query/v4/nodes/ will be used
  -h, --help                       help for sensu-puppet-handler
      --insecure-skip-tls-verify   skip SSL verification
      --key string                 path to the private key PEM file for that certificate
  -a, --sensu-api-key string       The Sensu API key
  -u, --sensu-api-url string       The Sensu API URL (default "http://localhost:8080")
  -c, --sensu-ca-cert string       The Sensu Go CA Certificate
```

## Configuration

### Asset registration

Assets are the best way to make use of this handler. If you're not using an asset, please consider doing so! If you're using sensuctl 5.13 or later, you can use the following command to add the asset:

`sensuctl asset add sensu/sensu-puppet-handler`

If you're using an earlier version of sensuctl, you can download the asset
definition from [this project's Bonsai Asset Index
page](https://bonsai.sensu.io/assets/sensu/sensu-puppet-handler).

### Handler definition

Create the handler using the following handler definition:

```yml
---
api_version: core/v2
type: Handler
metadata:
  namespace: default
  name: sensu-puppet-handler
spec:
  type: pipe
  command: sensu-puppet-handler
  timeout: 10
  env_vars:
  - PUPPET_ENDPOINT=https://puppetdb-host:8081
  - PUPPET_CERT=/path/to/puppet/cert.pem
  - PUPPET_KEY=/path/to/puppet/key.pem
  - PUPPET_CA_CERT=/path/to/puppet/ca.pem
  filters:
  - is_incident
  runtime_assets:
  - sensu/sensu-puppet-handler
  secrets:
  - name: SENSU_API_KEY
    secret: sensu-api-key
```

and then add the handler to the keepalive handler set:

``` yml
---
api_version: core/v2
type: Handler
metadata:
  name: keepalive
  namespace: default
spec:
  handlers:
  - sensu-puppet-handler
  type: set
```


### Check definition

No check definition is needed. This handler will only trigger on keepalive
events after it is added to the keepalive handler set.

## Installing from source and contributing

Download the latest version of the sensu-puppet-handler from [releases][4],
or create an executable script from this source.

### Compiling

From the local path of the sensu-puppet-handler repository:
```
go build
```

To contribute to this plugin, see [CONTRIBUTING](https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md)

[0]: https://github.com/sensu/sensu-puppet-handler
[1]: https://github.com/sensu/sensu-go
[2]: https://puppet.com/
[3]: https://docs.sensu.io/sensu-go/latest/reference/handlers/#how-do-sensu-handlers-work
[4]: https://github.com/sensu/sensu-puppet-handler/releases

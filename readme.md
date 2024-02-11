[![Build Status](https://github.com/iappx/external_traefik_config_plugin/workflows/Main/badge.svg?branch=master)](https://github.com/iappx/external_traefik_config_plugin/actions)

# External Traefik Config Provider

The plugin allows you to get routers of third-party instances traefik by api
and send requests to them through the creation of a separate service in the base instance

## Usage

For a plugin to be active for a given Traefik instance, it must be declared in the static configuration.

### Configuration

The plugin can be configured in a static traefik configuration:

```yaml
# Static configuration

experimental:
  plugins:
    external-config:
      moduleName: github.com/iappx/external-traefik-config-plugin
      version: v0.0.1

providers:
  plugin:
    external-config:
      pollInterval: 2s
      instanceMap:
        another-instance:
          apiHost: http://dashboard.traefik-instance.com
          entryPoints:
            - https
          credentials:
            username: user
            password: password
          service:
            loadBalancer:
              servers:
                - url: http://192.168.0.1:10080
              passHostHeader: true
```

#### Description of configuration fields

`pollinterval` - Instance polling period

`instanceMap` - An object of type `instanceName: options`, in which the settings for requesting external configurations are specified

`intance-name.apiHost` - The host on which the traefik api is accessed

`intance-name.entryPoints` - Entrypoints to which the created routers will be referenced

`intance-name.credentials` - Username and password for basic authorization in traefik api

`intance-name.service` - Provider-created service settings. The same as Http service settings in traefik
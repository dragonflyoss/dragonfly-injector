# dragonfly-injector

A Kubernetes Mutating Admission Webhook for automatic P2P capability injection in Dragonfly. This project simplifies Kubernetes Pod configuration by automating the injection of Dragonfly's P2P proxy settings, dfdaemon socket mounts, and CLI tools through annotation-based policies.

## Description

Refer to the [documentation](https://github.com/suiyuanla/design/tree/main/systems-analysis/webhook-based-automated-p2p-client-injection) for more details.

## Getting Started

### Deploy

```sh
kubectl apply -f https://raw.githubusercontent.com/dragonflyoss/dragonfly-injector/main/dist/install.yaml
```

### Configure

There are two ways to configure the injector:

**Method 1: Edit the configuration file directly**

Before `kubectl apply`, edit the ConfigMap named `dragonfly-injector-inject-config` in `dist/install.yaml`.

**Method 2: Use kubectl to edit the config**

After `kubectl apply`, you can edit the ConfigMap named `dragonfly-injector-inject-config` in `dragonfly-injector-system` namespace.

```sh
kubectl -n dragonfly-injector-system edit configmap dragonfly-injector-inject-config
```

Default configuration (reload time: 15s):
```yaml
config.yaml: |
  enable: true       # Whether to enable injection
  proxy_port: 4001   # The proxy port of Dragonfly Dfdaemon
  cli_tools_image: dragonflyoss/toolkits:latest # The image used for injection
  cli_tools_dir_path: /dragonfly-tools          # The tools directory path in the image
```

For more details, please refer to the [docs/install.md](docs/install.md).
# Altinity auto node-label

This controller automatically applies Kubernetes labels and taints to nodes based on special node labels.

## Supported Node Labels

- `altinity.cloud/auto-taint`: If set to `clickhouse` or `zookeeper`, applies a dedicated taint to the node. See below for details.
- `altinity.cloud/auto-taints`: (Deprecated) Comma-separated list of taints to apply to the node. Format: `key1=value1:effect1,key2:effect2,...`
- `altinity.cloud/auto-zone`: If set, this label will be automatically applied to the node as `topology.kubernetes.io/zone=<value>`. This is useful for custom or non-standard zone labeling.

## Taint Logic for `altinity.cloud/auto-taint`

If the node label `altinity.cloud/auto-taint` is set, the controller will apply a dedicated taint as follows:

- If the value is `clickhouse`, the node will get the taint:
  - `dedicated=clickhouse:NoSchedule`
- If the value is `zookeeper`, the node will get the taint:
  - `dedicated=zookeeper:NoSchedule`
- Any other value will be ignored and logged as an error.

## Examples

### Example: Applying Dedicated Taints

Add the following label to a node:

```
altinity.cloud/auto-taint: "clickhouse"
```

This will apply the following taint to the node:
- `dedicated=clickhouse:NoSchedule`

Add the following label to a node:

```
altinity.cloud/auto-taint: "zookeeper"
```

This will apply the following taint to the node:
- `dedicated=zookeeper:NoSchedule`

### Example: Applying Labels

Add the following label to a node:

```
altinity.cloud/auto-labels: "env=production,team=platform,region=us-west"
```

This will apply the following labels to the node:
- `env=production`
- `team=platform`
- `region=us-west`

### Example: Applying Zone Label

Add the following label to a node:

```
altinity.cloud/auto-zone: "eu-central-1a"
```

This will apply the following label to the node:
- `topology.kubernetes.io/zone=eu-central-1a`

### Example: Applying Taints

Add the following label to a node:

```
altinity.cloud/auto-taints: "key1=value1:NoSchedule,key2:PreferNoSchedule,key3=value3:NoExecute"
```

This will apply the following taints to the node:
- `key1=value1:NoSchedule`
- `key2:PreferNoSchedule`
- `key3=value3:NoExecute`

#### Supported Taint Effects
- `NoSchedule`
- `PreferNoSchedule`
- `NoExecute`

### Notes
- Spaces are ignored around keys, values, and separators.
- Empty values are supported (e.g., `debug=` or `key1=:NoSchedule`).
- Invalid formats will be ignored and logged as errors.

# Altinity auto node-label

This controller automatically applies Kubernetes labels and taints to nodes based on special node labels.

## Supported Node Labels

- `altinity.cloud/auto-taint`: If set to `clickhouse` or `zookeeper`, applies a dedicated taint to the node. See below for details.
- `altinity.cloud/auto-zone`: If set, this label will be automatically applied to the node as `topology.kubernetes.io/zone=<value>`. This is useful for custom or non-standard zone labeling.
- The controller always adds the label `altinity.cloud/use=anywhere` to every node.

## Taint Logic for `altinity.cloud/auto-taint`

If the node label `altinity.cloud/auto-taint` is set, the controller will apply a dedicated taint as follows:

- If the value is `clickhouse`, the node will get the taint:
  - `dedicated=clickhouse:NoSchedule`
- If the value is `zookeeper`, the node will get the taint:
  - `dedicated=zookeeper:NoSchedule`
- Any other value will be ignored and logged as an error.

## Example Usage

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

### Example: Applying Zone Label

Add the following label to a node:

```
altinity.cloud/auto-zone: "eu-central-1a"
```

This will apply the following label to the node:
- `topology.kubernetes.io/zone=eu-central-1a`

### Example: Always Applied Label

Every node processed by the controller will always have the following label applied:
- `altinity.cloud/use=anywhere`

---

**Note:**
- The following labels are NOT currently supported by the controller and are ignored if set:
  - `altinity.cloud/auto-taints`
  - `altinity.cloud/auto-labels`
- Only the features described above are implemented.

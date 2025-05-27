# node-label-controller

A Kubernetes controller that automatically applies zone labels to nodes based on custom labels.

## Overview

This controller watches for nodes with the `altinity.cloud/auto-zone` label and automatically applies the standard Kubernetes zone labels to those nodes. This is useful for environments where nodes need custom zone assignment that differs from the cloud provider's default zone detection.

## How it Works

The controller continuously monitors all nodes in the cluster. When it detects a node with the `altinity.cloud/auto-zone` label, it automatically applies both current and legacy Kubernetes zone labels with the specified value.

## Supported Labels

### Input Label
- `altinity.cloud/auto-zone`: The desired zone value for the node

### Applied Labels
When a node has the `altinity.cloud/auto-zone` label, the controller will automatically apply:
- `topology.kubernetes.io/zone`: The current standard Kubernetes zone label
- `failure-domain.beta.kubernetes.io/zone`: The legacy zone label for backward compatibility

## Example

To assign a node to zone `eu-central-1a`, apply the following label:

```bash
kubectl label node <node-name> altinity.cloud/auto-zone=eu-central-1a
```

The controller will then automatically apply:
- `topology.kubernetes.io/zone=eu-central-1a`
- `failure-domain.beta.kubernetes.io/zone=eu-central-1a`

## Deployment

Deploy the controller using the provided manifests:

```bash
kubectl apply -f deploy/
```

This will create:
- A deployment running the controller
- RBAC permissions for the controller to read and update nodes


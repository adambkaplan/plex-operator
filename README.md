# Plex Operator

Yet another operator to install the Plex media server on a Kubernetes cluster.

## Run Plex on Your Kubernetes Cluster

`plex-operator` makes it simple to run the [Plex Media Server](https://support.plex.tv/articles/categories/plex-media-server/) on your Kubernetes cluster.
Use the `PlexMediaServer` custom resource to define how you want to configure Plex, and the operator will take care of the rest.
Customize the installation by bringing your own storage and declaring how Plex services can be exposed outside of the cluster.

## Install the Operator

1. Clone this repository - `git clone https://github.com/adambkaplan/plex-operator.git`.
2. Make sure kubectl is installed and configured to connect to your Kubernetes cluster as cluster admin.
3. Run `make deploy`

## Deploy Plex

Install the latest Plex Media Server version with your claim token by creating an instance of the `PlexMediaServer` custom resource:

```bash
$ kubectl apply -f - << EOF
kind: PlexMediaServer
apiVersion: plex.adambkaplan.com/v1alpha1
metadata:
  name: plex
spec:
  claimToken: <claim-token>
EOF
```

*Note* - to unlock most features in Plex you need to provide a claim token.
You can obtain a claim token at [https://www.plex.tv/claim](https://www.plex.tv/claim).

Read [Deploy Plex](docs/deploy-plex.md) to see which configuration options are available.

## Why "yet another" Plex Operator?

I am not the first person to create a Kubernetes operator for Plex.
A lot of this work was inspired by [kubealex/k8s-mediaserver-operator](https://github.com/kubealex/k8s-mediaserver-operator) and [munnerz/kube-plex](https://github.com/munnerz/kube-plex).

Read the [FAQ](docs/faq.md) to understand why I decided to write this particular operator.

## Contributing

This project is mainly for my enjoyment and experimentation with operator-sdk.
Feel free to contribute by submitting a pull request!

## License

Copyright Adam B Kaplan

SPDX-License-Identifier: Apache-2.0

# shardlister

## Install


```shell
git clone git@github.com:jemisonf/shardlister.git && cd shardlister
go install
```

## Usage

A note about auth: `shardlister` uses your kubectl context and your argocd context for authentication. Before using, you should do the following:
* Authenticate to your ArgoCD cluster
* Login to ArgoCD: `argocd login --sso argocd.link.to.my.instance`

You can also use a custom ArgoCD context and `kubectl` context for auth. See `shardlister --help` for more details.

```shell
shardlister --shards 12 # list all clusters for all shards
shardlister --kubeconfig ./custom-kubeconfig --shards 12 # use a custom kubeconfig
shardlister --namespace custom-ns --shards 12 # for argocd clusters not in the argocd namespace
shardlister --shards 12 shard 1 # list all clusters for shard 1
shardlister --shards 12 shard apps 1 # list all apps for shard 1
shardlister --shards 12 shard apps 1 --count # list the number of apps in shard 1
shardlister cluster apps https://kubernetes.default.svc --count # list the number of apps in the default cluster
```



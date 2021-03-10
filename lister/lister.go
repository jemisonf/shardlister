package lister

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/argoproj/argo-cd/controller/sharding"
	argocdclient "github.com/argoproj/argo-cd/pkg/apiclient"
	"github.com/argoproj/argo-cd/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/settings"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Lister struct {
	db        db.ArgoDB
	appClient application.ApplicationServiceClient
	cache     cache.Cache
}

func NewLister(ctx context.Context, namespace string, kubeconfigPath string, argoCDConfigPath string) (*Lister, error) {
	cache := cache.New(30*time.Second, 5*time.Minute)
	kubeconfig, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error reading kubeconfig path %s: %v", kubeconfigPath, err)
	}

	restConf, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	kubeclientset, err := kubernetes.NewForConfig(restConf)

	if err != nil {
		return nil, err
	}

	settingsMgr := settings.NewSettingsManager(ctx, kubeclientset, namespace)
	argoDB := db.NewDB(namespace, settingsMgr, kubeclientset)

	argoCDClient := argocdclient.NewClientOrDie(&argocdclient.ClientOptions{GRPCWeb: true, ConfigPath: argoCDConfigPath})
	_, appClient := argoCDClient.NewApplicationClientOrDie()

	l := Lister{db: argoDB, appClient: appClient, cache: *cache}

	return &l, nil
}

func (l *Lister) ListClusters(ctx context.Context) ([]v1alpha1.Cluster, error) {
	clusterList, err := l.db.ListClusters(ctx)

	if err != nil {
		return nil, err
	}

	return clusterList.Items, nil
}

func (l *Lister) ListShardClusters(ctx context.Context, replicas int, shard int) ([]v1alpha1.Cluster, error) {
	clusters, err := l.ListClusters(ctx)

	if err != nil {
		return nil, err
	}

	filteredClusters := []v1alpha1.Cluster{}

	filter := sharding.GetClusterFilter(replicas, shard)

	for _, cluster := range clusters {
		if filter(&cluster) {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	return filteredClusters, nil
}

func (l *Lister) appsFromCache(ctx context.Context) (*v1alpha1.ApplicationList, error) {
	if list, found := l.cache.Get("apps"); found {
		return list.(*v1alpha1.ApplicationList), nil
	}

	log.Infof("getting clusters from API")
	apps, err := l.appClient.List(ctx, &application.ApplicationQuery{})

	if err != nil {
		return nil, err
	}

	l.cache.Set("apps", apps, 30*time.Second)

	return apps, nil
}

func (l *Lister) ListShardApps(ctx context.Context, replicas int, shard int) ([]v1alpha1.Application, error) {
	clusters, err := l.ListShardClusters(ctx, replicas, shard)

	if err != nil {
		return nil, fmt.Errorf("error listing clusters for shard: %v", err)
	}

	apps, err := l.appsFromCache(ctx)

	if err != nil {
		return nil, fmt.Errorf("error listing application: %v", err)
	}

	shardApps := []v1alpha1.Application{}

	for _, app := range apps.Items {
		for _, cluster := range clusters {
			if app.Spec.Destination.Name == cluster.Name || app.Spec.Destination.Server == cluster.Server {
				shardApps = append(shardApps, app)
			}
		}
	}

	return shardApps, nil
}

func (l *Lister) ListClusterApps(ctx context.Context, clusterName string) ([]v1alpha1.Application, error) {
	clusters, err := l.ListClusters(ctx)

	if err != nil {
		return nil, fmt.Errorf("error getting clusters: %v", err)
	}

	var targetCluster v1alpha1.Cluster
	var foundCluster bool

	for _, cluster := range clusters {
		if cluster.Name == clusterName || cluster.Server == clusterName {
			foundCluster = true
			targetCluster = cluster
		}
	}

	if !foundCluster {
		return nil, fmt.Errorf("cluster %s not found", clusterName)
	}

	apps, err := l.appClient.List(ctx, &application.ApplicationQuery{})

	if err != nil {
		return nil, fmt.Errorf("error getting apps: %v", err)
	}

	clusterApps := []v1alpha1.Application{}

	for _, app := range apps.Items {
		if app.Spec.Destination.Server == targetCluster.Server || app.Spec.Destination.Name == targetCluster.Name {
			clusterApps = append(clusterApps, app)
		}
	}

	return clusterApps, nil
}

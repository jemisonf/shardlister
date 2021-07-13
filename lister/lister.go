package lister

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/argoproj/argo-cd/controller/sharding"
	argocdclient "github.com/argoproj/argo-cd/pkg/apiclient"
	"github.com/argoproj/argo-cd/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/pkg/apiclient/cluster"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/settings"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Lister struct {
	db            db.ArgoDB
	appClient     application.ApplicationServiceClient
	cache         cache.Cache
	clusterClient cluster.ClusterServiceClient
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
	_, clusterClient := argoCDClient.NewClusterClientOrDie()

	l := Lister{db: argoDB, appClient: appClient, cache: *cache, clusterClient: clusterClient}

	return &l, nil
}

func (l *Lister) ListClusters(ctx context.Context, replicas int) ([]v1alpha1.Cluster, error) {
	// clusterClient gets the list of clusters from the ArgoCD API and gives you the cached app count,
	// while db.ListClusters gets the cluster secret and gives you the ID and the shard.
	clusterList, err := l.clusterClient.List(ctx, &cluster.ClusterQuery{})
	dbClusterList, err := l.db.ListClusters(ctx)

	if err != nil {
		return nil, fmt.Errorf("error listing clusters: %v", err)
	}

	clusters := []v1alpha1.Cluster{}

	for _, cluster := range clusterList.Items {
		// find matching cluster in dbClusterList
		for _, dbCluster := range dbClusterList.Items {
			if dbCluster.Name == cluster.Name && dbCluster.Server == cluster.Server {
				cluster.Shard = dbCluster.Shard
				cluster.ID = dbCluster.ID
			}
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (l *Lister) ListShardClusters(ctx context.Context, replicas int, shard int) ([]v1alpha1.Cluster, error) {
	clusters, err := l.ListClusters(ctx, replicas)

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

	log.Infof("getting apps from API")
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

func (l *Lister) ListClusterApps(ctx context.Context, replicas int, clusterName string) ([]v1alpha1.Application, error) {
	clusters, err := l.ListClusters(ctx, replicas)

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

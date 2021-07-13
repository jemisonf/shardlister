package cmd

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
	"source.datanerd.us/vind-playground/shardlister/lister"
)

type globalOpts struct {
	lister           *lister.Lister
	namespace        string
	kubeconfigPath   string
	argoCDConfigPath string
	numShards        int
}

var globals = globalOpts{}

var rootCmd = &cobra.Command{
	Use:   "shardlister --shards SHARDS",
	Short: "list clusters for all controller shards",
	Example: `shardlister --shards 12
shardlister --shards 12 shard 2`,
	Run: listAll,
}

func Execute() error {
	return rootCmd.Execute()
}

func printShard(cmd *cobra.Command, shard int, clusters []v1alpha1.Cluster) {
	formatString := "%-8s%-30s%-8s\n"
	cmd.Printf(formatString, "SHARD", "CLUSTER", "CACHED APP COUNT")
	for _, cluster := range clusters {
		cmd.Printf(formatString, strconv.Itoa(shard), cluster.Name, strconv.Itoa(int(cluster.Info.ApplicationsCount)))
	}
}

func printShards(cmd *cobra.Command, shards int, shardClusters map[int][]v1alpha1.Cluster) {
	formatString := "%-8s%-30s%-8s\n"
	cmd.Printf(formatString, "SHARD", "CLUSTER", "CACHED APP COUNT")
	for shard := 0; shard < shards; shard++ {
		clusters := shardClusters[shard]
		for _, cluster := range clusters {
			cmd.Printf(formatString, strconv.Itoa(shard), cluster.Name, strconv.Itoa(int(cluster.Info.ApplicationsCount)))
		}
	}

}

func listAll(cmd *cobra.Command, args []string) {
	if globals.numShards < 1 {
		cmd.PrintErr("cannot have less than 1 shard\n")
		os.Exit(1)
	}

	shardClusters := map[int][]v1alpha1.Cluster{}
	for i := 0; i < globals.numShards; i++ {
		clusters, err := globals.lister.ListShardClusters(context.Background(), globals.numShards, i)

		if err != nil {
			cmd.PrintErrf("error listing clusters for shard %d: %v", i, err)
		}

		shardClusters[i] = clusters
	}

	printShards(cmd, globals.numShards, shardClusters)
}

func init() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	defaultKubeConfigPath := fmt.Sprintf("%s/.kube/config", user.HomeDir)
	defaultArgoCDConfigPath := fmt.Sprintf("%s/.argocd/config", user.HomeDir)
	rootCmd.PersistentFlags().IntVar(&globals.numShards, "shards", 0, "number of total shards in the cluster")
	rootCmd.PersistentFlags().StringVar(&globals.namespace, "namespace", "argocd", "namespace where argocd cluster secrets are located")
	rootCmd.PersistentFlags().StringVar(&globals.kubeconfigPath, "kubeconfig", defaultKubeConfigPath, "path to kubeconfig to use for authentication")
	rootCmd.PersistentFlags().StringVar(&globals.argoCDConfigPath, "argocd-config", defaultArgoCDConfigPath, "path to argoCD config to use for authentication")
	l, err := lister.NewLister(context.Background(), globals.namespace, globals.kubeconfigPath, globals.argoCDConfigPath)

	if err != nil {
		rootCmd.Printf("error creating argo client: %v", err)
		os.Exit(1)
	}

	globals.lister = l
	rootCmd.AddCommand(shardCmd)
	rootCmd.AddCommand(appsCmd)
	shardCmd.AddCommand(shardAppsCmd)
	shardAppsCmd.PersistentFlags().BoolVar(&shardAppsOpts.count, "count", false, "only show the number of applications")
	appsCmd.PersistentFlags().BoolVar(&appsOpts.count, "count", false, "only show the number of applications")

	rootCmd.AddCommand(clusterCmd)
	clusterCmd.AddCommand(clusterAppsCmd)
	clusterAppsCmd.PersistentFlags().BoolVar(&clusterAppsOpts.count, "count", false, "only show the number of applications")
}

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/user"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/jemisonf/shardlister/lister"
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
	cmd.Printf("shard %d:\n", shard)
	for _, cluster := range clusters {
		cmd.Printf("\t%s\n", cluster.Name)
	}
}

func listAll(cmd *cobra.Command, args []string) {
	if globals.numShards < 1 {
		cmd.PrintErr("cannot have less than 1 shard\n")
		os.Exit(1)
	}

	for i := 0; i < globals.numShards; i++ {
		clusters, err := globals.lister.ListShardClusters(context.Background(), globals.numShards, i)

		if err != nil {
			cmd.PrintErrf("error listing clusters for shard %d: %v", i, err)
		}

		printShard(cmd, i, clusters)
	}
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

package cmd

import (
	"context"
	"os"
	"strconv"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
)

var shardCmd = &cobra.Command{
	Use:   "shard SHARD",
	Short: "list all clusters for a given shard",
	Args:  cobra.ExactArgs(1),
	Run:   shard,
}

var shardAppsCmd = &cobra.Command{
	Use:   "apps SHARD",
	Short: "list all apps for a shard",
	Args:  cobra.ExactArgs(1),
	Run:   shardApps,
}

func shard(cmd *cobra.Command, args []string) {
	shard, err := strconv.Atoi(args[0])
	if err != nil {
		cmd.PrintErr("shard is not a number\n")
		os.Exit(1)
	}

	clusters, err := globals.lister.ListShardClusters(context.Background(), globals.numShards, shard)
	if err != nil {
		cmd.PrintErrf("error getting clusters: %v\n", err)
	}

	printShard(cmd, shard, clusters)
}

type shardAppsArgs struct {
	count bool
}

var shardAppsOpts = shardAppsArgs{}

func shardApps(cmd *cobra.Command, args []string) {
	shard, err := strconv.Atoi(args[0])
	if err != nil {
		cmd.PrintErr("shard is not a number\n")
		os.Exit(1)
	}

	apps, err := globals.lister.ListShardApps(context.Background(), globals.numShards, shard)

	shardApps := map[int][]v1alpha1.Application{
		shard: apps,
	}

	if shardAppsOpts.count {
		printShardAppsCount(cmd, globals.numShards, shardApps)
		return
	}

	printShardApps(cmd, globals.numShards, shardApps)
}

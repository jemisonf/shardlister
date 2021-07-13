package cmd

import (
	"context"
	"strconv"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "list all apps for each shard",
	Run:   apps,
}

type appsArgs struct {
	count bool
}

var appsOpts = appsArgs{}

func printShardApps(cmd *cobra.Command, shards int, shardApps map[int][]v1alpha1.Application) {
	formatString := "%-8s%-64s\n"

	cmd.Printf(formatString, "SHARD", "APP NAME")

	for shard := 0; shard < shards; shard++ {
		apps, found := shardApps[shard]

		if !found {
			continue
		}

		for _, app := range apps {
			cmd.Printf(formatString, strconv.Itoa(shard), app.ObjectMeta.Name)
		}
	}
}

func printShardAppsCount(cmd *cobra.Command, shards int, shardApps map[int][]v1alpha1.Application) {
	formatString := "%-8s%-8s\n"

	cmd.Printf(formatString, "SHARD", "APP COUNT")

	for shard := 0; shard < shards; shard++ {
		apps, found := shardApps[shard]

		if !found {
			continue
		}
		cmd.Printf(formatString, strconv.Itoa(shard), strconv.Itoa(len(apps)))

	}

}
func apps(cmd *cobra.Command, args []string) {
	appResults := map[int][]v1alpha1.Application{}
	for shard := 0; shard < globals.numShards; shard++ {
		apps, err := globals.lister.ListShardApps(context.Background(), globals.numShards, shard)

		if err != nil {
			cmd.PrintErrf("error listing apps for shard %d: %v\n", shard, err)
		}

		appResults[shard] = apps
	}

	if appsOpts.count {
		printShardAppsCount(cmd, globals.numShards, appResults)
		return
	}

	printShardApps(cmd, globals.numShards, appResults)
}

package cmd

import (
	"context"

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

func apps(cmd *cobra.Command, args []string) {
	appResults := map[int][]v1alpha1.Application{}
	for shard := 0; shard < globals.numShards; shard++ {
		apps, err := globals.lister.ListShardApps(context.Background(), globals.numShards, shard)

		if err != nil {
			cmd.PrintErrf("error listing apps for shard %d: %v\n", shard, err)
		}

		appResults[shard] = apps
	}

	for shard, apps := range appResults {
		if appsOpts.count {
			cmd.Printf("apps for shard %d: \t%d\n", shard, len(apps))
			continue
		}
		cmd.Printf("shard %d:\n", shard)
		for _, app := range apps {
			cmd.Printf("%s\t%s\n", app.ClusterName, app.Name)
		}
	}
}

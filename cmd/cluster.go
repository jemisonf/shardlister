package cmd

import (
	"context"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "commands for a specific cluster",
	Args:  cobra.ExactArgs(1),
}

type clusterAppsArgs struct {
	count bool
}

var clusterAppsOpts clusterAppsArgs

var clusterAppsCmd = &cobra.Command{
	Use:   "apps CLUSTER",
	Short: "list apps for a cluster",
	Long:  `List apps for a cluster using either cluster name or server`,
	Example: `
	shardlister cluster apps us-cute-wall
	shardlister cluster apps https://kubernetes.default.svc
	shardlister cluster apps https://kubernetes.default.svc --count
	`,
	Args: cobra.ExactArgs(1),
	Run:  clusterApps,
}

func clusterApps(cmd *cobra.Command, args []string) {
	clusterName := args[0]
	clusterApps, err := globals.lister.ListClusterApps(context.Background(), globals.numShards, clusterName)

	if err != nil {
		cmd.PrintErrf("error getting apps: %v\n", err)
		os.Exit(1)
	}

	if clusterAppsOpts.count {
		formatString := "%-20%-8s\n"
		cmd.Printf(formatString, "CLUSTER", "APP COUNT")
		cmd.Printf(formatString, clusterName, strconv.Itoa(len(clusterApps)))
		os.Exit(0)
	}

	formatString := "%-64s%-20s%s"
	cmd.Printf(formatString, "APP NAME", "CLUSTER NAME", "CLUSTER SERVER\n")

	for _, app := range clusterApps {
		cmd.Printf(formatString, app.ObjectMeta.Name, app.Spec.Destination.Name, app.Spec.Destination.Server)
	}

}

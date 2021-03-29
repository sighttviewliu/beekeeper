package cmd

import (

	// "github.com/ethersphere/beekeeper/pkg/check"

	"fmt"

	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/spf13/cobra"
)

func (c *command) initCheckPing() *cobra.Command {
	const (
		optionNameDynamic      = "dynamic"
		optionNameSeed         = "seed"
		optionNameStartCluster = "start-cluster"
	)

	var (
		dynamic      bool
		startCluster bool
	)

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Executes ping from all nodes to all other nodes in the cluster",
		Long: `Executes ping from all nodes to all other nodes in the cluster,
and prints round-trip time (RTT) of each ping.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")

			cluster, err := setupCluster(cmd.Context(), cfg, startCluster)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			fmt.Println("cluster", cluster.Name(), "success")

			// var seed int64
			// if cmd.Flags().Changed("seed") {
			// 	seed = c.config.GetInt64(optionNameSeed)
			// } else {
			// 	seed = random.Int64()
			// }
			// buffer := 12

			// checkCtx, checkCancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			// defer checkCancel()

			// checkPing := pingpong.NewPing()
			// checkOptions := check.Options{
			// 	MetricsEnabled: c.config.GetBool(optionNamePushMetrics),
			// 	MetricsPusher:  push.New(c.config.GetString(optionNamePushGateway), namespace),
			// }

			// return check.RunConcurrently(checkCtx, cluster, checkPing, checkOptions, checkStages, buffer, seed)
			return
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().BoolVar(&dynamic, optionNameDynamic, false, "check on dynamic cluster")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating chunks; if not set, will be random")
	cmd.Flags().BoolVar(&startCluster, optionNameStartCluster, false, "start new cluster")

	return cmd
}

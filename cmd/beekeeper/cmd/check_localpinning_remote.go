package cmd

import (
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/localpinning"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"

	"github.com/spf13/cobra"
)

func (c *command) initCheckLocalPinningRemote() *cobra.Command {
	const (
		optionNameDbCapacity = "db-capacity"
		optionNameDivisor    = "capacity-divisor"
		optionNameSeed       = "seed"
	)

	cmdBytes := &cobra.Command{
		Use:   "pin-remote",
		Short: "Checks that a node on the cluster pins remote chunks correctly.",
		Long:  "Checks that a node on the cluster pins remote chunks correctly.",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			return localpinning.CheckRemoteChunksFound(cluster, localpinning.Options{
				NodeGroup:        "bee",
				StoreSize:        c.config.GetInt(optionNameDbCapacity),
				StoreSizeDivisor: c.config.GetInt(optionNameDivisor),
				Seed:             seed,
			})
		},
		PreRunE: c.checkPreRunE,
	}

	cmdBytes.Flags().Int(optionNameDbCapacity, 1000, "DB capacity in chunks")
	cmdBytes.Flags().Int(optionNameDivisor, 3, "divide store size by which value when uploading bytes")
	cmdBytes.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	return cmdBytes
}

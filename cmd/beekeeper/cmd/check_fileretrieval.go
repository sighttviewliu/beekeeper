package cmd

import (
	"errors"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/check/fileretrieval"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/cobra"
)

func (c *command) initCheckFileRetrieval() *cobra.Command {
	const (
		optionNameUploadNodeCount = "upload-node-count"
		optionNameFilesPerNode    = "files-per-node"
		optionNameFileName        = "file-name"
		optionNameFileSize        = "file-size"
		optionNameSeed            = "seed"
		optionNameFull            = "full"
	)

	var (
		full bool
	)

	cmd := &cobra.Command{
		Use:   "fileretrieval",
		Short: "Checks file retrieval ability of the cluster",
		Long: `Checks file retrieval ability of the cluster.
It uploads given number of files to given number of nodes, 
and attempts retrieval of those files from the last node in the cluster.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if c.config.GetInt(optionNameUploadNodeCount) > c.config.GetInt(optionNameNodeCount) {
				return errors.New("bad parameters: upload-node-count must be less or equal to node-count")
			}

			cfg := config.Read("config.yaml")
			cluster, err := setupCluster(cmd.Context(), cfg, false)
			if err != nil {
				return fmt.Errorf("cluster setup: %w", err)
			}

			pusher := push.New(c.config.GetString(optionNamePushGateway), cfg.Cluster.Namespace)

			var seed int64
			if cmd.Flags().Changed("seed") {
				seed = c.config.GetInt64(optionNameSeed)
			} else {
				seed = random.Int64()
			}

			fileSize := round(c.config.GetFloat64(optionNameFileSize) * 1024 * 1024)

			if full {
				return fileretrieval.CheckFull(cluster, fileretrieval.Options{
					NodeGroup:       "bee",
					UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
					FilesPerNode:    c.config.GetInt(optionNameFilesPerNode),
					FileName:        c.config.GetString(optionNameFileName),
					FileSize:        fileSize,
					Seed:            seed,
				}, pusher, c.config.GetBool(optionNamePushMetrics))
			}

			return fileretrieval.Check(cluster, fileretrieval.Options{
				NodeGroup:       "bee",
				UploadNodeCount: c.config.GetInt(optionNameUploadNodeCount),
				FilesPerNode:    c.config.GetInt(optionNameFilesPerNode),
				FileName:        c.config.GetString(optionNameFileName),
				FileSize:        fileSize,
				Seed:            seed,
			}, pusher, c.config.GetBool(optionNamePushMetrics))
		},
		PreRunE: c.checkPreRunE,
	}

	cmd.Flags().IntP(optionNameUploadNodeCount, "u", 1, "number of nodes to upload files to")
	cmd.Flags().IntP(optionNameFilesPerNode, "p", 1, "number of files to upload per node")
	cmd.Flags().String(optionNameFileName, "file", "file name template")
	cmd.Flags().Float64(optionNameFileSize, 1, "file size in MB")
	cmd.Flags().Int64P(optionNameSeed, "s", 0, "seed for generating files; if not set, will be random")
	cmd.Flags().BoolVar(&full, optionNameFull, false, "tries to download from all nodes in the cluster")

	return cmd
}

func round(val float64) int64 {
	if val < 0 {
		return int64(val - 0.5)
	}
	return int64(val + 0.5)
}

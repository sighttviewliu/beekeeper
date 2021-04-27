package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/beeclient/api"
	"github.com/ethersphere/beekeeper/pkg/beekeeper"
	"github.com/ethersphere/beekeeper/pkg/random"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

// Options represents check options
type Options struct {
	ChunksPerNode   int // number of chunks to upload per node
	MetricsPusher   *push.Pusher
	NodeGroup       string // TODO: support multi node group cluster
	Seed            int64
	UploadNodeCount int
}

// NewDefaultOptions returns new default options
func NewDefaultOptions() Options {
	return Options{
		ChunksPerNode:   1,
		MetricsPusher:   nil,
		NodeGroup:       "bee",
		Seed:            random.Int64(),
		UploadNodeCount: 1,
	}
}

// compile check whether Check implements interface
var _ beekeeper.Action = (*Check)(nil)

// Check instance
type Check struct{}

// NewCheck returns new check
func NewCheck() beekeeper.Action {
	return &Check{}
}

var errRetrieval = errors.New("retrieval")

// Check uploads given chunks on cluster and checks pushsync ability of the cluster
func (c *Check) Run(ctx context.Context, cluster *bee.Cluster, opts interface{}) (err error) {
	o, ok := opts.(Options)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	rnds := random.PseudoGenerators(o.Seed, o.UploadNodeCount)
	fmt.Printf("Seed: %d\n", o.Seed)

	if o.MetricsPusher != nil {
		o.MetricsPusher.Collector(uploadedCounter)
		o.MetricsPusher.Collector(uploadTimeGauge)
		o.MetricsPusher.Collector(uploadTimeHistogram)
		o.MetricsPusher.Collector(downloadedCounter)
		o.MetricsPusher.Collector(downloadTimeGauge)
		o.MetricsPusher.Collector(downloadTimeHistogram)
		o.MetricsPusher.Collector(retrievedCounter)
		o.MetricsPusher.Collector(notRetrievedCounter)
		o.MetricsPusher.Format(expfmt.FmtText)
	}

	ng := cluster.NodeGroup(o.NodeGroup)
	overlays, err := ng.Overlays(ctx)
	if err != nil {
		return err
	}

	sortedNodes := ng.NodesSorted()
	lastNodeName := sortedNodes[len(sortedNodes)-1]
	for i := 0; i < o.UploadNodeCount; i++ {
		nodeName := sortedNodes[i]
		for j := 0; j < o.ChunksPerNode; j++ {
			chunk, err := bee.NewRandomChunk(rnds[i])
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}

			t0 := time.Now()
			ref, err := ng.NodeClient(nodeName).UploadChunk(ctx, chunk.Data(), api.UploadOptions{Pin: false})
			if err != nil {
				return fmt.Errorf("node %s: %w", nodeName, err)
			}
			d0 := time.Since(t0)

			uploadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			uploadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d0.Seconds())
			uploadTimeHistogram.Observe(d0.Seconds())

			t1 := time.Now()
			data, err := ng.NodeClient(lastNodeName).DownloadChunk(ctx, ref, "")
			if err != nil {
				return fmt.Errorf("node %s: %w", lastNodeName, err)
			}
			d1 := time.Since(t1)

			downloadedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			downloadTimeGauge.WithLabelValues(overlays[nodeName].String(), ref.String()).Set(d1.Seconds())
			downloadTimeHistogram.Observe(d1.Seconds())

			if !bytes.Equal(chunk.Data(), data) {
				notRetrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
				fmt.Printf("Node %s. Chunk %d not retrieved successfully. Uploaded size: %d Downloaded size: %d Node: %s Chunk: %s\n", nodeName, j, chunk.Size(), len(data), overlays[nodeName].String(), ref.String())
				if bytes.Contains(chunk.Data(), data) {
					fmt.Printf("Downloaded data is subset of the uploaded data\n")
				}
				return errRetrieval
			}

			retrievedCounter.WithLabelValues(overlays[nodeName].String()).Inc()
			fmt.Printf("Node %s. Chunk %d retrieved successfully. Node: %s Chunk: %s\n", nodeName, j, overlays[nodeName].String(), chunk.Address().String())

			if o.MetricsPusher != nil {
				if err := o.MetricsPusher.Push(); err != nil {
					fmt.Printf("node %s: %v\n", nodeName, err)
				}
			}
		}
	}

	return
}

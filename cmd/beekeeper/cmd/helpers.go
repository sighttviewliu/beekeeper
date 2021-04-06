package cmd

import (
	"context"
	"fmt"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/check"
	"github.com/ethersphere/beekeeper/pkg/config"
	"github.com/ethersphere/beekeeper/pkg/k8s"
	"github.com/ethersphere/beekeeper/pkg/stress"
	"golang.org/x/sync/errgroup"
)

func deleteCluster(ctx context.Context, c *config.Config) (err error) {
	var k8sClient *k8s.Client
	if c.Kubernetes.Enable {
		k8sClient, err = setK8SClient(c.Kubernetes.Kubeconfig, c.Kubernetes.InCluster)
		if err != nil {
			return fmt.Errorf("kubernetes client: %w", err)
		}
	}

	cluster := bee.NewCluster(c.Cluster.Name, bee.ClusterOptions{
		APIDomain:           c.Cluster.API.Domain,
		APIInsecureTLS:      c.Cluster.API.InsecureTLS,
		APIScheme:           c.Cluster.API.Scheme,
		DebugAPIDomain:      c.Cluster.DebugAPI.Domain,
		DebugAPIInsecureTLS: c.Cluster.DebugAPI.InsecureTLS,
		DebugAPIScheme:      c.Cluster.DebugAPI.Scheme,
		K8SClient:           k8sClient,
		Namespace:           c.Cluster.Namespace,
		DisableNamespace:    c.Cluster.DisableNamespace,
	})

	for ng, v := range c.Cluster.NodeGroups {
		fmt.Printf("deleting %s node group\n", ng)
		if v.Mode == "bootnode" {
			// add node group to the cluster
			gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
			cluster.AddNodeGroup(ng, gProfile.Export())

			// delete nodes from the node group
			g := cluster.NodeGroup(ng)
			for i := 0; i < len(v.Nodes); i++ {
				nName := v.Nodes[i].Name
				if err := g.DeleteNode(ctx, nName); err != nil {
					return fmt.Errorf("deleting node %s from the node group %s", nName, ng)
				}
			}
		} else {
			// add node group to the cluster
			gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
			cluster.AddNodeGroup(ng, gProfile.Export())

			// delete nodes from the node group
			g := cluster.NodeGroup(ng)
			for i := 0; i < v.Count; i++ {
				nName := fmt.Sprintf("%s-%d", ng, i)
				if err := g.DeleteNode(ctx, nName); err != nil {
					return fmt.Errorf("deleting node %s from the node group %s", nName, ng)
				}
			}
		}
	}

	return
}

func setupCluster(ctx context.Context, c *config.Config, start bool) (cluster *bee.Cluster, err error) {
	var k8sClient *k8s.Client
	if c.Kubernetes.Enable {
		k8sClient, err = setK8SClient(c.Kubernetes.Kubeconfig, c.Kubernetes.InCluster)
		if err != nil {
			return nil, fmt.Errorf("kubernetes client: %w", err)
		}
	}

	cluster = bee.NewCluster(c.Cluster.Name, bee.ClusterOptions{
		APIDomain:           c.Cluster.API.Domain,
		APIInsecureTLS:      c.Cluster.API.InsecureTLS,
		APIScheme:           c.Cluster.API.Scheme,
		DebugAPIDomain:      c.Cluster.DebugAPI.Domain,
		DebugAPIInsecureTLS: c.Cluster.DebugAPI.InsecureTLS,
		DebugAPIScheme:      c.Cluster.DebugAPI.Scheme,
		K8SClient:           k8sClient,
		Namespace:           c.Cluster.Namespace,
		DisableNamespace:    c.Cluster.DisableNamespace,
	})

	if start {
		bootnodes := ""
		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode == "bootnode" {
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				cluster.AddNodeGroup(ng, gProfile.Export())

				// start nodes in the node group
				g := cluster.NodeGroup(ng)
				errGroup := new(errgroup.Group)
				for i := 0; i < len(v.Nodes); i++ {
					nName := v.Nodes[i].Name
					bProfile := c.BeeProfiles[v.BeeConfig]
					bConfig := bProfile.Export()

					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, c.Cluster.Namespace) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					bOptions := bee.NodeOptions{
						Config:       &bConfig,
						ClefKey:      v.Nodes[i].ClefKey,
						ClefPassword: v.Nodes[i].ClefPassword,
						LibP2PKey:    v.Nodes[i].LibP2PKey,
						SwarmKey:     v.Nodes[i].SwarmKey,
					}

					errGroup.Go(func() error {
						return g.AddStartNode(ctx, nName, bOptions)
					})
				}

				if err := errGroup.Wait(); err != nil {
					return nil, fmt.Errorf("starting node group %s: %w", ng, err)
				}
			}
		}

		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode != "bootnode" { // TODO: support standalone nodes
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				gOptions := gProfile.Export()
				nProfile := c.BeeProfiles[v.BeeConfig]
				nConfig := nProfile.Export()
				nConfig.Bootnodes = bootnodes
				gOptions.BeeConfig = &nConfig
				cluster.AddNodeGroup(ng, gOptions)

				// start nodes in the node group
				g := cluster.NodeGroup(ng)
				errGroup := new(errgroup.Group)
				for i := 0; i < v.Count; i++ {
					nName := fmt.Sprintf("%s-%d", ng, i)

					errGroup.Go(func() error {
						return g.AddStartNode(ctx, nName, bee.NodeOptions{})
					})
				}

				if err := errGroup.Wait(); err != nil {
					return nil, fmt.Errorf("starting node group %s: %w", ng, err)
				}
			}
		}
	} else {
		bootnodes := ""
		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode == "bootnode" {
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				cluster.AddNodeGroup(ng, gProfile.Export())

				// add nodes to the node group
				g := cluster.NodeGroup(ng)
				for i := 0; i < len(v.Nodes); i++ {
					nName := v.Nodes[i].Name
					bProfile := c.BeeProfiles[v.BeeConfig]
					bConfig := bProfile.Export()

					bConfig.Bootnodes = fmt.Sprintf(v.Nodes[i].Bootnodes, c.Cluster.Namespace) // TODO: improve bootnode management, support more than 2 bootnodes
					bootnodes += bConfig.Bootnodes + " "
					bOptions := bee.NodeOptions{
						Config:       &bConfig,
						ClefKey:      v.Nodes[i].ClefKey,
						ClefPassword: v.Nodes[i].ClefPassword,
						LibP2PKey:    v.Nodes[i].LibP2PKey,
						SwarmKey:     v.Nodes[i].SwarmKey,
					}

					if err := g.AddNode(nName, bOptions); err != nil {
						return nil, fmt.Errorf("adding node %s: %w", nName, err)
					}
				}
			}
		}

		for ng, v := range c.Cluster.NodeGroups {
			if v.Mode != "bootnode" { // TODO: support standalone nodes
				// add node group to the cluster
				gProfile := c.NodeGroupProfiles[v.Config].NodeGroup
				gOptions := gProfile.Export()
				nProfile := c.BeeProfiles[v.BeeConfig]
				nConfig := nProfile.Export()
				nConfig.Bootnodes = bootnodes
				gOptions.BeeConfig = &nConfig
				cluster.AddNodeGroup(ng, gOptions)

				// add nodes to the node group
				g := cluster.NodeGroup(ng)
				for i := 0; i < v.Count; i++ {
					nName := fmt.Sprintf("%s-%d", ng, i)

					if err := g.AddNode(nName, bee.NodeOptions{}); err != nil {
						return nil, fmt.Errorf("adding node %s: %w", nName, err)
					}
				}
			}
		}
	}

	return
}

func setK8SClient(kubeconfig string, inCluster bool) (c *k8s.Client, err error) {
	if c, err = k8s.NewClient(&k8s.ClientOptions{
		InCluster:      inCluster,
		KubeconfigPath: kubeconfig,
	}); err != nil && err != k8s.ErrKubeconfigNotSet {
		return nil, fmt.Errorf("creating Kubernetes client: %w", err)
	}

	return c, nil
}

var checkStages = []check.Stage{
	[]check.Update{
		{
			NodeGroup: "bee",
			Actions: check.Actions{
				AddCount:    2,
				StartCount:  0,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: check.Actions{
				AddCount:    4,
				StartCount:  0,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
	},
	[]check.Update{
		{
			NodeGroup: "bee",
			Actions: check.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: check.Actions{
				AddCount:    2,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
	[]check.Update{
		{
			NodeGroup: "bee",
			Actions: check.Actions{
				AddCount:    4,
				StartCount:  1,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
		{
			NodeGroup: "drone",
			Actions: check.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
}

var stressStages = []stress.Stage{
	[]stress.Update{
		{
			NodeGroup: "bee",
			Actions: stress.Actions{
				AddCount:    2,
				StartCount:  0,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: stress.Actions{
				AddCount:    4,
				StartCount:  0,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
	},
	[]stress.Update{
		{
			NodeGroup: "bee",
			Actions: stress.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   1,
				DeleteCount: 3,
			},
		},
		{
			NodeGroup: "drone",
			Actions: stress.Actions{
				AddCount:    2,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
	[]stress.Update{
		{
			NodeGroup: "bee",
			Actions: stress.Actions{
				AddCount:    4,
				StartCount:  1,
				StopCount:   3,
				DeleteCount: 1,
			},
		},
		{
			NodeGroup: "drone",
			Actions: stress.Actions{
				AddCount:    3,
				StartCount:  1,
				StopCount:   2,
				DeleteCount: 1,
			},
		},
	},
}

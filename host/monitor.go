package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/flynn/flynn/discoverd/client"
)

type ClusterMetadata struct {
	MonitorEnabled bool `json:"monitor_enabled,omitempty"`
}

type Monitor struct {
	dm        *DiscoverdManager
	discoverd Discoverd
	isLeader  bool
	c         *cluster.Client
}

func NewMonitor(dm *DiscoverdManager) *Monitor {
	return &Monitor{
		dm:        dm,
		discoverd: newDiscoverdWrapper(),
	}
}

func (m *Monitor) Run() error {
	for {
		if m.dm.localConnected() {
			break
		}
		time.Sleep(1 * time.Second)
		fmt.Println("waiting for local discoverd to come up")
	}

	fmt.Println("waiting for raft leader")

	for {
		_, err := discoverd.DefaultClient.RaftLeader()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
	}

	// connect cluster client now that discoverd is up.
	m.c = cluster.NewClient()

	hostSvc := discoverd.NewService("flynn-host")

	fmt.Println("waiting for monitor service to be enabled for this cluster")

	for {
		hostMeta, err := hostSvc.GetMeta()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		var clusterMeta ClusterMetadata
		if err := json.Unmarshal(hostMeta.Data, &clusterMeta); err != nil {
			return fmt.Errorf("error decoding cluster meta")
		}
		if clusterMeta.MonitorEnabled {
			break
		}
		time.Sleep(5 * time.Second)
	}

	for {
		m.isLeader, err = m.discoverd.Register()
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	leaderCh := m.discoverd.LeaderCh()

	for {
		select {
		case isLeader = <-leaderCh:
			m.isLeader = isLeader
			continue
		default:
		}
		if m.isLeader {
			pgSvc := discoverd.NewService("postgres")
			meta, err := pgSvc.GetMeta()
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}
			var state pgstate.State
			if err := json.Unmarshal(meta.Data, &state); err != nil {
				time.Sleep(5 * time.Second)
				continue
			}
			if m.hostAvailable(cluster.ExtractHostID(state.Primary.Meta["FLYNN_JOB_ID"])) {
				// host with primary dataset is up, can continue but should wait for the sync for a little while
				deadline := time.Now() + 60*time.Second
				for {
					if m.hostAvailable(cluster.ExtractHostID(state.Sync.Meta["FLYNN_JOB_ID"])) {
						continue
					} else if time.Now < deadline {
						time.Sleep(5 * time.Second)
					} else {
						continue
					}
				}
			} else {
				time.Sleep(5 * time.Second)
				continue
			}
		}
		fmt.Println("have enough stuff to get pg and controller going")
	}

	// if the postgres state is not populated then there is nothing we
	// can do... we will loop here anyways hoping someone else fixes it.

	// decode the postgres state, work out which hosts we need to wait
	// for and block until they are up.

	// ensure flannel is running on each node, else get it running

	// ensure postgres is in a good state, else fix it

	// ensure controller is running, else get it running
	return nil
}

func (m *Monitor) hostAvailable(id string) bool {
	for _, h := range m.c.Hosts() {
		if h.ID == id {
			return true
		}
	}
	return false
}

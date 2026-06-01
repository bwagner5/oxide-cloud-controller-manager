// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"testing"

	"github.com/oxidecomputer/oxide.go/oxide"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockOxideClient struct {
	InstanceNetworkInterfaceListAllPagesOutput []oxide.InstanceNetworkInterface
	InstanceNetworkInterfaceListAllPagesError  error

	InstanceExternalIpListOutput *oxide.ExternalIpResultsPage
	InstanceExternalIpListError  error

	InstanceViewOutput *oxide.Instance
	InstanceViewError  error
}

var (
	nodeWithoutProviderID = v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec:       v1.NodeSpec{},
	}
	nodeWithProviderID = v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Spec: v1.NodeSpec{
			ProviderID: "oxide://12345678-1234-1234-1234-123456789abc",
		},
	}
	nodeDoesNotExistInOxide = v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
		Spec: v1.NodeSpec{
			ProviderID: "oxide://87654321-1234-1234-1234-123456789abc",
		},
	}

	instanceRunning = oxide.Instance{
		Name:     oxide.Name("node-1"),
		Hostname: "node-1",
		Id:       "12345678-1234-1234-1234-123456789abc",
		RunState: oxide.InstanceStateRunning,
		Ncpus:    2,
		Memory:   1073741824, // 1GiB
	}

	instanceStopped = oxide.Instance{
		Name:     oxide.Name("node-1"),
		Hostname: "node-1",
		Id:       "12345678-1234-1234-1234-123456789abc",
		RunState: oxide.InstanceStateStopped,
		Ncpus:    2,
		Memory:   1073741824, // 1GiB
	}
	ipv4NIC = oxide.InstanceNetworkInterface{
		IpStack: oxide.PrivateIpStack{
			Value: &oxide.PrivateIpStackV4{
				Value: oxide.PrivateIpv4Stack{
					Ip: "192.168.0.2",
				},
			},
		},
	}
	externalEphemeralIPv4 = oxide.ExternalIp{
		Value: &oxide.ExternalIpEphemeral{Ip: "192.168.0.3"},
	}
)

func TestInstanceExists(t *testing.T) {
	t.Run("WithProviderID", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewOutput: &instanceRunning,
			},
			project: "test",
		}
		exists, err := instancesV2.InstanceExists(t.Context(), &nodeWithProviderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatal("expected instance to exist via the Oxide API")
		}
	})

	t.Run("NoProviderID", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewOutput: &instanceRunning,
			},
			project: "test",
		}
		exists, err := instancesV2.InstanceExists(t.Context(), &nodeWithoutProviderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatal("expected instance to exist via the Oxide API")
		}
	})

	t.Run("DoesNotExistInOxide", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewError: oxide.ErrObjectNotFound,
			},
			project: "test",
		}
		exists, err := instancesV2.InstanceExists(t.Context(), &nodeDoesNotExistInOxide)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatal("expected instance to NOT exist via the Oxide API")
		}
	})
}

func TestShutdown(t *testing.T) {
	t.Run("RunningWithProviderID", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewOutput: &instanceRunning,
			},
			project: "test",
		}
		shutdown, err := instancesV2.InstanceShutdown(t.Context(), &nodeWithProviderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shutdown {
			t.Fatal("expected instance to be running via the Oxide API")
		}
	})

	t.Run("RunningNoProviderID", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewOutput: &instanceRunning,
			},
			project: "test",
		}
		shutdown, err := instancesV2.InstanceShutdown(t.Context(), &nodeWithoutProviderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if shutdown {
			t.Fatal("expected instance to be running via the Oxide API")
		}
	})

	t.Run("InstanceStopped", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewOutput: &instanceStopped,
			},
			project: "test",
		}
		shutdown, err := instancesV2.InstanceShutdown(t.Context(), &nodeDoesNotExistInOxide)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !shutdown {
			t.Fatal("expected instance to be stopped via the Oxide API")
		}
	})

	t.Run("DoesNotExistInOxide", func(t *testing.T) {
		instancesV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewError: oxide.ErrObjectNotFound,
			},
			project: "test",
		}
		shutdown, err := instancesV2.InstanceShutdown(t.Context(), &nodeDoesNotExistInOxide)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !shutdown {
			t.Fatal("expected instance to NOT exist via the Oxide API")
		}
	})
}

func TestInstanceMetadata(t *testing.T) {
	t.Run("", func(t *testing.T) {
		instanceV2 := InstancesV2{
			client: &mockOxideClient{
				InstanceViewOutput:                         &instanceRunning,
				InstanceNetworkInterfaceListAllPagesOutput: []oxide.InstanceNetworkInterface{ipv4NIC},
				InstanceExternalIpListOutput: &oxide.ExternalIpResultsPage{
					Items: []oxide.ExternalIp{
						externalEphemeralIPv4,
					},
				},
			},
		}

		metadata, err := instanceV2.InstanceMetadata(t.Context(), &nodeWithoutProviderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedProviderID := "oxide://" + instanceRunning.Id
		if metadata.ProviderID != expectedProviderID {
			t.Fatalf("expected provider ID to equal \"%s\" but got \"%s\"", expectedProviderID, metadata.ProviderID)
		}

		// expects 3 since hostname is included from the instance
		if len(metadata.NodeAddresses) != 3 {
			t.Fatalf("expected node addresses to have a len of 3 but got %v", metadata.NodeAddresses)
		}
	})
}

func (c *mockOxideClient) InstanceNetworkInterfaceListAllPages(
	context.Context,
	oxide.InstanceNetworkInterfaceListParams,
) ([]oxide.InstanceNetworkInterface, error) {
	if c.InstanceNetworkInterfaceListAllPagesError != nil {
		return nil, c.InstanceNetworkInterfaceListAllPagesError
	}
	return c.InstanceNetworkInterfaceListAllPagesOutput, nil
}

func (c *mockOxideClient) InstanceExternalIpList(
	context.Context,
	oxide.InstanceExternalIpListParams,
) (*oxide.ExternalIpResultsPage, error) {
	if c.InstanceExternalIpListError != nil {
		return nil, c.InstanceExternalIpListError
	}
	return c.InstanceExternalIpListOutput, nil
}

func (c *mockOxideClient) InstanceView(
	context.Context,
	oxide.InstanceViewParams,
) (*oxide.Instance, error) {
	if c.InstanceViewError != nil {
		return nil, c.InstanceViewError
	}
	return c.InstanceViewOutput, nil
}

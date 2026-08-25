// Copyright © 2025, SAS Institute Inc., Cary, NC, USA. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package nondefaultplan

import (
	"test/helpers"
	"testing"
)

// TestPlanCiliumNetworkPolicy verifies that setting aks_network_policy=cilium and
// aks_network_dataplane=cilium propagates to the AKS network_profile.
// Both require aks_network_plugin=azure (the default), enforced by module preconditions.
func TestPlanCiliumNetworkPolicy(t *testing.T) {
	t.Parallel()

	variables := helpers.GetDefaultPlanVars(t)
	variables["prefix"] = "cilium-policy"
	variables["aks_network_plugin"] = "azure"
	variables["aks_network_policy"] = "cilium"
	variables["aks_network_dataplane"] = "cilium"

	tests := map[string]helpers.TestCase{
		"networkPlugin": {
			Expected:          "azure",
			ResourceMapName:   "module.aks.azurerm_kubernetes_cluster.aks",
			AttributeJsonPath: "{$.network_profile[0].network_plugin}",
			Message:           "network_plugin must be azure when using cilium",
		},
		"networkPolicy": {
			Expected:          "cilium",
			ResourceMapName:   "module.aks.azurerm_kubernetes_cluster.aks",
			AttributeJsonPath: "{$.network_profile[0].network_policy}",
			Message:           "network_policy must be cilium",
		},
		"networkDataPlane": {
			Expected:          "cilium",
			ResourceMapName:   "module.aks.azurerm_kubernetes_cluster.aks",
			AttributeJsonPath: "{$.network_profile[0].network_data_plane}",
			Message:           "network_data_plane must be cilium",
		},
	}

	plan := helpers.GetPlan(t, variables)
	helpers.RunTests(t, tests, plan)
}

// TestPlanCalicoNetworkPolicy verifies that setting aks_network_policy=calico
// propagates to the AKS network_profile.
func TestPlanCalicoNetworkPolicy(t *testing.T) {
	t.Parallel()

	variables := helpers.GetDefaultPlanVars(t)
	variables["prefix"] = "calico-policy"
	variables["aks_network_plugin"] = "azure"
	variables["aks_network_policy"] = "calico"

	tests := map[string]helpers.TestCase{
		"networkPlugin": {
			Expected:          "azure",
			ResourceMapName:   "module.aks.azurerm_kubernetes_cluster.aks",
			AttributeJsonPath: "{$.network_profile[0].network_plugin}",
			Message:           "network_plugin must be azure",
		},
		"networkPolicy": {
			Expected:          "calico",
			ResourceMapName:   "module.aks.azurerm_kubernetes_cluster.aks",
			AttributeJsonPath: "{$.network_profile[0].network_policy}",
			Message:           "network_policy must be calico",
		},
	}

	plan := helpers.GetPlan(t, variables)
	helpers.RunTests(t, tests, plan)
}

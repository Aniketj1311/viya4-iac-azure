// Copyright © 2025, SAS Institute Inc., Cary, NC, USA. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package defaultplan

import (
	"test/helpers"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test the Outputs section when using the sample-input-defaults.tfvars file.
func TestPlanOutputs(t *testing.T) {
	t.Parallel()

	tests := map[string]helpers.TestCase{
		"outputsLocation": {
			Expected:        "eastus",
			Retriever:       helpers.RetrieveFromRawPlan,
			ResourceMapName: "location",
			Message:         "Location should be set to eastus",
		},
		"outputsClusterApiMode": {
			Expected:        "public",
			Retriever:       helpers.RetrieveFromRawPlan,
			ResourceMapName: "cluster_api_mode",
			Message:         "Cluster API mode should be set to public",
		},
		"outputsJumpRwxFilestorePath": {
			Expected:        "/viya-share",
			Retriever:       helpers.RetrieveFromRawPlan,
			ResourceMapName: "jump_rwx_filestore_path",
			Message:         "Jump VM RWX Filestore Path should be set to /viya-share",
		},
		"outputsPrefix": {
			Expected:        "default",
			Retriever:       helpers.RetrieveFromRawPlan,
			ResourceMapName: "prefix",
			AssertFunction:  assert.Contains,
			Message:         "Prefix should contain default",
		},
	}

	helpers.RunTests(t, tests, helpers.GetDefaultPlan(t))
}

// TestPlanOutputSensitivityFlags verifies that all outputs declared with sensitive = true
// have AfterSensitive set to true in the plan, ensuring secrets are never exposed in plan output.
func TestPlanOutputSensitivityFlags(t *testing.T) {
	t.Parallel()

	sensitiveOutputs := []string{
		"aks_host",
		"kube_config",
		"aks_cluster_node_username",
		"aks_cluster_password",
		"postgres_servers",
		"cr_admin_password",
	}

	plan := helpers.GetDefaultPlan(t)

	for _, outputName := range sensitiveOutputs {
		outputName := outputName
		t.Run(outputName, func(t *testing.T) {
			t.Parallel()
			tc := helpers.TestCase{
				Expected:        "true",
				Retriever:       helpers.RetrieveAfterSensitiveFromOutputChanges,
				ResourceMapName: outputName,
				Message:         outputName + " output must have AfterSensitive=true",
			}
			helpers.RunTest(t, tc, plan)
		})
	}
}

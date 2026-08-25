# Terratest Documentation — viya4-iac-azure

---

## 1. Overview

This page documents the Terratest-based automated tests for the **viya4-iac-azure** Terraform repository, which provisions Azure infrastructure (AKS, networking, storage, VMs, and related services) for SAS Viya 4 deployments.

### Why Terraform plan and apply tests?

| Test type | What it validates |
|---|---|
| **Plan tests** | The Terraform execution plan produced from a given set of input variables. They assert that specific resource attributes, counts, and conditional branches have the expected values *before* any real infrastructure is created. Plan tests are fast, require no Azure credentials at the resource level, and serve as the first line of defense against configuration regressions. |
| **Apply tests** | Actual Azure resources created by `terraform apply`. They call Azure SDK APIs to verify that the deployed state (provisioning status, names, sizes, attachment states, network configuration) matches both the plan and well-known expected values. Apply tests require a live Azure subscription and take significantly longer to run. |

### What the test suite is intended to catch

- Accidental changes to default variable values that alter default resource configuration.
- Conditional logic regressions (e.g., resources that should be absent when a feature flag is `false` appearing in the plan, or vice versa).
- Attribute mismatches between what Terraform intends to create and what it actually creates in Azure.
- Configuration drift caused by provider upgrades or module changes.

### What the test suite does not guarantee

- Correctness of newly added Terraform resources or modules for which no test has yet been written. Tests are authored manually; adding a new resource to the Terraform code does not automatically produce a corresponding test.
- Application-level behaviour after deployment (Kubernetes workload scheduling, SAS Viya startup, etc.).
- Security posture beyond the configuration attributes that are explicitly asserted.

### Test categories covered

1. **Default Plan** — Runs `terraform plan` using the `sample-input-defaults.tfvars` file and asserts expected values for all major resources under that default configuration.
2. **Non-Default Plan** — Runs `terraform plan` with targeted variable overrides to exercise optional features, conditional resource creation/removal, and alternative configurations.
3. **Default Apply** — Runs `terraform apply` using the default configuration and queries the Azure APIs to validate that deployed resources match the plan and have the expected runtime state.

---

## 2. Default Plan Tests

Default Plan tests run `terraform plan` against `sample-input-defaults.tfvars` (no variable overrides). They confirm that the baseline configuration produces a plan whose resource attributes match the documented defaults in `CONFIG-VARS.md`.

All tests in this section are in `test/defaultplan/`.

---

### 2.1 Admin Access

**File:** `admin_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanAdminAccess` | Verifies that public-access CIDR variables are correctly reflected in the Terraform plan. | With default inputs, `default_public_access_cidrs` is expected to equal `{[123.45.67.89/16]}` (the value from the defaults tfvars). `cluster_endpoint_public_access_cidrs`, `vm_public_access_cidrs`, `postgres_public_access_cidrs`, and `acr_public_access_cidrs` are all expected to be `{<nil>}`, meaning no additional CIDR restrictions are applied by default. Asserts these raw-plan output variables match their expected values. |

---

### 2.2 AKS Cluster Defaults

**File:** `defaults_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanDefaults` | Asserts the default values on the AKS cluster resource (`module.aks.azurerm_kubernetes_cluster.aks`). | Validates: `linux_profile[0].admin_username` = `azureuser`; `network_profile[0].outbound_type` = `loadBalancer`; `network_profile[0].network_plugin` = `azure`; `kubernetes_version` = `1.35`; `sku_tier` = `Free`; `support_plan` = `KubernetesOfficial`; user-assigned identity (`azurerm_user_assigned_identity.uai[0]`) is not nil; `azure_active_directory_role_based_access_control` is `[]` (RBAC disabled by default); SSH key data is not nil; `run_command_enabled` = `false`; `azure_policy_enabled` = `false`; `node_os_upgrade_channel` = `NodeImage`. |
| `TestPlanGeneral` | Asserts that the kubeconfig, Jump VM, and Jump VM public IP resources are all present with correct defaults. | Validates: `module.kubeconfig.kubernetes_cluster_role_binding.kubernetes_crb[0]` is not nil; `module.kubeconfig.kubernetes_service_account.kubernetes_sa[0]` is not nil; `module.jump[0].azurerm_linux_virtual_machine.vm` is not nil; `module.jump[0].azurerm_public_ip.vm_ip[0]` is not nil; Jump VM public IP `allocation_method` = `Static`; Jump VM `admin_username` = `jumpuser`; Jump VM `size` = `Standard_B2ls_v2`. |
| `TestPlanAcrDisabled` | Confirms that the Azure Container Registry resource is absent from the plan by default. | With `create_container_registry` defaulting to `false`, `azurerm_container_registry.acr[0]` must be `nil` in the plan. |

---

### 2.3 Location

**File:** `location_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanLocation` | Asserts that every major resource has its `location` attribute set to `eastus` (the default location). | Checks the `location` attribute on: `azurerm_network_security_group.nsg[0]`, `azurerm_resource_group.aks_rg[0]`, `azurerm_user_assigned_identity.uai[0]`, `module.aks.azurerm_kubernetes_cluster.aks`, `module.jump[0].azurerm_linux_virtual_machine.vm`, `module.jump[0].azurerm_network_interface.vm_nic`, `module.jump[0].azurerm_public_ip.vm_ip[0]`, NFS managed disks 0–3 (`module.nfs[0].azurerm_managed_disk.vm_data_disk[0..3]`), `module.nfs[0].azurerm_network_interface.vm_nic`, and `module.vnet.azurerm_virtual_network.vnet[0]`. All must equal `eastus`. |

---

### 2.4 Network

**File:** `network_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanNetwork` | Validates default network configuration on the VNet and AKS cluster. | Asserts: VNet `address_space` = `["192.168.0.0/16"]`; VNet has no inline subnet (`subnet[0].name` is empty string); AKS `network_profile[0].outbound_type` = `loadBalancer`; AKS `network_profile[0].network_plugin` = `azure`; `aks_network_policy` expression reference is empty; `aks_network_plugin_mode` expression reference is empty; `aks_pod_cidr` plan output = `10.244.0.0/16`. |

---

### 2.5 NFS Storage

**File:** `nfs_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanNFSPublicIP` | Confirms the NFS VM public IP resource is not created by default. | With `create_nfs_public_ip` defaulting to `false`, `module.nfs[0].azurerm_public_ip.vm_ip[0]` must be `nil` in the plan. |
| `TestPlanNFSDisk` | Validates that all four NFS data disks are present with correct default configuration. | For each of `module.nfs[0].azurerm_managed_disk.vm_data_disk[0..3]`: the resource is not nil, `storage_account_type` = `Standard_LRS`, and `disk_size_gb` = `256`. |

---

### 2.6 Node Pools

**File:** `node_pools_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanNodePools` | Validates default node pool (system pool) configuration on the AKS cluster. | Asserts on `module.aks.azurerm_kubernetes_cluster.aks`: `linux_profile[0].admin_username` = `azureuser`; `default_node_pool[0].vm_size` = `Standard_E8s_v5`; `default_node_pool[0].os_disk_size_gb` = `128`; `default_node_pool[0].max_pods` = `110`; `default_node_pool[0].min_count` = `1`; `default_node_pool[0].max_count` = `5`; `default_node_pool[0].zones` = `["1"]`. |
| `TestPlanAdditionalNodePools` | Validates the default configuration of the four additional node pools: `stateless`, `stateful`, `cas`, and `compute`. | For each pool (`module.node_pools["<name>"].azurerm_kubernetes_cluster_node_pool.autoscale_node_pool[0]`) asserts VM size, OS disk size (200 GB for all), min/max node counts, max pods (110 for all), node taints, node labels, availability zones (`["1"]`), and `fips_enabled=false`. Expected VM sizes: `stateless` = `Standard_D4s_v5`, `stateful` = `Standard_D4s_v5`, `cas` = `Standard_E16ds_v5`, `compute` = `Standard_D4ds_v5`. Expected taints follow `workload.sas.com/class=<name>:NoSchedule`. The `compute` pool additionally carries the `launcher.sas.com/prepullImage` label and has `min_count=1`. |

---

### 2.7 Outputs

**File:** `output_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanOutputs` | Validates the Terraform output variables produced by the default plan. | Asserts: `location` output = `eastus`; `cluster_api_mode` output = `public`; `jump_rwx_filestore_path` output = `/viya-share`; `prefix` output contains the string `default`. |

---

### 2.8 NFS VM Storage

**File:** `storage_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanStorage` | Validates the NFS VM resource configuration. | Asserts on `module.nfs[0].azurerm_linux_virtual_machine.vm`: `admin_username` = `nfsuser`; `size` = `Standard_D4s_v5`; resource is not nil; `vm_zone` is empty (no availability zone assigned by default). |

---

### 2.9 Subnets

**File:** `subnets_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanSubnets` | Validates the default `aks` and `misc` subnet configurations. | For each of `module.vnet.azurerm_subnet.subnet["aks"]` and `module.vnet.azurerm_subnet.subnet["misc"]` asserts: `address_prefixes` (aks = `["192.168.0.0/23"]`, misc = `["192.168.2.0/24"]`); `service_endpoints` = `["Microsoft.Sql"]`; `private_endpoint_network_policies` = `Enabled`; `private_link_service_network_policies_enabled` = `false`; `service_delegations` = empty. |

---

## 3. Non-Default Plan Tests

Non-Default Plan tests intentionally override one or more Terraform input variables to exercise optional features, conditional resource creation or removal paths, alternative configurations, and edge cases. Each test calls `helpers.GetPlan(t, variables)` (or `helpers.GetPlanFromCache`) with a customised variables map derived from `helpers.GetDefaultPlanVars(t)`.

All tests in this section are in `test/nondefaultplan/`.

---

### 3.1 Azure Container Registry (ACR)

**File:** `acr_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanACRStandard` | Validates ACR creation with the Standard SKU and admin access enabled. | Sets `create_container_registry=true`, `container_registry_admin_enabled=true`, `container_registry_sku="Standard"`. Asserts on `azurerm_container_registry.acr[0]`: `georeplications` = `[]` (no geo-replications for Standard SKU); `name` contains `"acr"`; `sku` = `Standard`; `admin_enabled` = `true`. |
| `TestPlanACRPremium` | Validates ACR creation with the Premium SKU, admin access enabled, and two geo-replica locations. | Sets `create_container_registry=true`, `container_registry_admin_enabled=true`, `container_registry_sku="Premium"`, `container_registry_geo_replica_locs=["southeastus5","southeastus3"]`. Asserts: geo-replication locations contain `southeastus3` and `southeastus5`; name contains `"acr"`; `sku` = `Premium`; `admin_enabled` = `true`. |
| `TestPlanACRAdminDisabled` | Confirms that admin access can be disabled on the ACR. | Sets `create_container_registry=true`, `container_registry_admin_enabled=false`, `container_registry_sku="Standard"`. Asserts `admin_enabled` = `false`. |

---

### 3.2 AKS Cluster Configuration

**File:** `aks_config_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanAksSkuStandard` | Validates AKS Standard tier SKU. | Sets `aks_cluster_sku_tier="Standard"`. Asserts `sku_tier` = `Standard` and `support_plan` = `KubernetesOfficial` on `module.aks.azurerm_kubernetes_cluster.aks`. |
| `TestPlanAksPremiumLts` | Validates AKS Premium tier SKU paired with the required Long-Term Support plan. | Sets `aks_cluster_sku_tier="Premium"` and `cluster_support_tier="AKSLongTermSupport"`. Asserts `sku_tier` = `Premium` and `support_plan` = `AKSLongTermSupport`. |
| `TestPlanFipsEnabled` | Validates that `fips_enabled=true` propagates to all applicable node pools. | Sets `fips_enabled=true`. Asserts `fips_enabled` = `true` on the AKS default node pool (`default_node_pool[0]`) and on the `stateless`, `stateful`, and `cas` additional node pools. |
| `TestPlanHostEncryption` | Validates that host-level disk encryption propagates to all applicable node pools. | Sets `aks_cluster_enable_host_encryption=true`. Asserts `host_encryption_enabled` = `true` on the AKS default node pool and on the `stateless` and `stateful` additional node pools. |
| `TestPlanUserDefinedRouting` | Validates the User Defined Routing egress type. | Sets `cluster_egress_type="userDefinedRouting"`. Asserts `network_profile[0].outbound_type` = `userDefinedRouting`. |

---

### 3.3 Azure Policy and Custom Subnets

**File:** `azure_policy_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanAzurePolicy` | Validates that enabling Azure Policy sets the correct flag on the AKS cluster. | Sets `aks_azure_policy_enabled=true` and `aks_network_plugin="azure"`. Asserts `azure_policy_enabled` = `true`, `network_profile[0].network_plugin` = `azure`, and `aks_pod_cidr` output = `10.244.0.0/16`. |
| `TestPlanCustomSubnets` | Validates that fully customised subnet definitions (including a NetApp subnet with delegation) are accepted and result in the correct network plugin and pod CIDR. | Sets `aks_network_plugin="azure"` and replaces the `subnets` variable with a map containing custom CIDRs and delegations for the `aks`, `misc`, and `netapp` subnets (including a `Microsoft.Netapp/volumes` delegation). Asserts `network_plugin` = `azure` and `aks_pod_cidr` = `10.244.0.0/16`. |

---

### 3.4 Jump VM

**File:** `jump_vm_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanJumpVmDisabled` | Confirms that all Jump VM resources are absent when the VM is disabled. | Sets `create_jump_vm=false`. Asserts `module.jump[0].azurerm_linux_virtual_machine.vm`, `module.jump[0].azurerm_network_interface.vm_nic`, and `module.jump[0].azurerm_public_ip.vm_ip[0]` are all `nil` in the plan. |
| `TestPlanJumpPublicIpDynamic` | Validates dynamic IP allocation for the Jump VM public IP. | Sets `enable_jump_public_static_ip=false`. Asserts `module.jump[0].azurerm_public_ip.vm_ip[0]` has `allocation_method` = `Dynamic`. |

---

### 3.5 Static Kubeconfig

**File:** `kubeconfig_disabled_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanStaticKubeconfigDisabled` | Confirms that kubeconfig Kubernetes resources are removed when static kubeconfig creation is disabled. | Sets `create_static_kubeconfig=false`. Asserts `module.kubeconfig.kubernetes_cluster_role_binding.kubernetes_crb[0]` and `module.kubeconfig.kubernetes_service_account.kubernetes_sa[0]` are both `nil`. |

---

### 3.6 Azure Monitor

**File:** `monitor_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanAzureMonitor` | Validates that all Azure Monitor resources and the AKS OMS agent block are created when monitoring is enabled. | Sets `create_aks_azure_monitor=true`. Asserts: `azurerm_log_analytics_workspace.viya4[0]` is not nil and has `sku` = `PerGB2018`; `azurerm_log_analytics_solution.viya4[0]` is not nil; `azurerm_monitor_diagnostic_setting.audit[0]` is not nil; the `oms_agent` block on `module.aks.azurerm_kubernetes_cluster.aks` is not equal to `[]` (i.e., is present). |

---

### 3.7 Azure NetApp Files

**File:** `netapp_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanNetApp` | Validates the default Azure NetApp Files resource configuration when `storage_type=ha`. | Sets `storage_type="ha"`. Asserts: NetApp account (`module.netapp[0].azurerm_netapp_account.anf`) and pool (`module.netapp[0].azurerm_netapp_pool.anf`) are not nil; pool `service_level` = `Premium`; pool `size_in_tb` = `4`; volume (`module.netapp[0].azurerm_netapp_volume.anf`) is not nil with `protocols` = `["NFSv4.1"]`, `service_level` = `Premium`, `volume_path` contains `"export"`, `network_features` = `Basic`, `zone` = `1`; the `netapp` subnet is created. |
| `TestPlanNetAppCrossZoneReplication` | Validates that enabling cross-zone replication creates replica resources and DNS infrastructure. | Sets `storage_type="ha"`, `netapp_enable_cross_zone_replication=true`, `netapp_network_features="Standard"`, `netapp_availability_zone="1"`, `netapp_replication_zone="2"`, `netapp_size_in_tb=1`. Asserts: primary volume `network_features` = `Standard`; `module.netapp[0].azurerm_netapp_pool.anf_replica[0]` is not nil; `module.netapp[0].azurerm_netapp_volume.anf_replica[0]` is not nil and has `zone` = `2`; `module.netapp[0].azurerm_private_dns_zone.anf_dns[0]` is not nil; `module.netapp[0].azurerm_private_dns_a_record.anf_primary[0]` is not nil. |

---

### 3.8 Azure CNI Overlay

**File:** `overlay_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanAzureCniOverlay` | Validates Azure CNI Overlay mode configuration on the AKS network profile. | Sets `aks_network_plugin="azure"` and `aks_network_plugin_mode="overlay"`. Asserts on `module.aks.azurerm_kubernetes_cluster.aks`: `network_profile[0].network_plugin` = `azure`; `network_profile[0].network_plugin_mode` = `overlay`. |

---

### 3.9 PostgreSQL Flexible Server

**File:** `postgres_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanPostgresServers` | Validates the default PostgreSQL Flexible Server configuration when a server entry is provided. | Sets `postgres_servers={"default":{}}`. Asserts on `module.flex_postgresql["default"].azurerm_postgresql_flexible_server.flexpsql`: resource is not nil; `sku_name` = `GP_Standard_D4s_v3`; `storage_mb` = `131072`; `backup_retention_days` = `7`; `geo_redundant_backup_enabled` = `false`; `administrator_login` = `pgadmin`; `administrator_password` = `my$up3rS3cretPassw0rd`; `version` = `16`; `require_secure_transport` configuration is not `OFF`; `virtual_network_id` is empty. Also asserts the `max_prepared_transactions` server configuration resource has `name` = `max_prepared_transactions` and `value` = `1024`. |
| `TestPlanPostgresHA` | Validates Zone-Redundant HA configuration for PostgreSQL. | Sets `high_availability_mode="ZoneRedundant"`, `availability_zone="1"`, `standby_availability_zone="2"` within the `default` server map. Asserts `high_availability[0].mode` = `ZoneRedundant`, `high_availability[0].standby_availability_zone` = `2`, and `zone` = `1`. |
| `TestPlanPostgresPrivate` | Validates private connectivity mode for PostgreSQL: DNS zone, VNet link, and public access disabled. | Sets `connectivity_method="private"` for the `default` server. Also injects a `postgresql` subnet with the `Microsoft.DBforPostgreSQL/flexibleServers` delegation. Asserts: `module.flex_postgresql["default"].azurerm_private_dns_zone.flexpsql[0]` is not nil; `module.flex_postgresql["default"].azurerm_private_dns_zone_virtual_network_link.flexpsql[0]` is not nil; `public_network_access_enabled` = `false`. |

---

### 3.10 Private AKS Cluster

**File:** `private_cluster_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanPrivateCluster` | Validates that setting `cluster_api_mode=private` enables private cluster mode with the correct settings. | Sets `cluster_api_mode="private"`. Asserts on `module.aks.azurerm_kubernetes_cluster.aks`: `private_cluster_enabled` = `true`; `private_dns_zone_id` = `System` (no custom DNS zone provided); `api_server_access_profile` = `[]` (absent, because endpoint access CIDRs are forced to empty for private clusters). |

---

### 3.11 RBAC / Azure Active Directory

**File:** `rbac_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanRbacEnabledGroupIds` | Validates RBAC configuration with a tenant ID and explicit admin group object IDs. | Sets `rbac_aad_enabled=true`, `rbac_aad_tenant_id="2492e7f7-..."`, `rbac_aad_admin_group_object_ids=["59218b02-...","498afef2-..."]`. Asserts the `azure_active_directory_role_based_access_control` block is not nil and that `tenant_id` and `admin_group_object_ids` match the supplied values. |
| `TestPlanRbacEnabledWithTenant` | Validates RBAC enabled using the global `tenant_id` variable (no explicit admin group IDs). | Sets `rbac_aad_enabled=true` and `tenant_id="b1c14d5c-..."`. Asserts: RBAC block is not nil; `azure_rbac_enabled` = `false`; `tenant_id` matches; `admin_group_object_ids` = `null`. |
| `TestPlanAzureRbacEnabledWithTenant` | Validates RBAC with Azure RBAC (role-based) authorization explicitly enabled. | Sets `rbac_aad_enabled=true`, `rbac_aad_azure_rbac_enabled=true`, and `tenant_id="b1c14d5c-..."`. Asserts: RBAC block is not nil; `azure_rbac_enabled` = `true`; `tenant_id` matches. |

---

### 3.12 Storage Type None

**File:** `storage_none_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanStorageNone` | Validates that neither the NFS VM nor the NetApp account is created when storage is disabled. | Sets `storage_type="none"`. Asserts `module.nfs[0].azurerm_linux_virtual_machine.vm` = `nil` and `module.netapp[0].azurerm_netapp_account.anf` = `nil`. |

---

### 3.13 Tag Propagation

**File:** `tags_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanTagsPropagation` | Validates that a custom `tags` map is propagated to key resources. | Sets `tags={"env":"test","owner":"testuser"}`. Asserts: `module.aks.azurerm_kubernetes_cluster.aks` has `tags.env` = `test` and `tags.owner` = `testuser`; `module.vnet.azurerm_virtual_network.vnet[0]` has `tags.env` = `test`. |

---

### 3.14 Workload Identity

**File:** `workload_identity_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestPlanWorkloadIdentity` | Validates that enabling workload identity sets both OIDC issuer and workload identity flags on the AKS cluster. | Sets `enable_workload_identity=true`. Asserts `oidc_issuer_enabled` = `true` and `workload_identity_enabled` = `true` on `module.aks.azurerm_kubernetes_cluster.aks`. |

---

## 4. Default Apply Tests

Default Apply tests run a full `terraform apply` using the default configuration and then query the Azure Resource Manager APIs (via the Terratest Azure SDK helpers) to validate that deployed resources match the plan and have the expected runtime state. All tests are orchestrated by `TestApplyDefaultMain` in `test/defaultapply/default_apply_main_test.go`. Cleanup (`terraform destroy`) is deferred and runs automatically after all assertions complete.

> **Note:** A `TestApplyNonDefaultMain` entry point exists in `test/nondefaultapply/` but currently contains only placeholder variable overrides with no assertion logic. It is not documented as a test suite here.

All functions in this section are called from `TestApplyDefaultMain`.

---

### 4.1 Entry Point

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `TestApplyDefaultMain` | Orchestrates the entire default apply test run. | Runs `terraform init` and `terraform apply` with no variable overrides. Defers `terraform destroy`. Calls each component sub-test in sequence: `testApplyResourceGroup`, `testApplyVirtualMachine`, `testApplyAKSCluster`, `testApplyNFSDisks`, `testApplyNetwork`, `testApplyJumpPublicIP`, `testApplyNodePools`. |

---

### 4.2 Resource Group

**File:** `resource_group_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyResourceGroup` | Validates the deployed Azure resource group against the plan. | Calls `azure.GetAResourceGroupE` using the resource group name from the plan (`azurerm_resource_group.aks_rg[0]`). Asserts: resource group object is not nil (it exists); `Location` matches the plan; `Name` matches the plan; `ID` is not nil. All comparisons are against the Terraform plan. |

---

### 4.3 Virtual Machines

**File:** `vm_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyVirtualMachine` | Top-level function that validates the VM list and each individual VM. | Calls `testVMList` then `testVM` for both the `nfs` and `jump` VMs. |
| `testVMList` | Validates that exactly two VMs exist in the resource group and that their names match the plan. | Calls `azure.ListVirtualMachinesForResourceGroupE`. Asserts list length = 2, list contains the NFS VM name from the plan, and list contains the Jump VM name from the plan. |
| `testVM` (nfs) | Validates the deployed NFS VM properties against the plan. | Calls `azure.GetVirtualMachineE` for the NFS VM. Asserts: VM exists; `AdminUsername` matches plan; `ComputerName` is not nil; `ID` is not nil; `Location` matches plan; `NetworkInterfaces` is not nil; `PlatformFaultDomain` is nil; `Priority` matches plan; `VMSize` matches plan; `UltraSSDEnabled` matches plan; OS disk size matches plan; OS disk `ID` and `Name` are not nil; OS disk `StorageAccountType` matches plan; `WriteAcceleratorEnabled` matches plan; image `Offer`, `Publisher`, `Sku`, and `Version` all match plan. |
| `testVM` (jump) | Validates the deployed Jump VM properties against the plan. | Same assertions as for the NFS VM above, applied to the Jump VM (`module.jump[0].azurerm_linux_virtual_machine.vm`). |

---

### 4.4 AKS Cluster

**File:** `aks_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyAKSCluster` | Validates the deployed AKS cluster. | Calls `azure.GetManagedClusterE`. Asserts: cluster `ID` is not nil; `ProvisioningState` = `Succeeded`; `Name` matches plan; `Location` matches plan; `NodeResourceGroup` matches plan; `KubernetesVersion` contains the version prefix from the plan (Azure may normalise `"1.35"` to `"1.35.x"`). Calls `testAKSDefaultNodePool`. |
| `testAKSDefaultNodePool` | Validates the system (default) node pool configuration and running node count. | Locates the agent pool profile named `"system"`. Asserts: default pool `VMSize` matches plan; `MaxPods` matches plan; `OsDiskSizeGB` matches plan; actual running node `Count` is within the `[min_count, max_count]` range from the plan. |

---

### 4.5 NFS Managed Disks

**File:** `nfs_disks_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyNFSDisks` | Validates all four NFS data disks against the plan. | Iterates over disk indices 0–3. For each disk, calls `azure.GetDiskE` using the name from the plan (`module.nfs[0].azurerm_managed_disk.vm_data_disk[i]`). Asserts: disk `ID` is not nil; `DiskSizeGB` matches the plan value; `DiskState` = `Attached` (confirming the disk is attached to the NFS VM). |

---

### 4.6 Virtual Network

**File:** `network_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyNetwork` | Validates the deployed VNet against the plan. | Calls `azure.GetVirtualNetworkE`. Asserts: VNet `ID` is not nil; `Name` matches plan; first address prefix in `AddressSpace.AddressPrefixes` matches the first element of `address_space` from the plan. |

---

### 4.7 Jump VM Public IP and NFS IP Absence

**File:** `jump_ip_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyJumpPublicIP` | Validates the Jump VM public IP and confirms the NFS public IP was not created. | Calls `azure.GetPublicIPAddressE` for the Jump VM IP. Asserts: IP `ID` is not nil; `ProvisioningState` = `Succeeded`; `Name` matches plan; `PublicIPAllocationMethod` matches plan. Also asserts (via `jumpIPAddressAssignedTest`) that the IP address string is non-nil and non-empty. Separately, calls `azure.PublicAddressExistsE` for the NFS IP name and asserts it does NOT exist, confirming `create_nfs_public_ip=false` was honoured. |

---

### 4.8 Additional Node Pools

**File:** `node_pools_test.go`

| Test Function | Description | Scenario / What is Being Validated |
|---|---|---|
| `testApplyNodePools` | Validates all four additional node pools on the deployed AKS cluster. | Calls `azure.GetManagedClusterE` then iterates over pool names `["stateless", "stateful", "cas", "compute"]`. For each pool, calls `testAdditionalNodePool`. |
| `testAdditionalNodePool` | Validates a single additional node pool against the plan. | Locates the agent pool profile by name. Asserts: `ProvisioningState` = `Succeeded`; `VMSize` matches the plan value from `module.node_pools["<name>"].azurerm_kubernetes_cluster_node_pool.autoscale_node_pool[0]`. |

---

## 5. How to Add Terratest to a Repository

This section provides a practical guide for introducing Terraform plan and apply testing into a new Terraform repository using the [Terratest](https://terratest.gruntwork.io/) framework.

---

### 5.1 Prerequisites

- **Go** ≥ 1.21 (the module in this repo uses 1.23).
- **Terraform** CLI available on the `PATH`.
- **Cloud provider credentials** accessible at runtime (environment variables, service principal, workload identity, etc.).
- A working Terraform root module with a `terraform.tfvars` or equivalent defaults file to use as a test baseline.
- Familiarity with the `go test` command.

Initialize a Go module in your test directory:

```bash
mkdir test && cd test
go mod init <module-name>
go get github.com/gruntwork-io/terratest@v0.48.2
go get github.com/stretchr/testify@v1.10.0
```

Add cloud provider SDKs as needed (e.g., `github.com/Azure/azure-sdk-for-go` for Azure).

---

### 5.2 Identify Resources and Expected Behaviour

Before writing any code:

1. List the Terraform resources you want to validate (e.g., resource groups, clusters, VMs, networking).
2. For each resource, identify the attributes that matter: names, sizes, counts, flags, locations, connection properties.
3. Separate attributes that have hard-coded expected values (e.g., a default admin username) from attributes that should simply match what the plan says (e.g., a computed name).
4. Identify conditional paths: resources that should be absent when a feature flag is `false`, or resources that should only appear when a feature is enabled.

---

### 5.3 Creating Plan Tests

Plan tests use `terratest/modules/terraform.InitAndPlanAndShowWithStructNoLogTempPlanFileE` (or equivalent) to generate and parse a plan without applying it.

**General pattern:**

```go
func TestMyPlanTest(t *testing.T) {
    t.Parallel()

    // 1. Set up Terraform options pointing at your module root
    terraformOptions := &terraform.Options{
        TerraformDir: "../",                       // path to your root module
        VarFiles:     []string{"defaults.tfvars"}, // your baseline variable file
        Vars: map[string]interface{}{
            "some_variable": "override_value",     // per-test overrides
        },
    }

    // 2. Generate the plan
    _, err := terraform.InitAndPlanAndShowWithStructNoLogTempPlanFileE(t, terraformOptions)
    require.NoError(t, err)

    // 3. Retrieve planned values using JSON path or struct navigation
    actualValue := plan.ResourcePlannedValuesMap["azurerm_resource_group.rg"]["location"]

    // 4. Assert expected behaviour
    assert.Equal(t, "eastus", actualValue, "Resource group location should be eastus")
}
```

Key points:

- Use a **shared cached plan** for tests that all use the same variable set to avoid running `terraform plan` once per test function.
- Use **`t.Parallel()`** to run independent tests concurrently.
- For resources expected to be absent, assert the resource map entry is `nil`.
- For conditional attributes, prefer exact string comparisons over type-aware comparisons, since plan values are serialised as strings in the JSON output.

---

### 5.4 Creating Apply Tests

Apply tests run `terraform init` + `terraform apply`, then call cloud provider APIs to inspect actual resources.

**General pattern:**

```go
func TestMyApplyTest(t *testing.T) {
    terraformOptions := &terraform.Options{
        TerraformDir: "../",
        VarFiles:     []string{"defaults.tfvars"},
    }

    // 1. Apply (and capture the plan for cross-referencing)
    terraform.InitAndApply(t, terraformOptions)

    // 2. Defer destroy to ensure cleanup even on test failure
    defer terraform.Destroy(t, terraformOptions)

    // 3. Read outputs or known resource names from Terraform state
    resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")

    // 4. Query the actual cloud resource
    rg, err := azure.GetAResourceGroupE(resourceGroupName, os.Getenv("AZURE_SUBSCRIPTION_ID"))
    require.NoError(t, err)

    // 5. Assert actual state
    assert.Equal(t, "Succeeded", *rg.Properties.ProvisioningState)
    assert.Equal(t, "eastus", *rg.Location)
}
```

Key points:

- Always defer `Destroy` **before** making assertions so cleanup happens even when assertions fail.
- Cross-reference live values against the **plan** (not hard-coded strings) wherever the value is computed. Hard-code only values that should never change regardless of inputs (e.g., provisioning states, SKU tiers for default configurations).
- Apply tests take significantly longer than plan tests. Run them in a separate CI stage or on demand.
- Store sensitive credentials in environment variables; never commit them to the repository.

---

### 5.5 Testing Non-Default / Conditional Configurations

For each conditional feature or optional resource:

1. Start with the default variable set (e.g., load from your baseline tfvars file).
2. Override only the variables relevant to the feature under test.
3. Assert both the positive case (resource exists / attribute has new value) and, in a separate test, the negative case (resource is absent when flag is off).

```go
// Positive: feature enabled
variables := getDefaultVars(t)
variables["enable_feature_x"] = true
plan := generatePlan(t, variables)
assert.NotEqual(t, "nil", plan.ResourcePlannedValuesMap["azurerm_feature_x_resource"])

// Negative: feature disabled (often covered by the default plan tests)
variables2 := getDefaultVars(t)
variables2["enable_feature_x"] = false
plan2 := generatePlan(t, variables2)
assert.Equal(t, nil, plan2.ResourcePlannedValuesMap["azurerm_feature_x_resource"])
```

Use a unique `prefix` or name override in each test to prevent plan-cache collisions when multiple non-default tests are run in parallel against a shared cache.

---

### 5.6 Running the Tests

**Plan tests (no cloud credentials needed beyond Terraform init):**

```bash
cd test
# Run all default plan tests
go test ./defaultplan/... -v -timeout 30m

# Run all non-default plan tests
go test ./nondefaultplan/... -v -timeout 30m
```

**Apply tests (require cloud credentials):**

```bash
export TF_VAR_subscription_id="<your-subscription-id>"
# Set other required auth environment variables

cd test
go test ./defaultapply/... -v -timeout 90m -run TestApplyDefaultMain
```

**Running a specific test:**

```bash
go test ./nondefaultplan/... -v -run TestPlanNetApp
```

**In CI:**

- Run plan tests on every pull request (fast, no infrastructure cost).
- Run apply tests on a schedule or on merge to main (slower, incurs cloud costs, requires credentials in CI secrets).
- Use `go test -parallel <N>` to control concurrency within a package. Individual test functions already call `t.Parallel()` where safe.

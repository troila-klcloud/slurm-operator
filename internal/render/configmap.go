package render

import (
	"fmt"

	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/config"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RenderConfigMapSlurmConfigs renders new [corev1.ConfigMap] containing '.conf' files for the following components:
//
// [consts.ConfigMapKeySlurmConfig] - Slurm config
// [consts.ConfigMapKeyCGroupConfig] - cgroup config
// [consts.ConfigMapKeySpankConfig] - SPANK plugins config
// [consts.ConfigMapKeyGresConfig] - gres config
func RenderConfigMapSlurmConfigs(cluster *slurmv1alpha1.Cluster) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildConfigMapSlurmConfigsName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    RenderLabels(consts.ComponentTypeController, cluster.Name),
		},
		Data: map[string]string{
			consts.ConfigMapKeySlurmConfig:  generateSlurmConfig(cluster).Render(),
			consts.ConfigMapKeyGresConfig:   generateGresConfig().Render(),
			consts.ConfigMapKeySSSDConfig:   generateSSSdConfig().Render(),
			consts.ConfigMapKeyCGroupConfig: generateCgroupConfig().Render(),
		},
	}
}

func generateGresConfig() utils.ConfigFile {
	res := &utils.PropertiesConfig{}
	res.AddLine("Name=gpu Type=nvidia File=/dev/nvidia0")
	res.AddLine("Name=shard Count=8 File=/dev/nvidia0")
	res.AddLine("\n")
	return res
}

func generateSlurmConfig(cluster *slurmv1alpha1.Cluster) utils.ConfigFile {
	res := &utils.PropertiesConfig{}

	res.AddProperty("ClusterName", cluster.Name)
	res.AddComment("")
	// example: SlurmctldHost=controller-0(controller-0.controller.slurm-poc.svc.cluster.local)
	for i := int32(0); i < cluster.Spec.CtrlNode.Size; i++ {
		hostName, hostFQDN := utils.BuildServiceHostFQDN(
			consts.ComponentTypeController,
			cluster.Namespace,
			cluster.Name,
			i,
		)
		res.AddProperty("SlurmctldHost", fmt.Sprintf("%s(%s)", hostName, hostFQDN))
	}
	res.AddComment("")
	res.AddProperty("SlurmctldPidFile", "/var/run/"+consts.SlurmctldName+".pid")
	res.AddProperty("SlurmctldPort", consts.SlurmctldPort)
	res.AddComment("")
	res.AddProperty("SlurmdPidFile", "/var/run/"+consts.SlurmdName+".pid")
	res.AddProperty("SlurmdPort", consts.SlurmdPort)
	res.AddComment("")
	res.AddProperty("SlurmdSpoolDir", utils.BuildVolumeMountSpoolPath(consts.SlurmdName))
	res.AddComment("")
	res.AddProperty("SlurmUser", consts.SlurmUser)
	res.AddProperty("SlurmdUser", consts.SlurmUser)
	res.AddComment("")
	res.AddProperty("StateSaveLocation", utils.BuildVolumeMountSpoolPath(consts.SlurmctldName))
	res.AddComment("")

	res.AddProperty("AuthType", "auth/"+consts.Munge)
	res.AddProperty("CredType", "cred/"+consts.Munge)
	res.AddComment("")
	res.AddComment("SlurnConfig Spec")
	res.AddComment("")
	res.AddProperty("GresTypes", "gpu,shard")
	res.AddProperty("MailProg", "/usr/bin/true")
	res.AddComment("")

	// res.AddProperty("TaskPlugin", "task/affinity")
	res.AddComment("")
	res.AddProperty("CliFilterPlugins", "cli_filter/user_defaults")
	res.AddComment("")
	res.AddProperty("LaunchParameters", "use_interactive_step")
	res.AddComment("Scrontab")
	res.AddProperty("ScronParameters", "enable,explicit_scancel")
	res.AddComment("")
	res.AddProperty("PropagateResourceLimits", "NONE") // Don't propagate ulimits from the login node by default
	res.AddComment("")
	res.AddProperty("PrologFlags", "Contain")
	// res.AddProperty("SlurmctldParameters", "enable_configless,enable_stepmgr")
	res.AddProperty("SlurmctldParameters", "enable_stepmgr")
	res.AddComment("")
	res.AddProperty("ProctrackType", "proctrack/linuxproc")
	res.AddComment("")

	res.AddProperty("JobAcctGatherType", "jobacct_gather/linux")
	res.AddProperty("JobAcctGatherFrequency", "task=30")
	res.AddProperty("AccountingStorageTRES", "gres/gpu,gres/shard")
	res.AddProperty("AccountingStorageType", "accounting_storage/slurmdbd")
	res.AddProperty("AccountingStorageHost", utils.BuildServiceFQDN(consts.ComponentTypeAccounting, cluster.Namespace, cluster.Name))
	res.AddProperty("AccountingStoragePort", consts.SlurmdbdPort)
	res.AddComment("")

	res.AddProperty("InactiveLimit", 0)
	res.AddProperty("KillWait", 180)
	res.AddProperty("UnkillableStepTimeout", 600)
	res.AddProperty("SlurmctldTimeout", 30)
	res.AddProperty("SlurmdTimeout", 180)
	res.AddProperty("WaitTime", 0)
	res.AddComment("")
	res.AddComment("SCHEDULING")
	res.AddProperty("SchedulerType", "sched/backfill")
	res.AddProperty("SelectType", "select/cons_tres")
	res.AddProperty("SelectTypeParameters", "CR_Core_Memory")
	res.AddComment("")
	res.AddComment("LOGGING")
	res.AddProperty("SlurmctldDebug", consts.SlurmDefaultDebugLevel)
	res.AddProperty("SlurmctldLogFile", consts.SlurmLogFile)
	res.AddProperty("SlurmdDebug", consts.SlurmDefaultDebugLevel)
	res.AddProperty("SlurmdLogFile", consts.SlurmLogFile)
	res.AddComment("")
	res.AddComment("Partition Configuration")
	res.AddProperty("JobRequeue", 1)
	res.AddProperty("PreemptMode", "REQUEUE")
	res.AddProperty("PreemptType", "preempt/partition_prio")

	for _, nodeSet := range cluster.Spec.ComputingNodeSets {
		for i := int32(0); i < nodeSet.Size; i++ {
			hostName, hostFQDN := utils.BuildServiceHostFQDN(
				utils.StringToComponent(consts.ComponentTypeComputing.String(), nodeSet.PartitionName),
				cluster.Namespace,
				cluster.Name,
				i,
			)

			quantityToInt := func(quantity resource.Quantity) int64 {
				num, ok := quantity.AsInt64()
				if !ok {
					unscaledUnm := quantity.AsDec().UnscaledBig().Int64()
					if unscaledUnm < 1000 {
						num = 1
					} else {
						num = unscaledUnm / 1000
					}
				}
				return num
			}

			cpuNum := quantityToInt(nodeSet.SlurmdContainer.CPU)
			memMB := quantityToInt(nodeSet.SlurmdContainer.Memory) / 1024 / 1024

			if nodeSet.GPU != "" {
				gres := "gpu:nvidia:1,shard:nvidia:8"
				res.AddLine(fmt.Sprintf(
					"NodeName=%s-%d NodeHostname=%s NodeAddr=%s CPUs=%d RealMemory=%d Gres=%s Features=%s",
					nodeSet.PartitionName, i, hostName, hostFQDN, cpuNum, memMB, gres, nodeSet.PartitionName))
			} else {
				res.AddLine(fmt.Sprintf(
					"NodeName=%s-%d NodeHostname=%s NodeAddr=%s CPUs=%d RealMemory=%d Features=%s",
					nodeSet.PartitionName, i, hostName, hostFQDN, cpuNum, memMB, nodeSet.PartitionName))
			}
		}
	}
	for _, nodeSet := range cluster.Spec.ComputingNodeSets {
		res.AddLine(fmt.Sprintf("NodeSet=%s Feature=%s", nodeSet.PartitionName, nodeSet.PartitionName))
	}
	for _, nodeSet := range cluster.Spec.ComputingNodeSets {
		res.AddLine(fmt.Sprintf("PartitionName=%s Nodes=%s State=UP Default=YES MaxTime=INFINITE", nodeSet.PartitionName, nodeSet.PartitionName))
	}
	res.AddLine("\n")

	return res
}

func generateCgroupConfig() utils.ConfigFile {
	res := &utils.PropertiesConfig{}
	res.AddComment("")
	res.AddProperty("CgroupPlugin", "disabled")
	res.AddLine("\n")
	return res
}

func generateSSSdConfig() utils.ConfigFile {
	res := &utils.PropertiesConfig{}
	res.AddLine("[sssd]")
	res.AddProperty("config_file_version", 2)
	res.AddProperty("debug_level", 6)
	res.AddProperty("services", "nss, pam")
	res.AddProperty("domains", "LDAP")
	res.AddLine("\n")
	res.AddLine("[domain/LDAP]")
	res.AddProperty("debug_level", 6)
	res.AddProperty("id_provider", "ldap")
	res.AddProperty("auth_provider", "ldap")
	res.AddProperty("ldap_auth_disable_tls_never_use_in_production", config.LdapConfig.SkipInsecureTLS)
	res.AddProperty("ldap_uri", config.LdapConfig.LDAPUri)
	res.AddProperty("ldap_search_base", config.LdapConfig.LDAPSearchBase)
	res.AddProperty("ldap_default_bind_dn", config.LdapConfig.LDAPDefaultBindDN)
	res.AddProperty("ldap_default_authtok_type", config.LdapConfig.LDAPDefaultAuthtokType)
	res.AddProperty("ldap_default_authtok", config.LdapConfig.LDAPDefaultAuthtok)
	res.AddProperty("entry_cache_timeout", 0)
	res.AddLine("\n")
	res.AddLine("[nss]")
	res.AddProperty("memcache_timeout", 0)
	res.AddLine("\n")
	return res
}

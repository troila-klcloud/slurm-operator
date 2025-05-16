package controller

import (
	"fmt"
	"path"

	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func renderMungeContainer(spec slurmv1alpha1.CommonContainerSpec) corev1.Container {
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.CPU, corev1.ResourceMemory: spec.Memory}
	return corev1.Container{
		Name:            consts.Munge,
		Image:           spec.Image,
		ImagePullPolicy: spec.ImagePullPolicy,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "munge-run",
				MountPath: "/run/munge/",
			},
			{
				Name:      "munge-key",
				MountPath: "/etc/munge/munge.key",
				SubPath:   consts.SecretMungeKeyFileName,
				ReadOnly:  true,
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits:   resourceList,
			Requests: resourceList,
		},
		RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/bin/sh",
						"-c",
						fmt.Sprintf(
							"test -S %s",
							path.Join(consts.VolumeMountPathMungeSocket, "munge.socket.2"),
						),
					},
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/bin/sh",
						"-c",
						"/usr/bin/munge -n > /dev/null && exit 0 || exit 1",
					},
				},
			},
		},
	}
}

func renderSSSDContainer(spec slurmv1alpha1.CommonContainerSpec) corev1.Container {
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.CPU, corev1.ResourceMemory: spec.Memory}
	return corev1.Container{
		Name:            "sssd",
		Image:           spec.Image,
		ImagePullPolicy: spec.ImagePullPolicy,
		VolumeMounts: []corev1.VolumeMount{
			renderSSSLibVolumeMount(),
			renderSSSDConfigVolumeMount(),
		},
		Resources: corev1.ResourceRequirements{
			Limits:   resourceList,
			Requests: resourceList,
		},
	}
}

func renderSSSLibVolume() corev1.Volume {
	return corev1.Volume{
		Name: "sss-lib-dir",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func renderSSSLibVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "sss-lib-dir",
		MountPath: "/var/lib/sss/",
	}
}

func renderSSSDConfigVolume(clusterName string) corev1.Volume {
	return corev1.Volume{
		Name: "sssd-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: utils.BuildConfigMapSlurmConfigsName(clusterName),
				},
				Items: []corev1.KeyToPath{
					{Key: consts.ConfigMapKeySSSDConfig, Path: consts.ConfigMapKeySSSDConfig},
				},
				DefaultMode: ptr.To(int32(0600)),
			},
		},
	}
}

func renderSSSDConfigVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "sssd-config",
		MountPath: "/etc/sssd/sssd.conf",
		SubPath:   "sssd.conf",
	}
}

func renderMungeRunVolume() corev1.Volume {
	return corev1.Volume{
		Name: "munge-run",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func renderMungeKeyVolume(clusterName string) corev1.Volume {
	return corev1.Volume{
		Name: "munge-key",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: utils.BuildSecretName(consts.ComponentTypeMunge, clusterName),
				Items: []corev1.KeyToPath{
					{
						Key:  consts.SecretMungeKeyFileName,
						Path: consts.SecretMungeKeyFileName,
						Mode: ptr.To(int32(0600)),
					},
				},
			},
		},
	}

}

func renderHomeDirVolume(cluster slurmv1alpha1.Cluster) corev1.Volume {
	return corev1.Volume{
		Name: "home-dir",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: cluster.Spec.PersistentVolumeClaimName,
			},
		},
	}
}

func renderSlurmConfigVolume(cluster slurmv1alpha1.Cluster, withGres bool) corev1.Volume {
	vol := corev1.Volume{
		Name: "slurm-config",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: utils.BuildConfigMapSlurmConfigsName(cluster.Name)},
				Items: []corev1.KeyToPath{
					{Key: consts.ConfigMapKeySlurmConfig, Path: consts.ConfigMapKeySlurmConfig},
					{Key: consts.ConfigMapKeyCGroupConfig, Path: consts.ConfigMapKeyCGroupConfig},
				},
			},
		},
	}
	return vol
}

func renderSlurmCtrlSpoolVolume(pvcName string) corev1.Volume {
	return corev1.Volume{
		Name: "slurm-spool",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}
}

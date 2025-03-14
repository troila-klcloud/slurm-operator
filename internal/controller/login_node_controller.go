package controller

import (
	"context"

	"github.com/pkg/errors"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateLoginNodeDeployment(cluster slurmv1alpha1.Cluster, spec slurmv1alpha1.LoginNodeSpec) *appsv1.Deployment {
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.SshdContainer.CPU, corev1.ResourceMemory: spec.SshdContainer.Memory}
	labels := render.RenderLabels(consts.ComponentTypeLogin, cluster.Name)
	matchLabels := render.RenderMatchLabels(consts.ComponentTypeLogin, cluster.Name)
	// var sssdConfigFileMode int32 = 0600
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildDeploymentName(consts.ComponentTypeLogin, cluster.Name),
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(spec.Size),
			Selector: &metav1.LabelSelector{MatchLabels: matchLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						renderMungeContainer(spec.MungeContainer),
					},
					Containers: []corev1.Container{
						renderSSSDContainer(spec.SssdContainer),
						{
							Name:            "sshd",
							Image:           spec.SshdContainer.Image,
							ImagePullPolicy: spec.SshdContainer.ImagePullPolicy,
							Resources: corev1.ResourceRequirements{
								Limits:   resourceList,
								Requests: resourceList,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "home-dir", MountPath: "/home"},
								{Name: "slurm-config", MountPath: "/etc/slurm/"},
								{Name: "munge-run", MountPath: "/run/munge/"},
								{
									Name:      "munge-key",
									MountPath: "/etc/munge/munge.key",
									SubPath:   consts.SecretMungeKeyFileName,
									ReadOnly:  true,
								},
								renderSSSLibVolumeMount(),
							},
						},
					},
					Volumes: []corev1.Volume{
						renderHomeDirVolume(cluster),
						renderSSSDConfigVolume(cluster.Name),
						renderSSSLibVolume(),
						renderSlurmConfigVolume(cluster, false),
						renderMungeRunVolume(),
						renderMungeKeyVolume(cluster.Name),
					},
				},
			},
		},
	}
}

func generateLoginNodeService(cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.LoginNodeSpec) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildServiceName(consts.ComponentTypeLogin, cluster.Name),
			Namespace: cluster.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(spec.ServiceType),
			Selector: render.RenderMatchLabels(consts.ComponentTypeLogin, cluster.Name),
			Ports: []corev1.ServicePort{
				{
					Name:       "ssh",
					Protocol:   corev1.ProtocolTCP,
					Port:       22,
					TargetPort: intstr.FromInt32(22)},
			},
		},
	}
}

func (r *ClusterReconciler) reconcileLoginNode(ctx context.Context, cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.LoginNodeSpec) error {
	log := log.FromContext(ctx)

	expectedDep := generateLoginNodeDeployment(*cluster, spec)
	foundDep, err := r.getOrCreateK8sResource(ctx, cluster, expectedDep)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile login node deployment")
	}
	available := false
	if foundDep != nil {
		deployObj := foundDep.(*appsv1.Deployment)
		for _, c := range deployObj.Status.Conditions {
			if c.Type == consts.ConditionTypeAvailable && c.Status == consts.ConditionStatusTrue {
				err = r.setStatusCondition(
					ctx, cluster,
					&metav1.Condition{
						Type:    consts.LoginNodeConditionType,
						Status:  metav1.ConditionTrue,
						Reason:  "Available",
						Message: "Login node is available",
					})
				if err != nil {
					return err
				}
				available = true
				break
			}
		}
		patch := client.MergeFrom(deployObj.DeepCopy())
		deployObj.Spec.Replicas = ptr.To(spec.Size)
		deployObj.Spec.Template.Spec.Containers = expectedDep.Spec.Template.Spec.Containers
		if err = r.Patch(ctx, deployObj, patch); err != nil {
			msg := "Failed to patch login node deployment"
			log.Error(err, msg)
			return errors.Wrap(err, msg)
		}
	}
	if !available {
		err = r.setStatusCondition(
			ctx, cluster,
			&metav1.Condition{
				Type:    consts.LoginNodeConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  "NotAvailable",
				Message: "Login node is not available",
			})
		if err != nil {
			return err
		}
	}

	expectedSvc := generateLoginNodeService(cluster, spec)
	foundSvc, err := r.getOrCreateK8sResource(ctx, cluster, expectedSvc)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile login node service")
	}
	if foundSvc != nil {
		svcObj := foundSvc.(*corev1.Service)
		patch := client.MergeFrom(svcObj.DeepCopy())
		svcObj.Spec.Type = expectedSvc.Spec.Type
		if err = r.Patch(ctx, svcObj, patch); err != nil {
			msg := "Failed to patch login node service"
			log.Error(err, msg)
			return errors.Wrap(err, msg)
		}

		if err = r.updateCluserNodePort(ctx, cluster, svcObj); err != nil {
			return err
		}
	}

	return nil
}

func (r *ClusterReconciler) updateCluserNodePort(ctx context.Context, cluster *slurmv1alpha1.Cluster, svc *corev1.Service) error {
	log := log.FromContext(ctx)
	patch := client.MergeFrom(cluster.DeepCopy())
	for _, port := range svc.Spec.Ports {
		if port.Name == "ssh" {
			cluster.Status.NodePort = port.NodePort
			if err := r.Status().Patch(ctx, cluster, patch); err != nil {
				msg := "Failed to set cluster nodePort"
				log.Error(err, msg)
				return errors.Wrap(err, msg)
			}
		}
	}
	return nil
}

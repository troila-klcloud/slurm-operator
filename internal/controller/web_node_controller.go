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

func generateWebNodeDeployment(cluster slurmv1alpha1.Cluster, spec slurmv1alpha1.WebNodeSpec) *appsv1.Deployment {
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.WebContainer.CPU, corev1.ResourceMemory: spec.WebContainer.Memory}
	labels := render.RenderLabels(consts.ComponentTypeWeb, cluster.Name)
	matchLabels := render.RenderMatchLabels(consts.ComponentTypeWeb, cluster.Name)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildDeploymentName(consts.ComponentTypeWeb, cluster.Name),
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
							Name: "web",
							Env: []corev1.EnvVar{
								{Name: "DATABASE_HOST", Value: utils.BuildMariaDbName(cluster.Name)},
								{Name: "DATABASE_PORT", Value: "3306"},
								{Name: "DATABASE_USER", Value: consts.MariaDbUsername},
								{
									Name: "DATABASE_USER_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)},
											Key: consts.SlurmPassword,
										},
									},
								},
								{
									Name: "DATABASE_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)},
											Key: consts.RootPassword,
										},
									},
								},
								{Name: "DEPLOY_NODE_IP", Value: utils.BuildMariaDbName(cluster.Name)},
								{
									Name: "DATASOURCE_PASSWORD", ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)},
											Key: consts.RootPassword,
										},
									},
								},
							},
							Image:           spec.WebContainer.Image,
							ImagePullPolicy: spec.WebContainer.ImagePullPolicy,
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
							Resources: corev1.ResourceRequirements{
								Limits:   resourceList,
								Requests: resourceList,
							},
						},
					},
					Volumes: []corev1.Volume{
						renderSSSDConfigVolume(cluster.Name),
						renderSSSLibVolume(),
						renderHomeDirVolume(cluster),
						renderSlurmConfigVolume(cluster, false),
						renderMungeRunVolume(),
						renderMungeKeyVolume(cluster.Name),
					},
				},
			},
		},
	}

	if spec.InitSqlContainer.Image != "" {
		initSqlResourceList := corev1.ResourceList{corev1.ResourceCPU: spec.InitSqlContainer.CPU, corev1.ResourceMemory: spec.InitSqlContainer.Memory}
		deploy.Spec.Template.Spec.InitContainers = append(deploy.Spec.Template.Spec.InitContainers, corev1.Container{
			Name:            "init-sql",
			ImagePullPolicy: spec.WebContainer.ImagePullPolicy,
			Image:           spec.InitSqlContainer.Image,
			Resources: corev1.ResourceRequirements{
				Limits:   initSqlResourceList,
				Requests: initSqlResourceList,
			},
			Env: []corev1.EnvVar{
				{Name: "DATABASE_HOST", Value: utils.BuildMariaDbName(cluster.Name)},
				{Name: "DATABASE_PORT", Value: "3306"},
				{Name: "DATABASE_USER", Value: consts.MariaDbUsername},
				{
					Name: "DATABASE_USER_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)},
							Key: consts.SlurmPassword,
						},
					},
				},
				{
					Name: "DATABASE_ROOT_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)},
							Key: consts.RootPassword,
						},
					},
				},
				{Name: "hpc_init_db_ip", Value: utils.BuildMariaDbName(cluster.Name)},
				{
					Name: "hpc_init_db_password", ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)},
							Key: consts.RootPassword,
						},
					},
				},
			},
		})
	}
	return deploy
}

func generateWebNodeService(cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.WebNodeSpec) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildServiceName(consts.ComponentTypeWeb, cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    render.RenderLabels(consts.ComponentTypeWeb, cluster.Name),
		},
		Spec: corev1.ServiceSpec{
			Selector: render.RenderMatchLabels(consts.ComponentTypeWeb, cluster.Name),
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "web-port",
					Protocol:   corev1.ProtocolTCP,
					Port:       spec.Port,
					TargetPort: intstr.FromInt32(spec.Port),
				},
			},
		},
	}
}

func (r *ClusterReconciler) reconcileWebNode(ctx context.Context, cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.WebNodeSpec) error {
	log := log.FromContext(ctx)

	expectedDeploy := generateWebNodeDeployment(*cluster, spec)
	foundDeploy, err := r.getOrCreateK8sResource(ctx, cluster, expectedDeploy)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile web node service")
	}
	available := false
	if foundDeploy != nil {
		deployObj := foundDeploy.(*appsv1.Deployment)
		for _, c := range deployObj.Status.Conditions {
			if c.Type == consts.ConditionTypeAvailable && c.Status == consts.ConditionStatusTrue {
				err = r.setStatusCondition(
					ctx, cluster,
					&metav1.Condition{
						Type:    consts.WebNodeConditionType,
						Status:  metav1.ConditionTrue,
						Reason:  "Available",
						Message: "Web node is available",
					})
				if err != nil {
					return err
				}
				available = true
				break
			}
		}
		patch := client.MergeFrom(deployObj.DeepCopy())
		deployObj.Spec.Replicas = expectedDeploy.Spec.Replicas
		deployObj.Spec.Template.Spec.Containers = expectedDeploy.Spec.Template.Spec.Containers
		if err = r.Patch(ctx, deployObj, patch); err != nil {
			msg := "Failed to patch web node deployment"
			log.Error(err, msg)
			return errors.Wrap(err, msg)
		}
	}
	if !available {
		log.Info("Web node not available")
		err = r.setStatusCondition(
			ctx, cluster,
			&metav1.Condition{
				Type:    consts.WebNodeConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  "NotAvailable",
				Message: "Web node is not available"})
		if err != nil {
			return err
		}
	}

	expectedSvc := generateWebNodeService(cluster, spec)
	_, err = r.getOrCreateK8sResource(ctx, cluster, expectedSvc)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile web node service")
	}

	return nil
}

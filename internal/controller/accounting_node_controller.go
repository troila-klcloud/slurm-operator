package controller

import (
	"context"
	"time"

	"github.com/pkg/errors"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateSlurmdbdConfig(password []byte, clusterName string) utils.ConfigFile {
	res := &utils.PropertiesConfig{}
	res.AddProperty("AuthType", "auth/"+consts.Munge)
	res.AddProperty("SlurmUser", consts.SlurmUser)
	res.AddProperty("PidFile", consts.SlurmdbdPidFile)
	// res.AddProperty("DbdHost", consts.HostnameAccounting)
	res.AddProperty("DbdPort", consts.SlurmdbdPort)
	res.AddProperty("StorageLoc", "slurm_acct_db")
	res.AddProperty("StorageType", "accounting_storage/mysql")
	res.AddProperty("StoragePass", string(password))
	res.AddProperty("StorageUser", consts.MariaDbUsername)
	res.AddProperty("StorageHost", utils.BuildMariaDbName(clusterName))
	res.AddProperty("StoragePort", consts.MariaDbPort)
	return res
}

func generateSlurmdbdConfigSecret(cluster *slurmv1alpha1.Cluster, passSecret *corev1.Secret) (*corev1.Secret, error) {
	password, exists := passSecret.Data[consts.SlurmPassword]
	if !exists {
		return nil, errors.New("cannot get databases password for slurm user")
	}
	data := map[string][]byte{
		consts.ConfigMapKeySlurmdbdConfig: []byte(generateSlurmdbdConfig(password, cluster.Name).Render()),
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildSecretName(consts.ComponentTypeAccounting, cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    render.RenderLabels(consts.ComponentTypeAccounting, cluster.Name),
		},
		Data: data,
	}, nil
}

func generateAccountingNodeDeployment(cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.AccountingNodeSpec) *appsv1.Deployment {
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.SlurmdbdContainer.CPU, corev1.ResourceMemory: spec.SlurmdbdContainer.Memory}
	labels := render.RenderLabels(consts.ComponentTypeAccounting, cluster.Name)
	matchLabels := render.RenderMatchLabels(consts.ComponentTypeAccounting, cluster.Name)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildDeploymentName(consts.ComponentTypeAccounting, cluster.Name),
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
						{
							Name:            "slurm",
							Image:           spec.SlurmdbdContainer.Image,
							ImagePullPolicy: spec.SlurmdbdContainer.ImagePullPolicy,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "slurmdb-config",
									MountPath: "/etc/slurm/slurmdbd.conf",
									SubPath:   consts.ConfigMapKeySlurmdbdConfig,
									ReadOnly:  true,
								}, {
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
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt32(consts.SlurmdbdPort),
									},
								},
								FailureThreshold:    3,
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								TimeoutSeconds:      1,
							},
						},
					},
					Volumes: []corev1.Volume{
						renderMungeKeyVolume(cluster.Name),
						renderMungeRunVolume(),
						{
							Name: "slurmdb-config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: utils.BuildSecretName(consts.ComponentTypeAccounting, cluster.Name),
									Items: []corev1.KeyToPath{
										{Key: consts.ConfigMapKeySlurmdbdConfig, Path: consts.ConfigMapKeySlurmdbdConfig},
									},
									DefaultMode: ptr.To(int32(0600)),
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateAccountingNodeService(cluster *slurmv1alpha1.Cluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildServiceName(consts.ComponentTypeAccounting, cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    render.RenderLabels(consts.ComponentTypeAccounting, cluster.Name),
		},
		Spec: corev1.ServiceSpec{
			Selector: render.RenderMatchLabels(consts.ComponentTypeAccounting, cluster.Name),
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "slurmdb-port",
					Protocol:   corev1.ProtocolTCP,
					Port:       consts.SlurmdbdPort,
					TargetPort: intstr.FromInt32(consts.SlurmdbdPort),
				},
			},
		},
	}
}

func (r *ClusterReconciler) reconcileAccountingNode(ctx context.Context, cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.AccountingNodeSpec) (*ctrl.Result, error) {
	log := log.FromContext(ctx)
	passSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name)}, passSecret); err != nil {
		msg := "Failed to get mariadb password secret"
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}

	expectedSecret, err := generateSlurmdbdConfigSecret(cluster, passSecret)
	if err != nil {
		msg := "Failed to generate slurmdbd config secret"
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}
	foundSecret, err := r.getOrCreateK8sResource(ctx, cluster, expectedSecret)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to reconcile slurmdbd config secret")
	}
	if foundSecret != nil {
		secretObj := foundSecret.(*corev1.Secret)
		secretObj.Data = expectedSecret.Data
		if err = r.Update(ctx, secretObj); err != nil {
			msg := "Failed to update slurmdbd config secret"
			log.Error(err, msg)
			return nil, errors.Wrap(err, msg)
		}
	}

	expectedDeploy := generateAccountingNodeDeployment(cluster, spec)
	foundDeploy, err := r.getOrCreateK8sResource(ctx, cluster, expectedDeploy)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to reconcile accounting node deployment")
	}
	available := false
	if foundDeploy != nil {
		deployObj := foundDeploy.(*appsv1.Deployment)
		for _, c := range deployObj.Status.Conditions {
			if c.Type == consts.ConditionTypeAvailable && c.Status == consts.ConditionStatusTrue {
				err = r.setStatusCondition(
					ctx, cluster,
					&metav1.Condition{
						Type:    consts.AccountingNodeConditionType,
						Status:  metav1.ConditionTrue,
						Reason:  "Available",
						Message: "Accounting node is available",
					})
				if err != nil {
					return nil, err
				}
				available = true
				break
			}
		}
		patch := client.MergeFrom(deployObj.DeepCopy())
		deployObj.Spec.Replicas = expectedDeploy.Spec.Replicas
		deployObj.Spec.Template.Spec.Containers = expectedDeploy.Spec.Template.Spec.Containers
		if err = r.Patch(ctx, deployObj, patch); err != nil {
			msg := "Failed to patch accounting node deployment"
			log.Error(err, msg)
			return nil, errors.Wrap(err, msg)
		}
	}
	if !available {
		log.Info("Accounting node not available")
		err = r.setStatusCondition(
			ctx, cluster,
			&metav1.Condition{
				Type:    consts.AccountingNodeConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  "NotAvailable",
				Message: "Accounting node is not available"})
		if err != nil {
			return nil, err
		}
		return &ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	expectedSvc := generateAccountingNodeService(cluster)
	_, err = r.getOrCreateK8sResource(ctx, cluster, expectedSvc)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to reconcile accounting node service")
	}

	return nil, nil
}

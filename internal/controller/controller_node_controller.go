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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateControllerPersistentVolumeClaim(cluster slurmv1alpha1.Cluster) (*corev1.PersistentVolumeClaim, error) {
	storageSize, err := resource.ParseQuantity("100Mi")
	if err != nil {
		return nil, err
	}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildPersistentVolumeClaimName(consts.ComponentTypeController, cluster.Name),
			Namespace: cluster.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
			},
			StorageClassName: &cluster.Spec.CtrlNode.SpoolStorageClassName,
		},
	}, nil
}

func generateControllerNodeStatefulSet(cluster slurmv1alpha1.Cluster, spec slurmv1alpha1.CtrlNodeSpec) *appsv1.StatefulSet {
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.SlurmCtldContainer.CPU, corev1.ResourceMemory: spec.SlurmCtldContainer.Memory}
	labels := render.RenderLabels(consts.ComponentTypeController, cluster.Name)
	matchLabels := render.RenderMatchLabels(consts.ComponentTypeController, cluster.Name)
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        utils.BuildStatefulSetName(consts.ComponentTypeController, cluster.Name),
			Namespace:   cluster.Namespace,
			Annotations: render.RenderWaveAnnotations(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    ptr.To(spec.Size),
			ServiceName: utils.BuildServiceName(consts.ComponentTypeController, cluster.Name),
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
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
							Name:            "slurm",
							Image:           spec.SlurmCtldContainer.Image,
							ImagePullPolicy: spec.SlurmCtldContainer.ImagePullPolicy,
							Resources: corev1.ResourceRequirements{
								Limits:   resourceList,
								Requests: resourceList,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "home-dir", MountPath: "/home"},
								{Name: "slurm-config", MountPath: "/etc/slurm/"},
								{Name: "slurm-spool", MountPath: "/var/spool/"},
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
						renderSlurmCtrlSpoolVolume(utils.BuildPersistentVolumeClaimName(consts.ComponentTypeController, cluster.Name)),
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
}

func (r *ClusterReconciler) reconcileControllerNode(ctx context.Context, cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.CtrlNodeSpec) error {
	log := log.FromContext(ctx)

	expectedSvc := generateHeadlessService(consts.ComponentTypeController, *cluster, consts.SlurmctldPort)
	_, err := r.getOrCreateK8sResource(ctx, cluster, expectedSvc)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile controller node service")
	}

	expectedPvc, err := generateControllerPersistentVolumeClaim(*cluster)
	if err != nil {
		return errors.Wrap(err, "Failed to generate controller pvc")
	}
	_, err = r.getOrCreateK8sResource(ctx, cluster, expectedPvc)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile controller pvc")
	}

	expectedSts := generateControllerNodeStatefulSet(*cluster, spec)
	foundSts, err := r.getOrCreateK8sResource(ctx, cluster, expectedSts)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile controller node statefulset")
	}
	available := false
	if foundSts != nil {
		stsObj := foundSts.(*appsv1.StatefulSet)
		if stsObj.Status.AvailableReplicas != 0 {
			err = r.setStatusCondition(
				ctx, cluster,
				&metav1.Condition{
					Type:    consts.ControllerNodeConditionType,
					Status:  metav1.ConditionTrue,
					Reason:  "Available",
					Message: "Controller node is available",
				})
			if err != nil {
				return err
			}
			available = true
		}
		patch := client.MergeFrom(stsObj.DeepCopy())
		stsObj.Spec.Replicas = expectedSts.Spec.Replicas
		stsObj.Spec.Template.Spec.Containers = expectedSts.Spec.Template.Spec.Containers
		if err = r.Patch(ctx, stsObj, patch); err != nil {
			msg := "Failed to patch controller node statefulset"
			log.Error(err, msg)
			return errors.Wrap(err, msg)
		}
	}
	if !available {
		err = r.setStatusCondition(
			ctx, cluster,
			&metav1.Condition{
				Type:    consts.ControllerNodeConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  "NotAvailable",
				Message: "Controller node is not available",
			})
		if err != nil {
			return err
		}
	}

	return nil
}

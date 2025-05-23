package controller

import (
	"context"
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/pkg/errors"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateComputingNodeSetStatefulSet(cluster slurmv1alpha1.Cluster, spec slurmv1alpha1.ComputingNodeSetSpec) *appsv1.StatefulSet {
	component := utils.StringToComponent(consts.ComponentTypeComputing.String(), spec.PartitionName)
	resourceList := corev1.ResourceList{corev1.ResourceCPU: spec.SlurmdContainer.CPU, corev1.ResourceMemory: spec.SlurmdContainer.Memory}
	requireGresConfig := false
	if spec.GPU != "" {
		resourceList[corev1.ResourceName(spec.GPU)] = resource.MustParse("1")
		requireGresConfig = true
	}
	labels := render.RenderLabels(component, cluster.Name)
	matchLabels := render.RenderMatchLabels(component, cluster.Name)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        utils.BuildStatefulSetName(component, cluster.Name),
			Namespace:   cluster.Namespace,
			Annotations: render.RenderWaveAnnotations(),
			Labels:      render.RenderComputingNodeSetLables(cluster.Name, spec.PartitionName),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: utils.BuildServiceName(component, cluster.Name),
			Replicas:    ptr.To(spec.Size),
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
							Image:           spec.SlurmdContainer.Image,
							ImagePullPolicy: spec.SlurmdContainer.ImagePullPolicy,
							Resources: corev1.ResourceRequirements{
								Limits:   resourceList,
								Requests: resourceList,
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "home-dir", MountPath: "/home"},
								{Name: "slurm-config", MountPath: "/tmp/slurm/"},
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
						renderSSSDConfigVolume(cluster.Name),
						renderSSSLibVolume(),
						renderHomeDirVolume(cluster),
						renderSlurmConfigVolume(cluster, requireGresConfig),
						renderMungeRunVolume(),
						renderMungeKeyVolume(cluster.Name),
					},
				},
			},
		},
	}
	if spec.GPU != "" {
		sts.Spec.Template.Spec.RuntimeClassName = ptr.To("nvidia")
	}
	return sts
}

func (r *ClusterReconciler) reconcileComputingNodeSets(ctx context.Context, cluster *slurmv1alpha1.Cluster, nodeSets []slurmv1alpha1.ComputingNodeSetSpec) error {
	log := log.FromContext(ctx)
	stsList := appsv1.StatefulSetList{}
	stsSelector := labels.Set(render.RenderComputingNodeSetMatchLabels(cluster.Name)).AsSelector()
	if err := r.List(ctx, &stsList, &client.ListOptions{LabelSelector: stsSelector}); err != nil {
		msg := fmt.Sprintf("Failed to list computing node set, cluster: %s", cluster.Name)
		log.Error(err, msg)
		return errors.Wrap(err, msg)
	}
	for _, sts := range stsList.Items {
		needDelete := true
		for _, nodeSet := range nodeSets {
			if sts.Labels[consts.LabelComputingSetPartitonKey] == nodeSet.PartitionName {
				needDelete = false
				break
			}
		}
		if needDelete {
			if err := r.Delete(ctx, &sts); err != nil {
				msg := fmt.Sprintf("Failed to delete redundant statefulset: %s", sts.Name)
				log.Error(err, msg)
				return errors.Wrap(err, msg)
			}
		}
	}
	for _, nodeSet := range nodeSets {
		component := utils.StringToComponent(consts.ComponentTypeComputing.String(), nodeSet.PartitionName)

		expectedSvc := generateHeadlessService(component, *cluster, consts.SlurmdPort)
		_, err := r.getOrCreateK8sResource(ctx, cluster, expectedSvc)
		if err != nil {
			return errors.Wrap(err, "Failed to reconcile computing node set service")
		}

		expectedSts := generateComputingNodeSetStatefulSet(*cluster, nodeSet)
		foundSts, err := r.getOrCreateK8sResource(ctx, cluster, expectedSts)
		if err != nil {
			return errors.Wrap(err, "Failed to reconcile computing node set statefulset")
		}
		available := false
		if foundSts != nil {
			objSts := foundSts.(*appsv1.StatefulSet)
			if objSts.Status.AvailableReplicas != 0 {
				err = r.setStatusCondition(
					ctx, cluster,
					&metav1.Condition{
						Type:    fmt.Sprintf(consts.ComputingNodeSetConditionType, cases.Title(language.English, cases.Compact).String(nodeSet.PartitionName)),
						Status:  metav1.ConditionTrue,
						Reason:  "Available",
						Message: fmt.Sprintf("Computing node set %s is available", nodeSet.PartitionName),
					})
				if err != nil {
					return err
				}
				available = true
			}
			patch := client.MergeFrom(objSts.DeepCopy())
			objSts.Labels = expectedSts.Labels
			objSts.Spec.Replicas = expectedSts.Spec.Replicas
			objSts.Spec.Template.Spec.Containers = expectedSts.Spec.Template.Spec.Containers
			objSts.Spec.Template.Spec.Volumes = expectedSts.Spec.Template.Spec.Volumes
			if err = r.Patch(ctx, objSts, patch); err != nil {
				msg := "Failed to patch computing node set statefulset"
				log.Error(err, msg)
				return errors.Wrap(err, msg)
			}
			podList := &corev1.PodList{}
			selector := labels.Set(objSts.Spec.Selector.MatchLabels).AsSelector()
			if err := r.List(ctx, podList, &client.ListOptions{LabelSelector: selector}); err != nil {
				msg := fmt.Sprintf("Failed to list pod own to statefulset: %s", objSts.Name)
				log.Error(err, msg)
				return errors.Wrap(err, msg)
			}
			cpuAmount := resource.Quantity{}
			memAmount := resource.Quantity{}
			for _, container := range expectedSts.Spec.Template.Spec.Containers {
				if container.Name == "slurm" {
					for key, value := range container.Resources.Requests {
						if key == corev1.ResourceCPU {
							cpuAmount = value
						}
						if key == corev1.ResourceMemory {
							memAmount = value
						}
					}
				}
			}
			for _, pod := range podList.Items {
				if pod.Status.Phase == corev1.PodPending {
					for _, container := range pod.Spec.Containers {
						if container.Name == "slurm" {
							for key, value := range container.Resources.Requests {
								if key == corev1.ResourceCPU {
									if cpuAmount != value {
										if err := r.Delete(ctx, &pod); err != nil {
											msg := fmt.Sprintf("Failed to delete pending pod: %s", pod.Name)
											log.Error(err, msg)
											return errors.Wrap(err, msg)
										}
									}
								}
								if key == corev1.ResourceMemory {
									if memAmount != value {
										if err := r.Delete(ctx, &pod); err != nil {
											msg := fmt.Sprintf("Failed to delete pending pod: %s", pod.Name)
											log.Error(err, msg)
											return errors.Wrap(err, msg)
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if !available {
			err = r.setStatusCondition(
				ctx, cluster,
				&metav1.Condition{
					Type:    fmt.Sprintf(consts.ComputingNodeSetConditionType, cases.Title(language.English, cases.Compact).String(nodeSet.PartitionName)),
					Status:  metav1.ConditionFalse,
					Reason:  "NotAvailable",
					Message: fmt.Sprintf("Computing node set %s is not available", nodeSet.PartitionName),
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

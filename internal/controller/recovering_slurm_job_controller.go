package controller

import (
	"context"
	"github.com/pkg/errors"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/config"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateRecoveringCronJob(cluster slurmv1alpha1.Cluster, spec slurmv1alpha1.LoginNodeSpec) *batchv1.CronJob {
	labels := render.RenderLabels(consts.ComponentTypeLogin, cluster.Name)
	shareProcessNamespace := true
	mungeContainerSpec := slurmv1alpha1.CommonContainerSpec{CPU: resource.MustParse("100m"), Memory: resource.MustParse("100Mi"), Image: spec.MungeContainer.Image, ImagePullPolicy: corev1.PullIfNotPresent}
	recoveringResourceList := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("100Mi")}
	successfulJobsHistoryLimit := int32(0)
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-recovering-slurm",
			Namespace: cluster.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                   config.RecoveringJobConfig.Schedule,
			SuccessfulJobsHistoryLimit: &successfulJobsHistoryLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: corev1.PodSpec{
							ShareProcessNamespace: &shareProcessNamespace,
							InitContainers: []corev1.Container{
								renderMungeContainer(mungeContainerSpec),
							},
							Containers: []corev1.Container{
								{
									Name:            "recovering",
									Image:           config.RecoveringJobConfig.Image,
									ImagePullPolicy: corev1.PullIfNotPresent,
									Resources: corev1.ResourceRequirements{
										Limits:   recoveringResourceList,
										Requests: recoveringResourceList,
									},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "slurm-config", MountPath: "/etc/slurm/"},
										{Name: "munge-run", MountPath: "/run/munge/"},
										{
											Name:      "munge-key",
											MountPath: "/etc/munge/munge.key",
											SubPath:   consts.SecretMungeKeyFileName,
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								renderSlurmConfigVolume(cluster, false),
								renderMungeRunVolume(),
								renderMungeKeyVolume(cluster.Name),
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			},
		},
	}
}

func (r *ClusterReconciler) reconcileRecoveringCronJob(ctx context.Context, cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.LoginNodeSpec) error {
	log := log.FromContext(ctx)

	expectedCronJob := generateRecoveringCronJob(*cluster, spec)
	foundCronJob, err := r.getOrCreateK8sResource(ctx, cluster, expectedCronJob)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile recovering cron job")
	}
	if foundCronJob != nil {
		cronJobObj := foundCronJob.(*batchv1.CronJob)
		if expectedCronJob.Spec.Schedule != cronJobObj.Spec.Schedule {
			patch := client.MergeFrom(cronJobObj.DeepCopy())
			cronJobObj.Spec.Schedule = expectedCronJob.Spec.Schedule
			if err = r.Patch(ctx, cronJobObj, patch); err != nil {
				msg := "Failed to patch recovering cron job"
				log.Error(err, msg)
				return errors.Wrap(err, msg)
			}
		}
	}
	return nil
}

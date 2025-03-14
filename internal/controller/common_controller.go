package controller

import (
	"context"
	"crypto/rand"

	"github.com/pkg/errors"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ClusterReconciler) reconcileSlurmConfigMap(ctx context.Context, cluster *slurmv1alpha1.Cluster) error {
	log := log.FromContext(ctx)
	expectedCm := render.RenderConfigMapSlurmConfigs(cluster)
	foundCm, err := r.getOrCreateK8sResource(ctx, cluster, expectedCm)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile slurm configmap")
	}
	if foundCm != nil {
		objCm := foundCm.(*corev1.ConfigMap)
		patch := client.MergeFrom(objCm.DeepCopy())
		objCm.Data = expectedCm.Data
		if err = r.Patch(ctx, objCm, patch); err != nil {
			msg := "Failed to patch slurm configmap"
			log.Error(err, msg)
			return errors.Wrap(err, msg)
		}
	}

	return nil
}

func generateRandBytes(size int) ([]byte, error) {
	randBytes := make([]byte, size)
	_, err := rand.Read(randBytes)
	if err != nil {
		return nil, err
	}
	return randBytes, nil
}

func (r *ClusterReconciler) reconcileMungeKeySecret(ctx context.Context, cluster *slurmv1alpha1.Cluster) error {
	generateMungeKeySecret := func() (*corev1.Secret, error) {
		mungeKey, err := generateRandBytes(1024)
		if err != nil {
			return nil, errors.New("Failed to generate munge key")
		}
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.BuildSecretName(consts.ComponentTypeMunge, cluster.Name),
				Namespace: cluster.Namespace,
			},
			Data: map[string][]byte{consts.SecretMungeKeyFileName: mungeKey},
		}, nil
	}
	expectedSecret, err := generateMungeKeySecret()
	if err != nil {
		return err
	}
	_, err = r.getOrCreateK8sResource(ctx, cluster, expectedSecret)
	if err != nil {
		return errors.Wrap(err, "Failed to reconcile munge key secret")
	}
	return nil
}

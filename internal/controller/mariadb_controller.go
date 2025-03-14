package controller

import (
	"context"
	"fmt"
	"time"

	mariadbv1alpha1 "github.com/mariadb-operator/mariadb-operator/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func generateMariadbPasswordSecret(cluster *slurmv1alpha1.Cluster) (*corev1.Secret, error) {
	generator, err := password.NewGenerator(&password.GeneratorInput{
		Symbols: "@$^&*()_+-={}|[]<>",
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create password generator")
	}

	rootPassword, err := generator.Generate(16, 4, 2, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate root passowrd")
	}

	slurmPassword, err := generator.Generate(16, 4, 2, false, false)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate slurm passowrd")
	}

	labels := render.RenderLabels(consts.ComponentTypeMariaDb, cluster.Name)

	data := map[string][]byte{
		consts.RootPassword:  []byte(rootPassword),
		consts.SlurmPassword: []byte(slurmPassword),
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Data: data,
	}, nil
}

func generateMariadb(cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.DatabaseSpec) *mariadbv1alpha1.MariaDB {
	labels := render.RenderLabels(consts.ComponentTypeMariaDb, cluster.Name)
	resources := corev1.ResourceList{corev1.ResourceCPU: spec.CPU, corev1.ResourceMemory: spec.Memory}
	return &mariadbv1alpha1.MariaDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildMariaDbName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: mariadbv1alpha1.MariaDBSpec{
			Replicas: 1,
			Storage: mariadbv1alpha1.Storage{
				StorageClassName: spec.Storage.StorageClassName,
				Size:             &spec.Storage.Size,
			},
			Database: ptr.To(string(consts.MariaDbDatabase)),
			Username: ptr.To(string(consts.MariaDbUsername)),
			PasswordSecretKeyRef: &mariadbv1alpha1.GeneratedSecretKeyRef{
				SecretKeySelector: mariadbv1alpha1.SecretKeySelector{
					LocalObjectReference: mariadbv1alpha1.LocalObjectReference{
						Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name),
					},
					Key: consts.SlurmPassword,
				},
			},
			RootPasswordSecretKeyRef: mariadbv1alpha1.GeneratedSecretKeyRef{
				SecretKeySelector: mariadbv1alpha1.SecretKeySelector{
					LocalObjectReference: mariadbv1alpha1.LocalObjectReference{
						Name: utils.BuildSecretName(consts.ComponentTypeMariaDb, cluster.Name),
					},
					Key: consts.RootPassword,
				},
			},
			Service: &mariadbv1alpha1.ServiceTemplate{
				Type: corev1.ServiceTypeClusterIP,
			},
			ContainerTemplate: mariadbv1alpha1.ContainerTemplate{
				Resources: &mariadbv1alpha1.ResourceRequirements{
					Limits:   resources,
					Requests: resources,
				},
			},
			MyCnf: ptr.To(consts.MariaDbDefaultMyCnf),
		},
	}
}

func generateMariadbGrant(cluster *slurmv1alpha1.Cluster) *mariadbv1alpha1.Grant {
	return &mariadbv1alpha1.Grant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildMariaDbName(cluster.Name),
			Namespace: cluster.Namespace,
			Labels:    render.RenderLabels(consts.ComponentTypeMariaDb, cluster.Name),
		},
		Spec: mariadbv1alpha1.GrantSpec{
			MariaDBRef: mariadbv1alpha1.MariaDBRef{
				WaitForIt: true,
				ObjectReference: mariadbv1alpha1.ObjectReference{
					Name:      utils.BuildMariaDbName(cluster.Name),
					Namespace: cluster.Namespace,
				},
			},
			Privileges: []string{
				"ALL PRIVILEGES",
			},
			Database:    consts.MariaDbDatabase,
			Username:    consts.MariaDbUsername,
			Table:       consts.MariaDbTable,
			GrantOption: true,
			Host:        ptr.To("%"),
			SQLTemplate: mariadbv1alpha1.SQLTemplate{
				RequeueInterval: &metav1.Duration{
					Duration: 30,
				},
				RetryInterval: &metav1.Duration{
					Duration: 5,
				},
			},
		},
	}
}

func (r *ClusterReconciler) reconcileDatabase(ctx context.Context, cluster *slurmv1alpha1.Cluster, spec slurmv1alpha1.DatabaseSpec) (*ctrl.Result, error) {
	log := log.FromContext(ctx)
	expectedsecret, err := generateMariadbPasswordSecret(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate mariadb password secret")
	}
	_, err = r.getOrCreateK8sResource(ctx, cluster, expectedsecret)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to reconcile mariadb password secret")
	}

	expectedGrant := generateMariadbGrant(cluster)
	_, err = r.getOrCreateK8sResource(ctx, cluster, expectedGrant)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to reconcile mariadb grant")
	}

	expectedMariadb := generateMariadb(cluster, spec)
	mariadb, err := r.getOrCreateK8sResource(ctx, cluster, expectedMariadb)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to reconcile mariadb")
	}
	result, err := r.setControllerReferenceForMariadbPVC(ctx, cluster)
	if err != nil || result != nil {
		return result, err
	}
	ready := false
	if mariadb != nil {
		mariadbObj := mariadb.(*mariadbv1alpha1.MariaDB)
		for _, c := range mariadbObj.Status.Conditions {
			if c.Type == consts.ConditionTypeReady && c.Status == consts.ConditionStatusTrue {
				if err = r.setStatusCondition(
					ctx, cluster,
					&metav1.Condition{
						Type:    consts.DatabaseConditionType,
						Status:  metav1.ConditionTrue,
						Reason:  "Ready",
						Message: "Mariadb is ready"}); err != nil {
					return nil, err
				}
				ready = true
				break
			}
		}
	}
	if !ready {
		log.Info("Mariadb not ready")
		if err := r.setStatusCondition(
			ctx, cluster,
			&metav1.Condition{
				Type:    consts.DatabaseConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  "NotReady",
				Message: "Mariadb is not ready"}); err != nil {
			return nil, err
		}
		return &ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}

	return nil, nil
}

func (r *ClusterReconciler) setControllerReferenceForMariadbPVC(ctx context.Context, cluster *slurmv1alpha1.Cluster) (*ctrl.Result, error) {
	log := log.FromContext(ctx)
	pvcList := &corev1.PersistentVolumeClaimList{}
	err := r.List(
		ctx, pvcList, client.InNamespace(cluster.Namespace),
		client.MatchingLabels{
			"app.kubernetes.io/instance": utils.BuildMariaDbName(cluster.Name),
			"app.kubernetes.io/name":     "mariadb",
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list mariadb persistentvolumeclaims")
	}
	if len(pvcList.Items) == 0 {
		return &ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, nil
	}
	for _, pvc := range pvcList.Items {
		patch := client.MergeFrom(pvc.DeepCopy())
		if err = ctrl.SetControllerReference(cluster, &pvc, r.Scheme); err != nil {
			msg := fmt.Sprintf("Failed to set controller reference for mariadb PersistentVolumeClaim: %s", pvc.Name)
			log.Error(err, msg)
			return nil, errors.Wrap(err, msg)
		}
		if err = r.Patch(ctx, &pvc, patch); err != nil {
			msg := fmt.Sprintf("Failed to patch mariadb PersistentVolumeClaim: %s", pvc.Name)
			log.Error(err, msg)
			return nil, errors.Wrap(err, msg)
		}
	}
	return nil, nil
}

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	mariadbv1alpha1 "github.com/mariadb-operator/mariadb-operator/api/v1alpha1"
	"github.com/pkg/errors"
	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
	"github.com/troila-klcloud/slurm-operator/internal/consts"
	"github.com/troila-klcloud/slurm-operator/internal/render"
	"github.com/troila-klcloud/slurm-operator/internal/utils"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

/*
We generally want to ignore (not requeue) NotFound errors, since we'll get a
reconciliation request once the object exists, and requeuing in the meantime
won't help.
*/
func ignoreNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// +kubebuilder:rbac:groups=slurm.kunluncloud.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=slurm.kunluncloud.com,resources=clusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=slurm.kunluncloud.com,resources=clusters/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;patch;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;patch;update;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;patch;delete
// +kubebuilder:rbac:groups=k8s.mariadb.com,resources=mariadbs,verbs=get;list;watch;create;patch;delete
// +kubebuilder:rbac:groups=k8s.mariadb.com,resources=grants,verbs=get;list;watch;create;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Receive reconciliation request")

	event := &corev1.Event{}
	getEventErr := r.Get(ctx, req.NamespacedName, event)
	if getEventErr == nil {
		log.Info("Found event for Cluster. Re-emitting...")
		involvedObj, err := r.eventInvolvedObject(ctx, event)
		if err != nil {
			return ctrl.Result{}, ignoreNotFound(err)
		}
		involvedCluster := &slurmv1alpha1.Cluster{}
		clusterFound := false
		if event.InvolvedObject.Kind == consts.KindPod {
			if err := r.Get(ctx, types.NamespacedName{Namespace: involvedObj.GetNamespace(), Name: involvedObj.GetLabels()[consts.LabelInstanceKey]}, involvedCluster); err != nil {
				log.Info("unable to fetch Slurm Cluster by looking at event")
				return ctrl.Result{}, ignoreNotFound(err)
			}
			clusterFound = true
		} else {
			for _, owner := range involvedObj.GetOwnerReferences() {
				if owner.Kind == "Cluster" && owner.APIVersion == slurmv1alpha1.GroupVersion.String() {
					if err := r.Get(ctx, types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: owner.Name}, involvedCluster); err != nil {
						log.Info("unable to fetch Slurm Cluster by looking at event")
						return ctrl.Result{}, ignoreNotFound(err)
					}
					clusterFound = true
				}
			}
		}

		if clusterFound {
			// re-emit the event in the Slurm Cluster CR
			log.Info("Emitting Slurm Event.", "Event", event)
			r.EventRecorder.Eventf(involvedCluster, event.Type, event.Reason,
				"Reissued from %s/%s: %s", strings.ToLower(event.InvolvedObject.Kind), event.InvolvedObject.Name, event.Message)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, errors.Errorf("invalid event: %s/%s, involved object: %s/%s", event.Type, event.Name, event.InvolvedObject.Kind, event.InvolvedObject.Name)
	}
	if !apierrors.IsNotFound(getEventErr) {
		return ctrl.Result{}, getEventErr
	}

	cluster := &slurmv1alpha1.Cluster{}
	err := r.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("cluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get cluster")
		return ctrl.Result{}, errors.Wrap(err, "Failed to get cluster")
	}

	if err = r.reconcileSlurmConfigMap(ctx, cluster); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconfile slurm configmap")
	}

	if err = r.reconcileMungeKeySecret(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}

	result, err := r.reconcileDatabase(ctx, cluster, cluster.Spec.Database)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconcile database")
	}
	if result != nil {
		return *result, nil
	}

	result, err = r.reconcileAccountingNode(ctx, cluster, cluster.Spec.AccountingNode)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconcile accounting node")
	}
	if result != nil {
		return *result, nil
	}

	if err = r.reconcileLoginNode(ctx, cluster, cluster.Spec.LoginNode); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconcile login node")
	}

	if err = r.reconcileControllerNode(ctx, cluster, cluster.Spec.CtrlNode); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconcile controller node")
	}

	if err = r.reconcileComputingNodeSets(ctx, cluster, cluster.Spec.ComputingNodeSets); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconcile computing node set")
	}

	if err = r.reconcileWebNode(ctx, cluster, cluster.Spec.WebNode); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Failed to reconcile web node set")
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) setStatusCondition(ctx context.Context, cluster *slurmv1alpha1.Cluster, condition *metav1.Condition) error {
	log := log.FromContext(ctx)
	patch := client.MergeFrom(cluster.DeepCopy())
	cluster.Status.SetCondition(*condition)
	if err := r.Status().Patch(ctx, cluster, patch); err != nil {
		msg := "Failed to set status condition"
		log.Error(err, msg)
		return errors.Wrap(err, msg)
	}
	return nil
}

func (r *ClusterReconciler) getOrCreateK8sResource(ctx context.Context, cluster *slurmv1alpha1.Cluster, obj client.Object) (client.Object, error) {
	log := log.FromContext(ctx)
	var found client.Object
	switch obj.(type) {
	case *corev1.ConfigMap:
		found = &corev1.ConfigMap{}
	case *corev1.Secret:
		found = &corev1.Secret{}
	case *corev1.Service:
		found = &corev1.Service{}
	case *corev1.PersistentVolumeClaim:
		found = &corev1.PersistentVolumeClaim{}
	case *appsv1.StatefulSet:
		found = &appsv1.StatefulSet{}
	case *appsv1.Deployment:
		found = &appsv1.Deployment{}
	case *mariadbv1alpha1.MariaDB:
		found = &mariadbv1alpha1.MariaDB{}
	case *mariadbv1alpha1.Grant:
		found = &mariadbv1alpha1.Grant{}
	default:
		return nil, errors.New("unsuported resource kind")
	}

	gvk, err := apiutil.GVKForObject(found, r.Scheme)
	if err != nil {
		return nil, err
	}

	err = r.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, found)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating a new kubernetes resource %s: %s", gvk.String(), obj.GetName()))
		if err = ctrl.SetControllerReference(cluster, obj, r.Scheme); err != nil {
			msg := fmt.Sprintf("Failed to set controller reference for %s: %s", gvk.String(), obj.GetName())
			log.Error(err, msg)
			return nil, errors.Wrap(err, msg)
		}
		if err = r.Create(ctx, obj); err != nil {
			msg := fmt.Sprintf("Failed to create %s: %s", gvk.String(), obj.GetName())
			log.Error(err, msg)
			return nil, errors.Wrap(err, msg)
		}
		return nil, nil
	} else if err != nil {
		msg := fmt.Sprintf("Failed to get %s: %s", gvk.String(), obj.GetName())
		log.Error(err, msg)
		return nil, errors.Wrap(err, msg)
	}
	return found, nil
}

func (r *ClusterReconciler) eventInvolvedObject(ctx context.Context, event *corev1.Event) (client.Object, error) {
	log := log.FromContext(ctx)
	var InvolvedObj client.Object
	switch event.InvolvedObject.Kind {
	case consts.KindPersistentVolumeClaim:
		InvolvedObj = &corev1.PersistentVolumeClaim{}
	case consts.KindDeployment:
		InvolvedObj = &appsv1.Deployment{}
	case consts.KindStatefulset:
		InvolvedObj = &appsv1.StatefulSet{}
	case consts.KindService:
		InvolvedObj = &corev1.Service{}
	case consts.KindMariaDB:
		InvolvedObj = &mariadbv1alpha1.MariaDB{}
	case consts.KindGrant:
		InvolvedObj = &mariadbv1alpha1.Grant{}
	case consts.KindPod:
		InvolvedObj = &corev1.Pod{}
	default:
		return nil, nil
	}
	if err := r.Get(ctx, types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, InvolvedObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		log.Error(err, "Failed to get involved object")
		return nil, err
	}
	return InvolvedObj, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {

	eventPredicateFuncs := func(r *ClusterReconciler) predicate.Funcs {
		predicateFuncs := predicate.NewPredicateFuncs(func(object client.Object) bool {
			event := object.(*corev1.Event)
			if event.Type == "Normal" {
				return false
			}
			involvedObj, err := r.eventInvolvedObject(context.Background(), event)
			if err != nil || involvedObj == nil {
				return false
			}

			if event.InvolvedObject.Kind == consts.KindPod {
				selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: map[string]string{
					consts.LabelNameKey:      consts.LabelNameValue,
					consts.LabelPartOfKey:    consts.LabelPartOfValue,
					consts.LabelManagedByKey: consts.LabelManagedByValue,
				}})
				if err != nil {
					return false
				}
				return selector.Matches(labels.Set(involvedObj.GetLabels()))
			} else {
				for _, owner := range involvedObj.GetOwnerReferences() {
					if owner.Kind == "Cluster" && owner.APIVersion == slurmv1alpha1.GroupVersion.String() {
						return true
					}
				}
			}
			return false
		})
		predicateFuncs.DeleteFunc = func(e event.DeleteEvent) bool {
			return false
		}
		return predicateFuncs
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&slurmv1alpha1.Cluster{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Owns(&mariadbv1alpha1.MariaDB{}).
		Owns(&mariadbv1alpha1.Grant{}).
		Watches(
			&corev1.Event{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(eventPredicateFuncs(r)),
		).
		Named("cluster").
		Complete(r)
}

func generateHeadlessService(component consts.ComponentType, cluster slurmv1alpha1.Cluster, port int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.BuildServiceName(component, cluster.Name),
			Namespace: cluster.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			Selector:  render.RenderMatchLabels(component, cluster.Name),
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{Protocol: corev1.ProtocolTCP, Port: port, TargetPort: intstr.FromInt32(port)},
			},
		},
	}
}

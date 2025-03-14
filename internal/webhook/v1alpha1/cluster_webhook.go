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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	slurmv1alpha1 "github.com/troila-klcloud/slurm-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var clusterlog = logf.Log.WithName("cluster-resource")

// SetupClusterWebhookWithManager registers the webhook for Cluster in the manager.
func SetupClusterWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&slurmv1alpha1.Cluster{}).
		WithValidator(&ClusterCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-slurm-kunluncloud-com-v1alpha1-cluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=slurm.kunluncloud.com,resources=clusters,verbs=create;update,versions=v1alpha1,name=vcluster-v1alpha1.kb.io,admissionReviewVersions=v1

// ClusterCustomValidator struct is responsible for validating the Cluster resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ClusterCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ClusterCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Cluster.
func (v *ClusterCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster, ok := obj.(*slurmv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("expected a Cluster object but got %T", obj)
	}
	clusterlog.Info("Validation for Cluster upon creation", "name", cluster.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Cluster.
func (v *ClusterCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cluster, ok := newObj.(*slurmv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("expected a Cluster object for the newObj but got %T", newObj)
	}
	clusterlog.Info("Validation for Cluster upon update", "name", cluster.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Cluster.
func (v *ClusterCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cluster, ok := obj.(*slurmv1alpha1.Cluster)
	if !ok {
		return nil, fmt.Errorf("expected a Cluster object but got %T", obj)
	}
	clusterlog.Info("Validation for Cluster upon deletion", "name", cluster.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

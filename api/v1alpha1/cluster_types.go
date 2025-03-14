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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=NodePort;LoadBalancer
type ServiceType string

const (
	NodePort     ServiceType = "NodePort"
	LoadBalancer ServiceType = "LoadBalancer"
)

type CommonContainerSpec struct {
	// Specifies the number of CPUs for each node instance
	CPU resource.Quantity `json:"cpu"`

	// Specifies the amount of memory for each node instance
	Memory resource.Quantity `json:"memory"`

	// +kubebuilder:validation:Required
	//
	// Specifies the container image
	Image string `json:"image"`

	// Defines the image pull policy
	//
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="IfNotPresent"
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

type LoginNodeSpec struct {
	// +kubebuilder:validation:Minimum=1
	//
	// Defines the number of login node instances
	Size int32 `json:"size"`

	// defines ssshd containers
	SshdContainer CommonContainerSpec `json:"sshdContainer"`

	// defines sssd containers
	SssdContainer CommonContainerSpec `json:"sssdContainer"`

	// defines munge containers
	MungeContainer CommonContainerSpec `json:"mungeContainer"`

	// Specifies how to expose SSH endpoint
	// Valid values are:
	// - "NodePort" (default): expose SSH endpoint by physical node port
	// - "LoadBalancer": expose SSH endpoint by an external IP
	ServiceType string `json:"serviceType"`
}

type WebNodeSpec struct {
	// +kubebuilder:validation:Minimum=1
	//
	// Defines the number of login node instances
	Size int32 `json:"size"`

	// defines web interface container
	WebContainer CommonContainerSpec `json:"webContainer"`

	// defines sssd container
	SssdContainer CommonContainerSpec `json:"sssdContainer"`

	// defines munge container
	MungeContainer CommonContainerSpec `json:"mungeContainer"`

	// defines init sql container
	// +kubebuilder:validation:Optional
	InitSqlContainer CommonContainerSpec `json:"initSqlContainer,omitempty"`

	// Defines the expose port
	Port int32 `json:"port"`
}

type CtrlNodeSpec struct {
	// +kubebuilder:validation:Minimum=1
	//
	// Defines the number of login node instances
	Size int32 `json:"size"`

	// defines slurmctld containers
	SlurmCtldContainer CommonContainerSpec `json:"slurmctldContainer"`

	// defines sssd containers
	SssdContainer CommonContainerSpec `json:"sssdContainer"`

	// defines munge containers
	MungeContainer CommonContainerSpec `json:"mungeContainer"`

	// Specifies the storageclass name for slurm spool dir, the storage must support ReadWriteMany
	SpoolStorageClassName string `json:"spoolStroageClassName"`
}

type AccountingNodeSpec struct {
	// +kubebuilder:validation:Minimum=1
	//
	// Defines the number of login node instances
	Size int32 `json:"size"`

	// defines slurmdbd containers
	SlurmdbdContainer CommonContainerSpec `json:"slurmdbdContainer"`

	// defines munge containers
	MungeContainer CommonContainerSpec `json:"mungeContainer"`
}

type DBStroage struct {
	// Specifies the storageClass name
	StorageClassName string `json:"storageClassName"`

	// Specifies the size of the stroage
	Size resource.Quantity `json:"size"`
}

type DatabaseSpec struct {
	// Specifies the number of CPUs for each mariadb instance
	CPU resource.Quantity `json:"cpu"`

	// Specifies the amount of memory for each mariadb instance
	Memory resource.Quantity `json:"memory"`

	// Defines the storage for mariadb cluster
	Storage DBStroage `json:"storage"`
}

type ComputingNodeSetSpec struct {
	// +kubebuilder:validation:Minimum=1
	//
	// Defines the number of login node instances
	Size int32 `json:"size"`

	// Specifies the node of the partition which the computing node set belong to
	PartitionName string `json:"partitionName"`

	// defines slurmd containers
	SlurmdContainer CommonContainerSpec `json:"slurmdContainer"`

	// defines sssd containers
	SssdContainer CommonContainerSpec `json:"sssdContainer"`

	// defines munge containers
	MungeContainer CommonContainerSpec `json:"mungeContainer"`

	// Which type of GPU which this computing node set to use
	// +optional
	GPU string `json:"gpu,omitempty"`
}

// ClusterSpec defines the desired state of Cluster.
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Specifies the persistent volume to storage user data
	PersistentVolumeClaimName string `json:"persistentVolumeClaimName"`

	// Defines the login node
	LoginNode LoginNodeSpec `json:"loginNode"`

	// Defines the controller node
	CtrlNode CtrlNodeSpec `json:"controllerNode"`

	// Defines the slurmdbd node
	AccountingNode AccountingNodeSpec `json:"accountingNode"`

	// Defines the web node
	WebNode WebNodeSpec `json:"webNode"`

	// Defines mariadb cluster
	Database DatabaseSpec `json:"database"`

	// Defines the computing node sets
	ComputingNodeSets []ComputingNodeSetSpec `json:"computingNodeSets"`
}

type PodStatus struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Phrase corev1.PodPhase `json:"phrase"`
	// +kubebuilder:validation:Optional
	Conditions []corev1.PodCondition `json:"conditions,omitempty"`
}

// ClusterStatus defines the observed state of Cluster.
type ClusterStatus struct {
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +kubebuilder:validation:Optional
	PodStatuses []PodStatus `json:"podStatuses,omitempty"`
	// +kubebuilder:validation:Optional
	NodePort int32 `json:"nodePort,omitempty"`
}

func (s *ClusterStatus) SetCondition(condition metav1.Condition) {
	if s.Conditions == nil {
		s.Conditions = make([]metav1.Condition, 0)
	}
	meta.SetStatusCondition(&s.Conditions, condition)
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cluster is the Schema for the clusters API.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}

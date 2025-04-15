package render

import (
	"github.com/troila-klcloud/slurm-operator/internal/consts"
)

// RenderLabels prepends to the provided labels, the default set of labels used for all resources.
// These labels are recommended by k8s https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func RenderLabels(componentType consts.ComponentType, clusterName string) map[string]string {
	return map[string]string{
		consts.LabelNameKey:      consts.LabelNameValue,
		consts.LabelInstanceKey:  clusterName,
		consts.LabelComponentKey: componentType.String(),
		consts.LabelPartOfKey:    consts.LabelPartOfValue,
		consts.LabelManagedByKey: consts.LabelManagedByValue,
	}
}

// RenderMatchLabels prepends to the provided labels, the default set of match-labels used for all resources.
func RenderMatchLabels(componentType consts.ComponentType, clusterName string) map[string]string {
	return map[string]string{
		consts.LabelNameKey:      consts.LabelNameValue,
		consts.LabelInstanceKey:  clusterName,
		consts.LabelComponentKey: componentType.String(),
	}
}

func RenderComputingNodeSetLables(clusterName string, partition string) map[string]string {
	return map[string]string{
		consts.LabelNameKey:                 consts.LabelNameValue,
		consts.LabelInstanceKey:             clusterName,
		consts.LabelComponentKey:            consts.ComponentTypeComputing.String(),
		consts.LabelComputingSetPartitonKey: partition,
		consts.LabelPartOfKey:               consts.LabelPartOfValue,
		consts.LabelManagedByKey:            consts.LabelManagedByValue,
	}
}

func RenderComputingNodeSetMatchLabels(clusterName string) map[string]string {
	return map[string]string{
		consts.LabelNameKey:      consts.LabelNameValue,
		consts.LabelInstanceKey:  clusterName,
		consts.LabelComponentKey: consts.ComponentTypeComputing.String(),
		consts.LabelPartOfKey:    consts.LabelPartOfValue,
		consts.LabelManagedByKey: consts.LabelManagedByValue,
	}
}

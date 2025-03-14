package utils

import (
	"strings"

	"github.com/troila-klcloud/slurm-operator/internal/consts"
)

func StringToComponent(s ...string) consts.ComponentType {
	return consts.BaseComponentType{Value: strings.Join(s, "-")}
}

package render

import "github.com/troila-klcloud/slurm-operator/internal/consts"

func RenderWaveAnnotations() map[string]string {
	return map[string]string{
		consts.WaveOnConfigChangeKey: "true",
	}
}

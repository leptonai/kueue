package apis

import (
	"encoding/json"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

const (
	labelCanPreempt     = "kueue.lepton.ai/can-preempt"
	labelCanBePreempted = "kueue.lepton.ai/can-be-preempted"

	annotationPreemptionStrategy = "kueue.lepton.ai/preemption-strategy"
)

func CanPreempt(wl *kueue.Workload) bool {
	return wl.Labels[labelCanPreempt] == "true"
}

func CanBePreempted(wl *kueue.Workload) bool {
	return wl.Labels[labelCanBePreempted] == "true"
}

type PreemptionStrategy struct {
	CrossNamespaces      bool   `json:"crossNamespaces,omitempty"`
	MaxPriorityThreshold *int32 `json:"maxPriorityThreshold,omitempty"`
}

func GetQueuePreemptionStrategy(annotations map[string]string) PreemptionStrategy {
	val, ok := annotations[annotationPreemptionStrategy]
	if !ok {
		return PreemptionStrategy{}
	}

	var p PreemptionStrategy
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		return PreemptionStrategy{}
	}
	return p
}

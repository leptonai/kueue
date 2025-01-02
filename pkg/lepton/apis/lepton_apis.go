package apis

import (
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

const (
	labelCanPreempt     = "kueue.lepton.ai/can-preempt"
	labelCanBePreempted = "kueue.lepton.ai/can-be-preempted"
)

func CanPreempt(wl *kueue.Workload) bool {
	return wl.Labels[labelCanPreempt] == "true"
}

func CanBePreempted(wl *kueue.Workload) bool {
	return wl.Labels[labelCanBePreempted] == "true"
}

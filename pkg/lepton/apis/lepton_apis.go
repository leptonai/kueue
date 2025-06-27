package apis

import (
	"encoding/json"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/workload"
)

const (
	labelCanPreempt     = "kueue.lepton.ai/can-preempt"
	labelCanBePreempted = "kueue.lepton.ai/can-be-preempted"

	annotationPreemptionStrategy = "kueue.lepton.ai/preemption-strategy"

	labelNodeReservationRequestBinding                = "node-reservation.lepton.ai/request-binding"
	annotationCanBePreemptedByNodeReservationRequests = "node-reservation.lepton.ai/can-be-preempted-by"

	labelScheduleFailed = "kueue.lepton.ai/schedule-failed"
)

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

func ComparePreemptOrder(i, j, wl *kueue.Workload) int32 {
	if CanPreemptByNRRs(wl, i) {
		return -1
	} else if CanPreemptByNRRs(wl, j) {
		return 1
	}
	if CanPreemptByScheduleFailed(wl, i) {
		return -1
	} else if CanPreemptByScheduleFailed(wl, j) {
		return 1
	}
	return 0
}

func CanPreemptByNRRs(wl, target *kueue.Workload) bool {
	nrrName := wl.Labels[labelNodeReservationRequestBinding]
	if nrrName == "" {
		return false
	}
	if target.Labels[labelCanBePreempted] != "true" {
		return false
	}
	if val := target.Annotations[annotationCanBePreemptedByNodeReservationRequests]; val != "" {
		return slices.Contains(strings.Split(val, ","), nrrName)
	}
	return false
}

func CanPreemptByScheduleFailed(wl, target *kueue.Workload) bool {
	if target.Labels[labelScheduleFailed] != "true" {
		return false
	}
	if wl.Labels[labelNodeReservationRequestBinding] != "" {
		return true
	}
	return wl.Labels[labelCanPreempt] == "true" && priority.Priority(wl) > priority.Priority(target)
}

func CanPreempt(wl *kueue.Workload) bool {
	return wl.Labels[labelCanPreempt] == "true" || wl.Labels[labelNodeReservationRequestBinding] != ""
}

func CanBeCandidate(preemptionStrategy PreemptionStrategy, selfWl *kueue.Workload, candidateWl *workload.Info, frsNeedPreemption sets.Set[resources.FlavorResource]) bool {
	if !workloadUsesResources(candidateWl, frsNeedPreemption) {
		return false
	}

	// if the reservation request matches, can always be candidate
	if CanPreemptByNRRs(selfWl, candidateWl.Obj) {
		return true
	}
	if CanPreemptByScheduleFailed(selfWl, candidateWl.Obj) {
		return true
	}

	if selfWl.Labels[labelCanPreempt] != "true" || candidateWl.Obj.Labels[labelCanBePreempted] != "true" {
		return false
	}

	selfPriority := priority.Priority(selfWl)
	candidatePriority := priority.Priority(candidateWl.Obj)
	if candidatePriority >= selfPriority {
		return false
	}
	// if the candidate is bound to a reservation, currently we only allow it to be preempted by workload with the same reservation
	// TODO: maybe if the candidate deploys on nodes that are all not reserved, it can be preempted by others
	if nrrName := candidateWl.Obj.Labels[labelNodeReservationRequestBinding]; nrrName != "" {
		if nrrName != selfWl.Labels[labelNodeReservationRequestBinding] {
			return false
		}
	}
	if !preemptionStrategy.CrossNamespaces && selfWl.Namespace != candidateWl.Obj.Namespace {
		return false
	}
	if preemptionStrategy.MaxPriorityThreshold != nil && candidatePriority > *preemptionStrategy.MaxPriorityThreshold {
		return false
	}
	return true
}

func workloadUsesResources(wl *workload.Info, frsNeedPreemption sets.Set[resources.FlavorResource]) bool {
	for _, ps := range wl.TotalRequests {
		for res, flv := range ps.Flavors {
			if frsNeedPreemption.Has(resources.FlavorResource{Flavor: flv, Resource: res}) {
				return true
			}
		}
	}
	return false
}

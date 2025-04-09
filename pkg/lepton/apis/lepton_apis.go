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

	labelPreAllocationID  = "kueue.lepton.ai/pre-allocation-id"
	labelPreAllocationFor = "kueue.lepton.ai/pre-allocation-for"
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

func ComparePreemptOrder(i, j, by *kueue.Workload) int32 {
	if preAllocationID := by.Labels[labelPreAllocationID]; preAllocationID != "" {
		if i.Labels[labelPreAllocationFor] == preAllocationID {
			return -1
		}
		if j.Labels[labelPreAllocationFor] == preAllocationID {
			return -1
		}
	}
	if nrrName := by.Labels[labelNodeReservationRequestBinding]; nrrName != "" {
		if canBePreemptedByNRRs(i, nrrName) {
			return -1
		}
		if canBePreemptedByNRRs(j, nrrName) {
			return 1
		}
	}
	return 0
}

func canBePreemptedByNRRs(wl *kueue.Workload, nrrName string) bool {
	if val := wl.Annotations[annotationCanBePreemptedByNodeReservationRequests]; val != "" {
		return slices.Contains(strings.Split(val, ","), nrrName)
	}
	return false
}

func CanPreemptByNRRs(wl, target *kueue.Workload) bool {
	nrrName := wl.Labels[labelNodeReservationRequestBinding]
	if nrrName == "" {
		return false
	}
	return canBePreemptedByNRRs(target, nrrName)
}

func CanPreempt(wl *kueue.Workload) bool {
	return wl.Labels[labelCanPreempt] == "true" ||
		wl.Labels[labelNodeReservationRequestBinding] != "" ||
		wl.Labels[labelPreAllocationID] != ""
}

func CanBeCandidate(preemptionStrategy PreemptionStrategy, selfWl *kueue.Workload, candidateWl *workload.Info, frsNeedPreemption sets.Set[resources.FlavorResource]) bool {
	if !workloadUsesResources(candidateWl, frsNeedPreemption) {
		return false
	}

	// if the workload matches pre-allocation for self workload
	if preAllocationID := selfWl.Labels[labelPreAllocationID]; preAllocationID != "" && preAllocationID == candidateWl.Obj.Labels[labelPreAllocationFor] {
		return true
	}

	// if the reservation request matches, can always be candidate
	if nrrName := selfWl.Labels[labelNodeReservationRequestBinding]; nrrName != "" && canBePreemptedByNRRs(candidateWl.Obj, nrrName) {
		return true
	}

	if selfWl.Labels[labelCanPreempt] != "true" {
		return false
	}

	selfPriority := priority.Priority(selfWl)
	candidatePriority := priority.Priority(candidateWl.Obj)
	if candidateWl.Obj.Labels[labelCanBePreempted] != "true" {
		return false
	}
	if candidatePriority >= selfPriority {
		return false
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

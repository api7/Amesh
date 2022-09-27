package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionSync        = "Sync"
	ConditionSyncSuccess = "Sync Success"

	ResourceReconciled = "Reconciled"
)

// VerifyGeneration verify generation to decide whether to update status
func VerifyGeneration(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition != nil && existingCondition.ObservedGeneration > newCondition.ObservedGeneration {
		return false
	}
	return true
}

// Copyright 2022 The Amesh Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

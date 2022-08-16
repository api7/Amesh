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

package controlplane

// ControlPlane represents the control plane in e2e test cases.
type ControlPlane interface {
	// Type returns the control plane type.
	Type() string
	// Namespace fetches the deployed namespace of control plane components.
	Namespace() string
	// InjectNamespace marks the target namespace as injectable. Pod in this
	// namespace will be injected by control plane.
	InjectNamespace(string) error
	// Deploy deploys the control plane.
	Deploy() error
	// Uninstall uninstalls the control plane.
	Uninstall() error
	// Addr returns the address to communicate with the control plane for fetching
	// configuration changes.
	Addr() string
}

/*
Copyright 2022 Spectrocloud Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package waitforready implements the wait for ready action
package provisiondns

import (
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	"time"
)

// Action implements an action for waiting for the cluster to be ready
type Action struct {
	waitTime time.Duration
}

// NewAction returns a new action for waiting for the cluster to be ready
func NewAction(waitTime time.Duration) actions.Action {
	return &Action{
		waitTime: waitTime,
	}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("SpectroCloud preflight check 📡")
	defer ctx.Status.End(false)
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}
	// get a control plane node to use to check cluster status
	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	node := controlPlanes[0]
	if err := a.recreateDnsResources(node); err != nil {
		return err
	}
	ctx.Status.End(true)
	return nil
}

func (a *Action) recreateDnsResources(node nodes.Node) error {
	//deleting kube-root-ca.crt in kube-system namespace
	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "cm", "kube-root-ca.crt", "-n", "kube-system",
	).Run()

	//deleting kube-root-ca.crt in kube-public namespace
	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "cm", "kube-root-ca.crt", "-n", "kube-public",
	).Run()

	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"delete", "pods", "-l", "k8s-app=kube-proxy", "-n", "kube-system",
	).Run()

	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"rollout", "restart", "deployment/coredns", "-n", "kube-system",
	).Run()

	_ = node.Command(
		"kubectl", "--kubeconfig=/etc/kubernetes/admin.conf",
		"wait", "--for=condition=available", "--timeout=300s", "deployment/coredns", "-n", "kube-system",
	).Run()

	return nil
}
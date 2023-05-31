/*
Copyright 2023.

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

package controller

import (
	"context"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	vpngwv1 "github.com/bobz965/kube-ovn-operator/api/v1"
)

// VpnGwReconciler reconciles a VpnGw object
type VpnGwReconciler struct {
	client.Client
	Logger    log.Logger
	Scheme    *runtime.Scheme
	Namespace string
	Handler   func(log.Logger, string, string) SyncState
	Reload    chan event.GenericEvent
}

//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=vpngws,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=vpngws/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=vpngws/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *VpnGwReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	level.Info(r.Logger).Log("controller", "VpnGwReconciler", "start reconcile", req.NamespacedName.String())
	defer level.Info(r.Logger).Log("controller", "VpnGwReconciler", "end reconcile", req.NamespacedName.String())
	updates.Inc()

	var gw vpngwv1.VpnGw
	if err := r.Get(ctx, req.NamespacedName, &gw); err != nil {
		level.Error(r.Logger).Log("controller", "VpnGwReconciler", "message", "failed to get vpn gw", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	res := r.Handler(r.Logger, r.Namespace, gw.Name)
	switch res {
	case SyncStateError:
		updateErrors.Inc()
		level.Info(r.Logger).Log("controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw), "event", "failed to handle vpn gw")
		return ctrl.Result{}, errRetry
	case SyncStateReprocessAll:
		// vpn gw may has dependency on other resources, so we need to reprocess all
		// such as add|remove a ipsec container
		// such as add|remove a ssl container
		level.Info(r.Logger).Log("controller", "VpnGwReconciler", "event", "force vpn gw reload")
		// r.forceReload()
		return ctrl.Result{}, nil
	case SyncStateErrorNoRetry:
		updateErrors.Inc()
		level.Error(r.Logger).Log("controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw), "event", "failed to handle vpn gw")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VpnGwReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vpngwv1.VpnGw{}).
		Complete(r)
}

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
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	vpngwv1 "github.com/bobz965/kube-ovn-operator/api/v1"
	"github.com/go-logr/logr"
)

const (
	VpnGwLabel = "vpn-gw"
)

// IpsecConnReconciler reconciles a IpsecConn object
type IpsecConnReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
	Handler   func(ipsecConn *vpngwv1.IpsecConn, req ctrl.Request) SyncState
	Reload    chan event.GenericEvent
}

func (r *IpsecConnReconciler) validateIpsecConnection(ipsecConn *vpngwv1.IpsecConn, namespacedName string) error {
	if ipsecConn.Spec.VpnGw == "" {
		err := fmt.Errorf("ipsecConn vpn gw is required")
		r.Log.Error(err, "should set vpn gw")
		return err
	}
	if ipsecConn.Status.VpnGw != "" && ipsecConn.Spec.VpnGw != ipsecConn.Status.VpnGw {
		err := fmt.Errorf("ipsecConn vpn gw can not be changed")
		r.Log.Error(err, "ipsecConn should not change vpn gw")
		return err
	}

	if ipsecConn.Spec.RemotePublicIp == "" {
		err := fmt.Errorf("ipsecConn remote public ip is required")
		r.Log.Error(err, "should set remote public ip")
		return err
	}

	if ipsecConn.Spec.RemotePrivateCidrs == "" {
		err := fmt.Errorf("ipsecConn remote private cidrs is required")
		r.Log.Error(err, "should set remote private cidrs")
		return err
	}

	if ipsecConn.Spec.LocalPrivateCidrs == "" {
		err := fmt.Errorf("ipsecConn local private cidrs is required")
		r.Log.Error(err, "should set local private cidrs")
		return err
	}

	return nil
}

func (r *IpsecConnReconciler) isChanged(ipsecConn *vpngwv1.IpsecConn) bool {
	changed := false
	if ipsecConn.Status.VpnGw == "" && ipsecConn.Spec.VpnGw != "" {
		ipsecConn.Status.VpnGw = ipsecConn.Spec.VpnGw
		changed = true
	}
	if ipsecConn.Status.RemotePublicIp == "" && ipsecConn.Spec.RemotePublicIp != "" {
		ipsecConn.Status.RemotePublicIp = ipsecConn.Spec.RemotePublicIp
		changed = true
	}
	if ipsecConn.Status.RemotePrivateCidrs == "" && ipsecConn.Spec.RemotePrivateCidrs != "" {
		ipsecConn.Status.RemotePrivateCidrs = ipsecConn.Spec.RemotePrivateCidrs
		changed = true
	}
	if ipsecConn.Status.LocalPrivateCidrs == "" && ipsecConn.Spec.LocalPrivateCidrs != "" {
		ipsecConn.Status.LocalPrivateCidrs = ipsecConn.Spec.LocalPrivateCidrs
		changed = true
	}
	return changed
}

func labelsForIpsecConnection(conn *vpngwv1.IpsecConn) map[string]string {
	return map[string]string{
		VpnGwLabel: conn.Name,
	}
}

func (r *IpsecConnReconciler) handleAddOrUpdateIpsecConnection(ipsecConn *vpngwv1.IpsecConn, req ctrl.Request) SyncState {
	// create ipsecConn statefulset
	namespacedName := req.NamespacedName.String()
	r.Log.Info("start handleAddOrUpdateIpsecConnection", "ipsecConn", namespacedName)
	defer r.Log.Info("end handleAddOrUpdateIpsecConnection", "ipsecConn", namespacedName)

	// validate ipsecConn spec
	if err := r.validateIpsecConnection(ipsecConn, namespacedName); err != nil {
		r.Log.Error(err, "failed to validate ipsecConn")
		// invalid spec no retry
		return SyncStateErrorNoRetry
	}
	newConn := ipsecConn.DeepCopy()
	if r.isChanged(newConn) {
		labels := labelsForIpsecConnection(newConn)
		newConn.SetLabels(labels)
		err := r.Update(context.Background(), newConn)
		if err != nil {
			r.Log.Error(err, "failed to update the ipsecConn")
			return SyncStateError
		}
		return SyncStateSuccess
	}
	return SyncStateSuccess
}

//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=ipsecconns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=ipsecconns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=ipsecconns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IpsecConn object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *IpsecConnReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	namespacedName := req.NamespacedName.String()
	r.Log.Info("start reconcile", "ipsecConn", namespacedName)
	defer r.Log.Info("end reconcile", "ipsecConn", namespacedName)
	updates.Inc()
	// fetch ipsecConn
	ipsecConn, err := r.getIpsecConnection(ctx, req.NamespacedName)
	if err != nil {
		r.Log.Error(err, "failed to get ipsecConn")
		return ctrl.Result{}, err
	}
	if ipsecConn == nil {
		// ipsecConn is deleted
		// onwner reference will trigger vpn gw update ipsec connections
		return ctrl.Result{}, nil
	}

	r.Handler = r.handleAddOrUpdateIpsecConnection
	// TODO:// Handler should set in main.go

	// update vpn gw spec
	res := r.Handler(ipsecConn, req)
	switch res {
	case SyncStateError:
		updateErrors.Inc()
		r.Log.Error(err, "failed to handle ipsecConn")
		return ctrl.Result{}, errRetry
	case SyncStateErrorNoRetry:
		updateErrors.Inc()
		r.Log.Error(err, "failed to handle ipsecConn")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IpsecConnReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vpngwv1.IpsecConn{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(
					func(object client.Object) bool {
						_, ok := object.(*vpngwv1.IpsecConn)
						if !ok {
							err := errors.New("invalid ipsecConn")
							r.Log.Error(err, "expected ipsecConn in worequeue but got something else")
							return false
						}
						return true
					},
				),
			),
		).
		Complete(r)
}

func (r *IpsecConnReconciler) getIpsecConnection(ctx context.Context, name types.NamespacedName) (*vpngwv1.IpsecConn, error) {
	var res vpngwv1.IpsecConn
	err := r.Get(ctx, name, &res)
	if apierrors.IsNotFound(err) { // in case of delete, get fails and we need to pass nil to the handler
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

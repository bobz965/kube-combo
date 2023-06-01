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
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	vpngwv1 "github.com/bobz965/kube-ovn-operator/api/v1"
	// kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	VpcVpnGatewayCmVersion = ""
	// TODO:// 需要跟踪cm的版本，如果cm的版本发生变化，需要重新启动vpn gateway

)

const (
	SslVpnServer   = "ssl-vpn-server"
	IpsecVpnServer = "ipsec-vpn-server"

	SslVpnStartUpCMD   = "/etc/openvpn/setup/configure.sh"
	IpsecVpnStartUpCMD = "/etc/ipsec/setup/configure.sh"

	SslVpnGatewayLabel   = "ssl-vpn-gateway"
	IpsecVpnGatewayLabel = "ipsec-vpn-gateway"

	KubeovnIpAddressAnnotation     = "ovn.kubernetes.io/ip_address"
	KubeovnLogicalSwitchAnnotation = "ovn.kubernetes.io/logical_switch"
)

// VpnGwReconciler reconciles a VpnGw object
type VpnGwReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
	Handler   func(logr.Logger, string) SyncState
	Reload    chan event.GenericEvent
}

func (r *VpnGwReconciler) validateVpnGw(gw *vpngwv1.VpnGw, req ctrl.Request) error {
	if gw.Spec.Subnet == "" {
		err := fmt.Errorf("vpn gw subnet is required")
		r.Log.Error(err, "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
		return err
	}
	return nil
}

func (r *VpnGwReconciler) isVpnGwChanged(gw *vpngwv1.VpnGw) bool {
	// compare spec and status to check if vpn gw changed
	// TODO
	if gw.Status.Subnet == "" && gw.Spec.Subnet != "" {
		// subnet not support change
		gw.Status.Subnet = gw.Spec.Subnet
		return true
	}
	if gw.Status.Ip != gw.Spec.Ip {
		gw.Status.Ip = gw.Spec.Ip
		return true
	}
	if !reflect.DeepEqual(gw.Spec.Selector, gw.Status.Selector) {
		gw.Status.Selector = gw.Spec.Selector
		return true
	}
	if !reflect.DeepEqual(gw.Spec.Tolerations, gw.Status.Tolerations) {
		gw.Status.Tolerations = gw.Spec.Tolerations
		return true
	}
	if !reflect.DeepEqual(gw.Spec.Affinity, gw.Status.Affinity) {
		gw.Status.Affinity = gw.Spec.Affinity
		return true
	}
	return false
}

func (*VpnGwReconciler) genSslVpnGwStatefulSet(gw *vpngwv1.VpnGw, oldSts *appsv1.StatefulSet) (newSts *appsv1.StatefulSet) {
	replicas := int32(1)
	// TODO: HA
	allowPrivilegeEscalation := true
	privileged := true
	labels := map[string]string{
		SslVpnGatewayLabel:   strconv.FormatBool(gw.Spec.EnableSslVpn),
		IpsecVpnGatewayLabel: strconv.FormatBool(gw.Spec.EnableIpsecVpn),
	}
	newPodAnnotations := map[string]string{}
	if oldSts != nil && len(oldSts.Annotations) != 0 {
		newPodAnnotations = oldSts.Annotations
	}
	podAnnotations := map[string]string{
		KubeovnLogicalSwitchAnnotation: gw.Spec.Subnet,
		KubeovnIpAddressAnnotation:     gw.Spec.Ip,
	}
	for key, value := range podAnnotations {
		newPodAnnotations[key] = value
	}

	selectors := make(map[string]string)
	for _, v := range gw.Spec.Selector {
		parts := strings.Split(strings.TrimSpace(v), ":")
		if len(parts) != 2 {
			continue
		}
		selectors[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	containers := []corev1.Container{}
	if gw.Spec.EnableSslVpn {
		sslContainer := corev1.Container{
			Name:            SslVpnServer,
			Image:           gw.Spec.SslVpnImage,
			Command:         []string{"bash"},
			Args:            []string{"-c", "sleep infinity"},
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &privileged,
				AllowPrivilegeEscalation: &allowPrivilegeEscalation,
			},
		}
		containers = append(containers, sslContainer)
	}
	if gw.Spec.EnableIpsecVpn {
		ipsecContainer := corev1.Container{
			Name:            IpsecVpnServer,
			Image:           gw.Spec.SslVpnImage,
			Command:         []string{"bash"},
			Args:            []string{"-c", "sleep infinity"},
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &privileged,
				AllowPrivilegeEscalation: &allowPrivilegeEscalation,
			},
		}
		containers = append(containers, ipsecContainer)
	}

	newSts = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gw.Name,
			Namespace: gw.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: newPodAnnotations,
				},
				Spec: corev1.PodSpec{
					Containers:   containers,
					NodeSelector: selectors,
					Tolerations:  gw.Spec.Tolerations,
					Affinity:     &gw.Spec.Affinity,
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	return
}

func (r *VpnGwReconciler) updateVpnGwStatus(gw *vpngwv1.VpnGw) error {
	var changed bool
	if gw.Status.Subnet == "" && gw.Status.Subnet != gw.Spec.Subnet {
		// subnet not support update
		gw.Status.Subnet = gw.Spec.Subnet
		changed = true
	}
	if gw.Status.Ip != gw.Spec.Ip {
		gw.Status.Ip = gw.Spec.Ip
		changed = true
	}
	if !reflect.DeepEqual(gw.Spec.Selector, gw.Status.Selector) {
		gw.Status.Selector = gw.Spec.Selector
		changed = true
	}
	if !reflect.DeepEqual(gw.Spec.Tolerations, gw.Status.Tolerations) {
		gw.Status.Tolerations = gw.Spec.Tolerations
		changed = true
	}
	if !reflect.DeepEqual(gw.Spec.Affinity, gw.Status.Affinity) {
		gw.Status.Affinity = gw.Spec.Affinity
		changed = true
	}

	if changed {
		return r.Status().Update(context.Background(), gw)
	}
	return nil
}

func (r *VpnGwReconciler) handleAddOrUpdateVpcVpnGateway(gw *vpngwv1.VpnGw, req ctrl.Request) error {
	// create vpn gw statefulset
	key := fmt.Sprintf("%s/%s", gw.Namespace, gw.Name)
	r.Log.Info("controller", "VpnGwReconciler", "start handleAddOrUpdateVpcVpnGateway", key)
	defer r.Log.Info("controller", "VpnGwReconciler", "end handleAddOrUpdateVpcVpnGateway", key)

	if err := r.validateVpnGw(gw, req); err != nil {
		r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
		return err
	}
	// create or update statefulset
	var needToCreate, needToUpdate bool
	oldSts, err := r.getStatefulSet(context.Background(), req.NamespacedName)
	if err != nil {
		r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
		return err
	}
	if oldSts == nil {
		needToCreate = true
	}

	// subnet, err := r.getKubeovnSubnet(context.Background(), req.NamespacedName)
	// if err != nil {
	// 	r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
	// 	return err
	// }
	if needToCreate {
		newSts := r.genSslVpnGwStatefulSet(gw, nil)
		err = r.Create(context.Background(), newSts)
		if err != nil {
			r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
			return err
		}
	}
	gw, err = r.getVpnGw(context.Background(), req.NamespacedName)
	gw = gw.DeepCopy()
	// gw.Status.Subnet = subnet.Name
	// gw.Status.OvpnPodSubnetCidr = subnet.Spec.PodSubnet
	if err != nil {
		r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
		return err
	}
	if !needToCreate && r.isVpnGwChanged(gw) {
		needToUpdate = true
	}
	if needToUpdate {
		newSts := r.genSslVpnGwStatefulSet(gw, oldSts.DeepCopy())
		err = r.Update(context.Background(), newSts)
		if err != nil {
			r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
			return err
		}
	}

	if err = r.updateVpnGwStatus(gw); err != nil {
		r.Log.Error(err, "controller", "VpnGwReconciler", "name", req.NamespacedName.String(), "vpnGw", dumpResource(gw))
		return err
	}
	return nil
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
	r.Log = ctrllog.FromContext(ctx)
	r.Log.Info("controller", "VpnGwReconciler", "start reconcile", req.NamespacedName.String())
	defer r.Log.Info("controller", "VpnGwReconciler", "end reconcile", req.NamespacedName.String())
	updates.Inc()

	var vpnGw *vpngwv1.VpnGw
	var err error
	vpnGw, err = r.getVpnGw(ctx, req.NamespacedName)
	if err != nil {
		r.Log.Error(err, "failed to get vpn gw")
		return ctrl.Result{}, err
	}
	// delete

	// add or update
	err = r.handleAddOrUpdateVpcVpnGateway(vpnGw, req)
	if err != nil {
		r.Log.Error(err, "failed to handle or update vpn gw")
		return ctrl.Result{}, err
	}
	res := r.Handler(r.Log, req.NamespacedName.String())
	switch res {
	case SyncStateError:
		updateErrors.Inc()
		r.Log.Error(err, "failed to handle vpn gw")
		return ctrl.Result{}, errRetry
	case SyncStateErrorNoRetry:
		updateErrors.Inc()
		r.Log.Error(err, "failed to handle vpn gw")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VpnGwReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vpngwv1.VpnGw{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(
					func(object client.Object) bool {
						vpnGw, ok := object.(*vpngwv1.VpnGw)
						if !ok {
							err := errors.New("invalid vpn gw")
							r.Log.Error(err, "expected vpn gw in worequeue but got something else")
							return false
						}
						r.Log.Info("controller", "SetupWithManager", "vpnGw", dumpResource(vpnGw))
						return true
					},
				),
			),
		).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

// TODO: vpn gw server 基于sts来维护，后续需要通过owner机制来自动化维护依赖与被依赖的维护关系，包括finalizer的维护以及owner的维护。
// TODO: vpn gw 启动变量 基于cm来维护，后续需要通过owner机制来自动化维护依赖与被依赖的维护关系，包括finalizer的维护以及owner的维护。

// func (r *VpnGwReconciler) getKubeovnSubnet(ctx context.Context, name types.NamespacedName) (*kubeovnv1.Subnet, error) {
// 	var res kubeovnv1.Subnet
// 	err := r.Get(ctx, name, &res)
// 	if apierrors.IsNotFound(err) { // in case of delete, get fails and we need to pass nil to the handler
// 		return nil, nil
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &res, nil
// }

func (r *VpnGwReconciler) getVpnGw(ctx context.Context, name types.NamespacedName) (*vpngwv1.VpnGw, error) {
	var res vpngwv1.VpnGw
	err := r.Get(ctx, name, &res)
	if apierrors.IsNotFound(err) { // in case of delete, get fails and we need to pass nil to the handler
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *VpnGwReconciler) getStatefulSet(ctx context.Context, name types.NamespacedName) (*appsv1.StatefulSet, error) {
	var res appsv1.StatefulSet
	err := r.Get(ctx, name, &res)
	if apierrors.IsNotFound(err) { // in case of delete, get fails and we need to pass nil to the handler
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

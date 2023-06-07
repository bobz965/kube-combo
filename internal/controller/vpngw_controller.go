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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	vpngwv1 "github.com/bobz965/kube-ovn-operator/api/v1"
	// kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	SslVpnServer   = "ssl-vpn-server"
	IpsecVpnServer = "ipsec-vpn-server"

	IpsecVpnLocalPortKey  = "ipsec-local"
	IpsecVpnRemotePortKey = "ipsec-remote"

	SslVpnStartUpCMD   = "/etc/openvpn/setup/configure.sh"
	IpsecVpnStartUpCMD = "/etc/ipsec/setup/configure.sh"

	EnableSslVpnLabel   = "enable_ssl_vpn"
	EnableIpsecVpnLabel = "enable_ipsec_vpn"

	KubeovnIpAddressAnnotation = "ovn.kubernetes.io/ip_address"
	// TODO:// HA use ip pool
	KubeovnLogicalSwitchAnnotation = "ovn.kubernetes.io/logical_switch"

	KubeovnIngressRateAnnotation = "ovn.kubernetes.io/ingress_rate"
	KubeovnEgressRateAnnotation  = "ovn.kubernetes.io/egress_rate"

	IpSecLocalPort  = 500
	IpSecRemotePort = 4500
	IpsecProto      = "UDP"

	// vpn gw pod env
	OvpnProtoKey      = "OVPN_PROTO"
	OvpnPortKey       = "OVPN_PORT"
	OvpnCipherKey     = "OVPN_CIPHER"
	OvpnSubnetCidrKey = "OVPN_SUBNET_CIDR"

	IpsecRemoteAddrsKey = "IPSEC_REMOTE_ADDRS"
	IpsecRemoteTsKey    = "IPSEC_REMOTE_TS"
)

// VpnGwReconciler reconciles a VpnGw object
type VpnGwReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Namespace string
	Handler   func(gw *vpngwv1.VpnGw, req ctrl.Request) SyncState
	Reload    chan event.GenericEvent
}

func (r *VpnGwReconciler) validateVpnGw(gw *vpngwv1.VpnGw, namespacedName string) error {
	if gw.Spec.Subnet == "" {
		err := fmt.Errorf("vpn gw subnet is required")
		r.Log.Error(err, "should set subnet")
		return err
	}
	if gw.Spec.Ip == "" {
		r.Log.Info("vpn gw ip should random allocate", "name", namespacedName)
	}

	if gw.Spec.Replicas != 1 {
		err := fmt.Errorf("vpn gw replicas should only be 1 for now, ha mode will be supported in the future")
		r.Log.Error(err, "should set reasonable replicas")
		return err
	}

	if gw.Spec.EnableSslVpn {
		if gw.Spec.OvpnCipher == "" {
			err := fmt.Errorf("ssl vpn cipher is required")
			r.Log.Error(err, "should set cipher")
			return err
		}
		if gw.Spec.OvpnProto == "" {
			err := fmt.Errorf("ssl vpn proto is required")
			r.Log.Error(err, "should set ssl vpn proto")
			return err
		}
		if gw.Spec.OvpnPort == 0 {
			err := fmt.Errorf("ssl vpn port is required")
			r.Log.Error(err, "should set vpn port, udp 1149, tcp 443")
			return err
		}
		if gw.Spec.OvpnSubnetCidr == "" {
			err := fmt.Errorf("ssl vpn subnet cidr is required")
			r.Log.Error(err, "should set ssl vpn client and server subnet")
			return err
		}
		if gw.Spec.OvpnProto != "udp" && gw.Spec.OvpnProto != "tcp" {
			err := fmt.Errorf("ssl vpn proto should be udp or tcp")
			r.Log.Error(err, "should set reasonable vpn proto")
			return err
		}
		if gw.Spec.SslVpnImage == "" {
			err := fmt.Errorf("ssl vpn image is required")
			r.Log.Error(err, "should set ssl vpn image")
			return err
		}
	}
	return nil
}

func (r *VpnGwReconciler) isChanged(gw *vpngwv1.VpnGw) bool {
	changed := false
	if gw.Status.Subnet == "" && gw.Spec.Subnet != "" {
		// subnet not support change
		gw.Status.Subnet = gw.Spec.Subnet
		changed = true
	}

	if gw.Status.Cpu != gw.Spec.Cpu {
		gw.Status.Cpu = gw.Spec.Cpu
		changed = true
	}
	if gw.Status.Memory != gw.Spec.Memory {
		gw.Status.Memory = gw.Spec.Memory
		changed = true
	}
	if gw.Status.QoSBandwidth != gw.Spec.QoSBandwidth {
		gw.Status.QoSBandwidth = gw.Spec.QoSBandwidth
		changed = true
	}
	if gw.Status.Ip != gw.Spec.Ip {
		gw.Status.Ip = gw.Spec.Ip
		changed = true
	}
	if gw.Status.Replicas != gw.Spec.Replicas {
		gw.Status.Replicas = gw.Spec.Replicas
		changed = true
	}

	if gw.Status.EnableSslVpn != gw.Spec.EnableSslVpn {
		gw.Status.EnableSslVpn = gw.Spec.EnableSslVpn
		if gw.Status.OvpnCipher != gw.Spec.OvpnCipher {
			gw.Status.OvpnCipher = gw.Spec.OvpnCipher
		}
		if gw.Status.OvpnProto != gw.Spec.OvpnProto {
			gw.Status.OvpnProto = gw.Spec.OvpnProto
		}
		if gw.Status.OvpnPort != gw.Spec.OvpnPort {
			gw.Status.OvpnPort = gw.Spec.OvpnPort
		}
		if gw.Status.OvpnSubnetCidr != gw.Spec.OvpnSubnetCidr {
			gw.Status.OvpnSubnetCidr = gw.Spec.OvpnSubnetCidr
		}
		if gw.Status.SslVpnImage != gw.Spec.SslVpnImage {
			gw.Status.SslVpnImage = gw.Spec.SslVpnImage
		}
		changed = true
	}

	if gw.Status.EnableIpsecVpn != gw.Spec.EnableIpsecVpn {
		gw.Status.EnableIpsecVpn = gw.Spec.EnableIpsecVpn
		if gw.Status.IpsecRemoteAddrs != gw.Spec.IpsecRemoteAddrs {
			gw.Status.IpsecRemoteAddrs = gw.Spec.IpsecRemoteAddrs
		}
		if gw.Status.IpsecRemoteAddrs != gw.Spec.IpsecRemoteAddrs {
			gw.Status.IpsecRemoteAddrs = gw.Spec.IpsecRemoteAddrs
		}
		if gw.Status.IpsecVpnImage != gw.Spec.IpsecVpnImage {
			gw.Status.IpsecVpnImage = gw.Spec.IpsecVpnImage

		}
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
	return changed
}

func (r *VpnGwReconciler) statefulSetForVpnGw(gw *vpngwv1.VpnGw, oldSts *appsv1.StatefulSet) (newSts *appsv1.StatefulSet) {
	namespacedName := fmt.Sprintf("%s/%s", gw.Namespace, gw.Name)
	r.Log.Info("start statefulSetForVpnGw", "vpn gw", namespacedName)
	defer r.Log.Info("end statefulSetForVpnGw", "vpn gw", namespacedName)
	replicas := gw.Spec.Replicas
	// TODO: HA may use router lb external eip as fontend
	allowPrivilegeEscalation := true
	privileged := true
	labels := labelsForVpnGw(gw)
	newPodAnnotations := map[string]string{}
	if oldSts != nil && len(oldSts.Annotations) != 0 {
		newPodAnnotations = oldSts.Annotations
	}
	podAnnotations := map[string]string{
		KubeovnLogicalSwitchAnnotation: gw.Spec.Subnet,
		KubeovnIpAddressAnnotation:     gw.Spec.Ip,
		KubeovnIngressRateAnnotation:   gw.Spec.QoSBandwidth,
		KubeovnEgressRateAnnotation:    gw.Spec.QoSBandwidth,
	}
	for key, value := range podAnnotations {
		newPodAnnotations[key] = value
	}

	containers := []corev1.Container{}
	if gw.Spec.EnableSslVpn {
		sslContainer := corev1.Container{
			Name:  SslVpnServer,
			Image: gw.Spec.SslVpnImage,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(gw.Spec.Cpu),
					corev1.ResourceMemory: resource.MustParse(gw.Spec.Memory),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(gw.Spec.Cpu),
					corev1.ResourceMemory: resource.MustParse(gw.Spec.Memory),
				},
			},
			Command: []string{SslVpnStartUpCMD},
			Ports: []corev1.ContainerPort{{
				ContainerPort: int32(gw.Spec.OvpnPort),
				Name:          SslVpnServer,
				Protocol:      corev1.Protocol(strings.ToUpper(gw.Spec.OvpnProto)),
			}},
			Env: []corev1.EnvVar{
				{
					Name:  OvpnProtoKey,
					Value: gw.Spec.OvpnProto,
				},
				{
					Name:  OvpnPortKey,
					Value: strconv.Itoa(gw.Spec.OvpnPort),
				},
				{
					Name:  OvpnCipherKey,
					Value: gw.Spec.OvpnCipher,
				},
				{
					Name:  OvpnSubnetCidrKey,
					Value: gw.Spec.OvpnSubnetCidr,
				},
			},
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
			Name:  IpsecVpnServer,
			Image: gw.Spec.IpsecVpnImage,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(gw.Spec.Cpu),
					corev1.ResourceMemory: resource.MustParse(gw.Spec.Memory),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(gw.Spec.Cpu),
					corev1.ResourceMemory: resource.MustParse(gw.Spec.Memory),
				},
			},
			Command: []string{"bash"},
			Args:    []string{"-c", "sleep infinity"},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: IpSecLocalPort,
					Name:          IpsecVpnLocalPortKey,
					Protocol:      corev1.Protocol(IpsecProto),
				},
				{
					ContainerPort: IpSecRemotePort,
					Name:          IpsecVpnRemotePortKey,
					Protocol:      corev1.Protocol(IpsecProto)},
			},
			Env: []corev1.EnvVar{
				{
					Name:  IpsecRemoteAddrsKey,
					Value: gw.Spec.IpsecRemoteAddrs,
				},
				{
					Name:  IpsecRemoteTsKey,
					Value: gw.Spec.IpsecRemoteTs,
				},
			},
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
					Containers: containers,
				},
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	if len(gw.Spec.Selector) > 0 {
		r.Log.Info("init vpn gw pod selector", "vpn gw", namespacedName)
		selectors := make(map[string]string)
		for _, v := range gw.Spec.Selector {
			parts := strings.Split(strings.TrimSpace(v), ":")
			if len(parts) != 2 {
				continue
			}
			selectors[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		newSts.Spec.Template.Spec.NodeSelector = selectors
	}

	if len(gw.Spec.Tolerations) > 0 {
		r.Log.Info("init vpn gw pod tolerations", "vpn gw", namespacedName)
		newSts.Spec.Template.Spec.Tolerations = gw.Spec.Tolerations
	}

	if gw.Spec.Affinity.NodeAffinity != nil ||
		gw.Spec.Affinity.PodAffinity != nil ||
		gw.Spec.Affinity.PodAntiAffinity != nil {
		r.Log.Info("init vpn gw pod node affinity", "vpn gw", namespacedName)
		newSts.Spec.Template.Spec.Affinity = &gw.Spec.Affinity
	}

	// set gw instance as the owner and controller
	controllerutil.SetControllerReference(gw, newSts, r.Scheme)
	return
}

// belonging to the given vpn gw CR name.
func labelsForVpnGw(gw *vpngwv1.VpnGw) map[string]string {
	return map[string]string{"app": "vpngw",
		EnableSslVpnLabel:   strconv.FormatBool(gw.Spec.EnableSslVpn),
		EnableIpsecVpnLabel: strconv.FormatBool(gw.Spec.EnableIpsecVpn),
	}
}

func (r *VpnGwReconciler) handleAddOrUpdateVpnGw(gw *vpngwv1.VpnGw, req ctrl.Request) SyncState {
	// create vpn gw statefulset
	namespacedName := req.NamespacedName.String()
	r.Log.Info("start handleAddOrUpdateVpnGw", "vpn gw", namespacedName)
	defer r.Log.Info("end handleAddOrUpdateVpnGw", "vpn gw", namespacedName)

	// create or update statefulset
	var needToCreate, needToUpdate bool
	oldSts := &appsv1.StatefulSet{}
	err := r.Get(context.Background(), req.NamespacedName, oldSts)
	if err != nil {
		if apierrors.IsNotFound(err) {
			needToCreate = true
		} else {
			r.Log.Error(err, "failed to get old statefulset")
			return SyncStateError
		}
	}
	if needToCreate {
		newSts := r.statefulSetForVpnGw(gw, nil)
		err = r.Create(context.Background(), newSts)
		if err != nil {
			r.Log.Error(err, "failed to create the new statefulset")
			return SyncStateError
		}
		err = r.Status().Update(context.Background(), gw)
		if err != nil {
			r.Log.Error(err, "failed to update vpn gw after creating new statefulset")
			return SyncStateError
		}
		return SyncStateSuccess
	}
	gw = gw.DeepCopy()
	if !needToCreate && r.isChanged(gw) {
		needToUpdate = true
	}
	if needToUpdate {
		newSts := r.statefulSetForVpnGw(gw, oldSts.DeepCopy())
		err = r.Update(context.Background(), newSts)
		if err != nil {
			r.Log.Error(err, "failed to update the statefulset")
			return SyncStateError
		}
		err = r.Status().Update(context.Background(), gw)
		if err != nil {
			r.Log.Error(err, "failed to update vpn gw after updating exist statefulset")
			return SyncStateError
		}
		return SyncStateSuccess
	}
	return SyncStateSuccess
}

//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=vpngws,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=vpngws/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vpn-gw.kube-ovn-operator.com,resources=vpngws/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *VpnGwReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// delete vpn gw itself, its owned statefulset will be deleted automatically
	namespacedName := req.NamespacedName.String()
	r.Log.Info("start reconcile", "vpn gw", namespacedName)
	defer r.Log.Info("end reconcile", "vpn gw", namespacedName)
	updates.Inc()

	// fetch vpn gw
	gw, err := r.getVpnGw(ctx, req.NamespacedName)
	if err != nil {
		r.Log.Error(err, "failed to get vpn gw")
		return ctrl.Result{}, err
	}
	if gw == nil {
		return ctrl.Result{}, nil
	}
	// validate vpn gw
	if err := r.validateVpnGw(gw, namespacedName); err != nil {
		r.Log.Error(err, "failed to validate vpn gw")
		return ctrl.Result{}, err
	}
	// fetch vpn gw statefulset, if not exist, create it

	r.Handler = r.handleAddOrUpdateVpnGw
	// TODO:// Handler should set in main.go

	res := r.Handler(gw, req)
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
						_, ok := object.(*vpngwv1.VpnGw)
						if !ok {
							err := errors.New("invalid vpn gw")
							r.Log.Error(err, "expected vpn gw in worequeue but got something else")
							return false
						}
						return true
					},
				),
			),
		).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

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

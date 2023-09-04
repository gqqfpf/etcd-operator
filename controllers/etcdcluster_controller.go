/*
Copyright 2023 fpf.

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

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1alpha1 "github.com/gqq/etcd-operator/api/v1alpha1"
)

// EtcdClusterReconciler reconciles a EtcdCluster object
type EtcdClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *EtcdClusterReconciler) Schemes() *runtime.Scheme {
	return r.Scheme
}

//+kubebuilder:rbac:groups=etcd.gqq.com,resources=etcdclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.gqq.com,resources=etcdclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.gqq.com,resources=etcdclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *EtcdClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)
	clusetrlog := logs.WithValues("etcdcluster", req.NamespacedName)
	// TODO(user): your logic here

	//首先获取etcdcluster 实例
	var etcdcluster etcdv1alpha1.EtcdCluster
	if err := r.Get(ctx, req.NamespacedName, &etcdcluster); err != nil {
		//if client.IgnoreNotFound(err) != nil {
		//	return ctrl.Result{}, nil
		//}
		//return ctrl.Result{}, err
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 一斤获取到etcdcluster 实例
	// 创建或者更新 statefulset 以及service 对象
	// CreateOrUpdate
	// 调谐 观察当前状态和期望状态对比

	// CreateOrUpdate Service
	var svc corev1.Service
	svc.Namespace = etcdcluster.Namespace
	svc.Name = etcdcluster.Name
	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		// 调谐函数必须在这里实现，，实际就是拼装svc
		MutateHeadlessSvc(&etcdcluster, &svc)
		return controllerutil.SetControllerReference(&etcdcluster, &svc, r.Schemes())
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	clusetrlog.Info("Create Or Update Result", "service", or)

	var statefulset appsv1.StatefulSet
	statefulset.Name = etcdcluster.Name
	statefulset.Namespace = etcdcluster.Namespace

	stateresult, err := ctrl.CreateOrUpdate(ctx, r.Client, &statefulset, func() error {
		// 调谐函数 必须要在这里实现，实际就是拼装statefulset
		MutateStatefulSet(&etcdcluster, &statefulset)
		return controllerutil.SetControllerReference(&etcdcluster, &statefulset, r.Schemes())
	})

	if err != nil {
		return ctrl.Result{}, err
	}
	clusetrlog.Info("create Or Update Result", "StatefulSet", stateresult)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

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
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1alpha1 "github.com/gqq/etcd-operator/api/v1alpha1"
)

// EtcdBackupReconciler reconciles a EtcdBackup object
type EtcdBackupReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	BacupImage string
}

//+kubebuilder:rbac:groups=etcd.gqq.com,resources=etcdbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.gqq.com,resources=etcdbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.gqq.com,resources=etcdbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *EtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logs := log.FromContext(ctx)
	ectdbackupLogs := logs.WithValues("etcdbackup", req.NamespacedName)
	// TODO(user): your logic here
	state, err := r.getState(ctx, req)
	if err != nil {
		return ctrl.Result{}, nil
	}
	// 根据状态判断 下一步要执行的动作
	var action Action

	switch {
	case state.backup == nil: // 被删除了
		ectdbackupLogs.Info("Backup Object not found. Ignoring.")
	case !state.backup.DeletionTimestamp.IsZero(): // 标记为了删除
		ectdbackupLogs.Info("Backup Object has been deleted. Ignoring")
	case state.backup.Status.Phase == "": // 开始备份
		ectdbackupLogs.Info("Backup Object Starting . Update status")
		newBackup := state.backup.DeepCopy()                           // 深拷贝一份
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseBackingUp // 更新状态为备份中
		action = &PatchStatus{                                         // 下一不要执行的动作
			client:   r.Client,
			original: state.backup,
			new:      newBackup,
		}
	case state.backup.Status.Phase == etcdv1alpha1.EtcdBackupPhaseFailed: // 备份失败
		ectdbackupLogs.Info("Backup has fialed. Ignoring")
	case state.backup.Status.Phase == etcdv1alpha1.EtcdBakcupPhaseCompleted: // 备份完成
		ectdbackupLogs.Info("Backup has completed. Ignoring")
	case state.actual.pod == nil: // 当前还没备份的pod
		ectdbackupLogs.Info("Backup pod does not exists. Createing")
		action = &CreateObject{
			client: r.Client,
			obj:    state.desired.pod,
		}
	case state.actual.pod.Status.Phase == corev1.PodFailed: // pod 执行备份失败
		ectdbackupLogs.Info("Backip pod failed Update status")
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseFailed
		action = &PatchStatus{
			client:   r.Client,
			original: state.backup,
			new:      newBackup,
		}
	case state.actual.pod.Status.Phase == corev1.PodSucceeded: // pod 备份完成
		ectdbackupLogs.Info("Backup pod succedded Update status")
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBakcupPhaseCompleted
		action = &PatchStatus{
			client:   r.Client,
			original: state.backup,
			new:      newBackup,
		}

		if action != nil {
			if err := action.Execute(ctx); err != nil {
				return ctrl.Result{}, fmt.Errorf("execting action error: %s\n", err)
			}
		}

	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdBackup{}).
		Complete(r)
}

package controllers

import (
	"context"
	"fmt"
	etcdv1alpha1 "github.com/gqq/etcd-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// backupState 包含  EtcdBackup 真实和期望的状态（这里的状态并不是说status）
type backupState struct {
	backup  *etcdv1alpha1.EtcdBackup //EtcdBackup 对象本身
	actual  *backupStateContainer    //真实状态
	desired *backupStateContainer    // 期望状态
}

// / backupStateContainer 包含 EtcdBackup 状态
type backupStateContainer struct {
	pod *corev1.Pod
}

// setStateActual 用于设置 backupState 真是状态

func (r *EtcdBackupReconciler) setStateActual(ctx context.Context, state *backupState) error {
	var actual backupStateContainer

	key := client.ObjectKey{
		Name:      state.backup.Name,
		Namespace: state.backup.Namespace,
	}

	// 获取对应pod
	actual.pod = &corev1.Pod{}
	if err := r.Get(ctx, key, actual.pod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("getting pod error: %s\n", err)
		}
		actual.pod = nil
	}

	state.actual = &actual
	return nil
}

// setSatateDesired 用于设置 backupstate 的期望状态（根据EtcdBackup 对象）
func (r *EtcdBackupReconciler) setStateDesired(state *backupState) error {
	var desired backupStateContainer

	// 创建一个 管理的 pod 用于执行备份操作
	//po, err := podForBackup(state.backup, r.BacupImage)
	po, err := podForBackup(state.backup)
	if err != nil {
		return fmt.Errorf("computing pod for backup error : %s\n", err)
	}
	//设置  controller reference

	err = controllerutil.SetControllerReference(state.backup, po, r.Scheme)
	if err != nil {
		return fmt.Errorf("setting controller reference err %s\n", err)
	}
	desired.pod = po
	state.desired = &desired
	return nil
}

// func podForBackup(backup *etcdv1alpha1.EtcdBackup, image string) (*corev1.Pod, error) {
func podForBackup(backup *etcdv1alpha1.EtcdBackup) (*corev1.Pod, error) {
	var secretRef *corev1.SecretEnvSource
	var backupURL, backupEndpoint string

	if backup.Spec.StorageType == etcdv1alpha1.BackupStorageTypeS3 {
		backupURL = fmt.Sprintf("%s://%s", backup.Spec.StorageType, backup.Spec.S3.Path)
		backupEndpoint = backup.Spec.S3.Endpoint
		secretRef = &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: backup.Spec.S3.S3Secret,
			},
		}
	} else {
		backupURL = fmt.Sprintf("%s://%s", backup.Spec.StorageType, backup.Spec.OSS.Path)
		backupEndpoint = backup.Spec.OSS.Endpoint
		secretRef = &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: backup.Spec.OSS.OSSSecret,
			},
		}
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "etcd-backup",
					Image: backup.Spec.Image,
					Args: []string{
						"--etcd-url", backup.Spec.EtcdUrl,
						"--bucketname", backupURL,
						"--objectname", "snapshot.db",
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ENDPOINT",
							Value: backupEndpoint,
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: secretRef,
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("50Mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}, nil
}

// getState  获取当前应用的pod 状态，然后才方便判断 下一步动作

func (r *EtcdBackupReconciler) getState(ctx context.Context, req ctrl.Request) (*backupState, error) {
	var state backupState

	// 获取etcdbackup 对象
	state.backup = &etcdv1alpha1.EtcdBackup{}

	if err := r.Get(ctx, req.NamespacedName, state.backup); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, fmt.Errorf("getting backup error %s\n", err)
		}
		// 被删除直接忽略
		state.backup = nil
		return &state, nil
	}

	// 获取当前备份的真实状态
	if err := r.setStateActual(ctx, &state); err != nil {
		return nil, fmt.Errorf("setting actual state error: %s\n", err)
	}
	// 获取当前的期望状态
	if err := r.setStateDesired(&state); err != nil {
		return nil, fmt.Errorf("setting desired state error %s\n", err)
	}

	return &state, nil
}

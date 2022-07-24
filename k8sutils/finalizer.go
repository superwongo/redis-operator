package k8sutils

import (
	"context"

	"github.com/go-logr/logr"
	redisv1alpha1 "github.com/superwongo/redis-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	RedisFinalizer string = "redisFinalizer"
)

// 设置日志实例
func finalizerLogger(namespace string, name string) logr.Logger {
	reqLogger := log.Log.WithValues("Request.Service.Namespace", namespace, "Request.Finalizer.Name", name)
	return reqLogger
}

// 若实例标记为删除，则结束资源
func HandlerRedisFinalizer(cr *redisv1alpha1.Redis, cl client.Client) error {
	logger := finalizerLogger(cr.Namespace, RedisFinalizer)
	// 若存在删除时间戳，说明该资源已被删除
	if cr.GetDeletionTimestamp() != nil {
		// 若finalizer中存在redisFinalizer，则删除相关资源
		if controllerutil.ContainsFinalizer(cr, RedisFinalizer) {
			// 删除其service、headless service资源
			if err := finalizeRedisService(cr); err != nil {
				return err
			}
			// 删除其pvc资源
			if err := finalizeRedisPVC(cr); err != nil {
				return err
			}
		}
		// 移除finalizer中存在redisFinalizer
		controllerutil.RemoveFinalizer(cr, RedisFinalizer)
		// 更新资源信息
		if err := cl.Update(context.TODO(), cr); err != nil {
			logger.Error(err, "Could not remove finalizer "+RedisFinalizer)
			return err
		}
	}
}

func finalizeRedisService(cr *redisv1alpha1.Redis) error {
	logger := finalizerLogger(cr.Namespace, RedisFinalizer)
	serivceName, headlessSerivceName := cr.Name, cr.Name+"-headless"
	for _, svc := range []string{serivceName, headlessSerivceName} {
		err := generateK8sClient().CoreV1().Services(cr.Namespace).Delete(context.TODO(), svc, metav1.DeleteOptions{})
		if err != nil && errors.IsNotFound(err) {
			logger.Error(err, "Could not delete service "+svc)
			return err
		}
	}
	return nil
}

func finalizeRedisPVC(cr *redisv1alpha1.Redis) error {
	logger := finalizerLogger(cr.Namespace, RedisFinalizer)
	PVCName := cr.Name + "-" + cr.Name + "-0"
	err := generateK8sClient().CoreV1().PersistentVolumeClaims(cr.Namespace).Delete(context.TODO(), PVCName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "Could not delete Persistent Volume Claim "+PVCName)
		return err
	}
	return nil
}

func AddRedisFinalizer(cr *redisv1alpha1.Redis, cl client.Client) error {
	// 若finalizer中不存在redisFinalizer，则进行添加
	if !controllerutil.ContainsFinalizer(cr, RedisFinalizer) {
		controllerutil.AddFinalizer(cr, RedisFinalizer)
		return cl.Update(context.TODO(), cr)
	}
	return nil
}

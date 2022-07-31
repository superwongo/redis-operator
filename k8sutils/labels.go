package k8sutils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getRedisLabels(name, setupType, role string, labels map[string]string) map[string]string {
	lbls := map[string]string{
		"app":              name,
		"redis_setup_type": setupType,
		"role":             role,
	}
	for k, v := range labels {
		lbls[k] = v
	}
	return lbls
}

// 初始化statefulset注解
func generateStatefulSetsAnots(stsMeta metav1.ObjectMeta) map[string]string {
	anots := map[string]string{
		"redis.superwongo.com":      "true",
		"redis.superwongo.instance": stsMeta.GetName(),
	}
	for k, v := range stsMeta.GetAnnotations() {
		anots[k] = v
	}
	return filterAnnotations(anots)
}

// 删除一些自动添加的无用注解
func filterAnnotations(anots map[string]string) map[string]string {
	delete(anots, "kubectl.kubernetes.io/last-applied-configuration")
	return anots
}

// 初始化对象元数据
func generateObjectMetaInformation(name string, namespace string, labels map[string]string, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
	}
}

// 初始化元数据
func generateMetaInformation(resourceKind string, apiVersion string) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       resourceKind,
		APIVersion: apiVersion,
	}
}

// 标签选择
func labelSelector(labels map[string]string) *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: labels}
}

// 添加所有者引用
func AddOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

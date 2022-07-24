package k8sutils

import redisv1alpha1 "github.com/superwongo/redis-operator/api/v1alpha1"

func CreateStandaloneRedis(cr *redisv1alpha1.Redis) error {
	logger := statefulSetLogger(cr.Namespace, cr.ObjectMeta.Name)
	labels := getRedisLabels(cr.ObjectMeta.Name, "standalone", "standalone", cr.ObjectMeta.Labels)
	anots := generateStatefulSetsAnots(cr.ObjectMeta)
	objectMetaInfo := generateObjectMetaInformation(cr.ObjectMeta.Name, cr.Namespace, labels, anots)
	// TODO
	logger.Info(objectMetaInfo.Name)
}

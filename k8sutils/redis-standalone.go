package k8sutils

import (
	redisv1alpha1 "github.com/superwongo/redis-operator/api/v1alpha1"
)

var (
	enabledMetrics bool
)

func CreateStandaloneRedis(cr *redisv1alpha1.Redis) error {
	logger := statefulSetLogger(cr.Namespace, cr.ObjectMeta.Name)
	// 设置redis单例label
	labels := getRedisLabels(cr.ObjectMeta.Name, "standalone", "standalone", cr.ObjectMeta.Labels)
	// 设置redis单例annotation
	anots := generateObjectAnots(cr.ObjectMeta)
	// 设置redis单例Meta数据
	objectMetaInfo := generateObjectMetaInformation(cr.ObjectMeta.Name, cr.Namespace, labels, anots)
	// 创建或更新redis单例
	err := CreateOrUpdateStateful(
		cr.Namespace,
		objectMetaInfo,
		generateRedisStandaloneParams(cr),
		redisAsOwner(cr),
		generateRedisStandaloneContainerParams(cr),
		cr.Spec.Sidecars,
	)
	if err != nil {
		logger.Error(err, "Cannot create standalone statefulset for Redis")
		return err
	}
	return nil
}

// 初始化redis单例参数
func generateRedisStandaloneParams(cr *redisv1alpha1.Redis) statefulSetParameters {
	replicas := int32(1)
	res := statefulSetParameters{
		Replicas:          &replicas,
		NodeSelector:      cr.Spec.NodeSelector,
		SecurityContext:   cr.Spec.SecurityContext,
		PriorityClassName: cr.Spec.PriorityClassName,
		Affinity:          cr.Spec.Affinity,
		Tolerations:       cr.Spec.Tolerations,
	}
	if cr.Spec.KubernetesConfig.ImagePullSecrets != nil {
		res.ImagePullSecrets = cr.Spec.KubernetesConfig.ImagePullSecrets
	}
	if cr.Spec.RedisStorage != nil {
		res.PersistentVolumeClaim = cr.Spec.RedisStorage.VolumeClaimTemplate
	}
	if cr.Spec.RedisConfig != nil {
		res.ExternalConfig = cr.Spec.RedisConfig.AdditionalRedisConfig
	}
	if cr.Spec.RedisExporter != nil {
		res.EnabledMetrics = cr.Spec.RedisExporter.Enabled
	}
	return res
}

// 初始化redis单例容器参数
func generateRedisStandaloneContainerParams(cr *redisv1alpha1.Redis) containerParameters {
	trueProperty := true
	falseProperty := false
	containerProp := containerParameters{
		Role:            "standalone",
		Image:           cr.Spec.KubernetesConfig.Image,
		ImagePullPolicy: cr.Spec.KubernetesConfig.ImagePullPolicy,
		Resources:       cr.Spec.KubernetesConfig.Resources,
	}
	if cr.Spec.KubernetesConfig.ExistingPasswordSecret != nil {
		containerProp.EnabledPassword = &trueProperty
		containerProp.SecretName = cr.Spec.KubernetesConfig.ExistingPasswordSecret.Name
		containerProp.SecretKey = cr.Spec.KubernetesConfig.ExistingPasswordSecret.Key
	} else {
		containerProp.EnabledPassword = &falseProperty
	}
	if cr.Spec.RedisExporter != nil {
		containerProp.RedisExporterImage = cr.Spec.RedisExporter.Image
		containerProp.RedisExporterImagePullPolicy = cr.Spec.RedisExporter.ImagePullPolicy
		if cr.Spec.RedisExporter.Resources != nil {
			containerProp.RedisExporterResources = cr.Spec.RedisExporter.Resources
		}
		if cr.Spec.RedisExporter.EnvVars != nil {
			containerProp.RedisExporterEnvs = cr.Spec.RedisExporter.EnvVars
		}
	}
	if cr.Spec.ReadinessProbe != nil {
		containerProp.ReadinessProbe = cr.Spec.ReadinessProbe
	}
	if cr.Spec.LivenessProbe != nil {
		containerProp.LivenessProbe = cr.Spec.LivenessProbe
	}
	if cr.Spec.RedisStorage != nil {
		containerProp.PersistenceEnabled = &trueProperty
	}
	return containerProp
}

func CreateStandaloneService(cr *redisv1alpha1.Redis) error {
	logger := serviceLogger(cr.Namespace, cr.ObjectMeta.Name)
	// 初始化labels
	labels := getRedisLabels(cr.ObjectMeta.Name, "standalone", "standalone", cr.ObjectMeta.Labels)
	// 初始化annotations
	annotations := generateObjectAnots(cr.ObjectMeta)
	if cr.Spec.RedisExporter != nil && cr.Spec.RedisExporter.Enabled {
		enabledMetrics = true
	}
	// 初始化svc对象元数据
	objectMetaInfo := generateObjectMetaInformation(cr.ObjectMeta.Name, cr.Namespace, labels, annotations)
	// 初始化svc headless对象元数据
	headlessObjectMetaInfo := generateObjectMetaInformation(cr.ObjectMeta.Name+"-headless", cr.Namespace, labels, annotations)
	// 创建或更新headless svc
	err := CreateOrUpdateService(cr.Namespace, headlessObjectMetaInfo, redisAsOwner(cr), false, true)
	if err != nil {
		logger.Error(err, "Cannot create standalone headless service for Redis")
		return err
	}
	// 创建或更新svc
	err = CreateOrUpdateService(cr.Namespace, objectMetaInfo, redisAsOwner(cr), enabledMetrics, false)
	if err != nil {
		logger.Error(err, "Cannot create standalone service for Redis")
		return err
	}
	return nil
}

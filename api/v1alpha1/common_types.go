package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// redis基础配置
type KubernetesConfig struct {
	Image                  string                         `json:"image"`
	ImagePullPolicy        corev1.PullPolicy              `json:"imagePullPolicy,omitempty"`
	Resources              *corev1.ResourceRequirements   `json:"resources,omitempty"`
	ExistingPasswordSecret *ExistingPasswordSecret        `json:"redisSecret,omitempty"`
	ImagePullSecrets       *[]corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// 已存在密码secret
type ExistingPasswordSecret struct {
	Name *string `json:"name,omitempty"`
	Key  *string `json:"key,omitempty"`
}

// redis外部配置
type RedisConfig struct {
	AdditionalRedisConfig *string `json:"additionalRedisConfig,omitempty"`
}

// redis添加pvc和pv支持的接口
type Storage struct {
	VolumeClaimTemplate corev1.PersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

// 为redis exporter提供相关特征信息的接口
type RedisExporter struct {
	Enabled         bool                         `json:"enabled,omitempty"`
	Image           string                       `json:"image"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
	ImagePullPolicy corev1.PullPolicy            `json:"imagePullPolicy,omitempty"`
	EnvVars         *[]corev1.EnvVar             `json:"env,omitempty"`
}

// tls配置
type TLSConfig struct {
	CaKeyFile   string `json:"ca,omitempty"`
	CertKeyFile string `json:"cert,omitempty"`
	KeyFile     string `json:"key,omitempty"`
	// 包含证书的secret的引用
	Secret corev1.SecretVolumeSource `json:"secret"`
}

// ReadinessProbe and LivenessProbe探针接口
type Probe struct {
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" protobuf:"varint,2,opt,name=initialDelaySeconds"`
	TimeoutSeconds      int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,3,opt,name=timeoutSeconds"`
	PeriodSeconds       int32 `json:"periodSeconds,omitempty" protobuf:"varint,4,opt,name=periodSeconds"`
	SuccessThreshold    int32 `json:"successThreshold,omitempty" protobuf:"varint,5,opt,name=successThreshold"`
	FailureThreshold    int32 `json:"failureThreshold,omitempty" protobuf:"varint,6,opt,name=failureThreshold"`
}

type Sidecar struct {
	Name            string                       `json:"name"`
	Image           string                       `json:"image"`
	ImagePullPolicy corev1.PullPolicy            `json:"imagePullPolicy,omitempty"`
	Resouces        *corev1.ResourceRequirements `json:"resources,omitempty"`
	EnvVars         *[]corev1.EnvVar             `json:"env,omitempty"`
}

package k8sutils

import (
	"context"
	"fmt"
	"path"
	"sort"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	redisv1alpha1 "github.com/superwongo/redis-operator/api/v1alpha1"
)

func statefulSetLogger(namespace string, name string) logr.Logger {
	reqLogger := log.Log.WithValues("Request.StatefulSet.Namespace", namespace, "Request.StatefulSetName", name)
	return reqLogger
}

type statefulSetParameters struct {
	Replicas              *int32
	Metadata              metav1.ObjectMeta
	NodeSelector          map[string]string
	SecurityContext       *corev1.PodSecurityContext
	PriorityClassName     string
	Affinity              *corev1.Affinity
	Tolerations           *[]corev1.Toleration
	EnabledMetrics        bool
	PersistentVolumeClaim corev1.PersistentVolumeClaim
	ImagePullSecrets      *[]corev1.LocalObjectReference
	ExternalConfig        *string
}

type containerParameters struct {
	Image                        string
	ImagePullPolicy              corev1.PullPolicy
	Resources                    *corev1.ResourceRequirements
	RedisExporterImage           string
	RedisExporterImagePullPolicy corev1.PullPolicy
	RedisExporterResources       *corev1.ResourceRequirements
	RedisExporterEnvs            *[]corev1.EnvVar
	Role                         string
	EnabledPassword              *bool
	SecretName                   *string
	SecretKey                    *string
	PersistenceEnabled           *bool
	TLSConfig                    *redisv1alpha1.TLSConfig
	ReadinessProbe               *redisv1alpha1.Probe
	LivenessProbe                *redisv1alpha1.Probe
}

func CreateOrUpdateStateful(namespace string, stsMeta metav1.ObjectMeta, params statefulSetParameters, ownerRef metav1.OwnerReference, containerParams containerParameters, sidecars *[]redisv1alpha1.Sidecar) error {
	logger := statefulSetLogger(namespace, stsMeta.GetName())
	// 初始化statefulset声明
	statefulSetRef := generateStatefulSetsDef(stsMeta, params, ownerRef, containerParams, sidecars)
	// 查询已存在的statefulset
	storedStatefulSet, err := GetStatefulSet(namespace, stsMeta.GetName())
	if err != nil {
		// 将修改的配置添加到注解中
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(statefulSetRef); err != nil {
			logger.Error(err, "Unable to patch redis statefulset with comparison object")
			return err
		}
		// 不存在statefulset，则创建
		if errors.IsNotFound(err) {
			return createStatefulSet(namespace, statefulSetRef)
		}
		return err
	}
	// 存在statefulset，则更新
	return patchStatefulSet(namespace, storedStatefulSet, statefulSetRef)
}

func GetStatefulSet(namespace string, name string) (*appsv1.StatefulSet, error) {
	logger := statefulSetLogger(namespace, name)
	getOpts := metav1.GetOptions{
		TypeMeta: generateMetaInformation("StatefulSet", "apps/v1"),
	}
	statefulInfo, err := generateK8sClient().AppsV1().StatefulSets(namespace).Get(context.TODO(), name, getOpts)
	if err != nil {
		logger.Info("Redis statefulset get actions failed")
		return nil, err
	}
	logger.Info("Redis statefulset get action was successful")
	return statefulInfo, nil
}

// 初始化statefulset声明
func generateStatefulSetsDef(stsMeta metav1.ObjectMeta, params statefulSetParameters, ownerRef metav1.OwnerReference, containerParams containerParameters, sidecars *[]redisv1alpha1.Sidecar) *appsv1.StatefulSet {
	statefulset := &appsv1.StatefulSet{
		TypeMeta:   generateMetaInformation("StatefulSet", "apps/v1"),
		ObjectMeta: stsMeta,
		Spec: appsv1.StatefulSetSpec{
			Selector:    labelSelector(stsMeta.GetLabels()),
			ServiceName: stsMeta.GetName(),
			Replicas:    params.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      stsMeta.GetLabels(),
					Annotations: generateObjectAnots(stsMeta),
				},
				Spec: corev1.PodSpec{
					Containers:        generateContainerDef(stsMeta.GetName(), containerParams, params.EnabledMetrics, params.ExternalConfig, sidecars),
					NodeSelector:      params.NodeSelector,
					SecurityContext:   params.SecurityContext,
					PriorityClassName: params.PriorityClassName,
					Affinity:          params.Affinity,
				},
			},
		},
	}
	// 设置污点
	if params.Tolerations != nil {
		statefulset.Spec.Template.Spec.Tolerations = *params.Tolerations
	}
	// 设置镜像拉取策略
	if params.ImagePullSecrets != nil {
		statefulset.Spec.Template.Spec.ImagePullSecrets = *params.ImagePullSecrets
	}
	// 设置持久化卷模板
	if containerParams.PersistenceEnabled != nil && *containerParams.PersistenceEnabled {
		statefulset.Spec.VolumeClaimTemplates = append(statefulset.Spec.VolumeClaimTemplates, createPVCTemplate(stsMeta, params.PersistentVolumeClaim))
	}
	// 设置外部挂载configMap
	if params.ExternalConfig != nil {
		statefulset.Spec.Template.Spec.Volumes = getExternalConfig(*params.ExternalConfig)
	}
	// 挂载TLS secret
	if containerParams.TLSConfig != nil {
		statefulset.Spec.Template.Spec.Volumes = append(statefulset.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: "tls-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &containerParams.TLSConfig.Secret,
				},
			})
	}
	// 添加拥有者引用
	AddOwnerRefToObject(statefulset, ownerRef)
	return statefulset
}

// 初始化容器声明
func generateContainerDef(name string, containerParams containerParameters, enabledMetrics bool, externalConfig *string, sidecars *[]redisv1alpha1.Sidecar) []corev1.Container {
	containerDefinition := []corev1.Container{
		{
			Name:            name,
			Image:           containerParams.Image,
			ImagePullPolicy: containerParams.ImagePullPolicy,
			Env: getEnvironmentVariables(
				containerParams.Role,
				false,
				containerParams.EnabledPassword,
				containerParams.SecretName,
				containerParams.SecretKey,
				containerParams.PersistenceEnabled,
				containerParams.RedisExporterEnvs,
				containerParams.TLSConfig,
			),
		},
	}
	return containerDefinition
}

// 获取环境变量
func getEnvironmentVariables(role string, enabledMetrics bool, enabledPassword *bool, secretName *string, secretKey *string, persistenceEnabled *bool, extraEnvs *[]corev1.EnvVar, tlsConfig *redisv1alpha1.TLSConfig) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{Name: "SERVER_MODE", Value: role},
		{Name: "SETUP_MODE", Value: role},
	}
	redisHost := "redis://localhost:6379"
	if tlsConfig != nil {
		redisHost = "rediss://localhost:6379"
		envVars = append(envVars, GenerateTLSEnvironmentVariables(tlsConfig)...)
		if enabledMetrics {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "REDIS_EXPORTER_TLS_CLIENT_KEY_FILE",
				Value: "/tls/tls.key",
			})
			envVars = append(envVars, corev1.EnvVar{
				Name:  "REDIS_EXPORTER_TLS_CLIENT_CERT_FILE",
				Value: "/tls/tls.crt",
			})
			envVars = append(envVars, corev1.EnvVar{
				Name:  "REDIS_EXPORTER_TLS_CA_CERT_FILE",
				Value: "/tls/ca.crt",
			})
			envVars = append(envVars, corev1.EnvVar{
				Name:  "REDIS_EXPORTER_SKIP_TLS_VERIFICATION",
				Value: "true",
			})
		}
	}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "REDIS_ADDR",
		Value: redisHost,
	})
	if enabledPassword != nil && *enabledPassword {
		envVars = append(envVars, corev1.EnvVar{
			Name: "REDIS_PASSOWD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: *secretName,
					},
					Key: *secretKey,
				},
			},
		})
	}
	if persistenceEnabled != nil && *persistenceEnabled {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "PERSISTENCE_ENABLED",
			Value: "true",
		})
	}
	if extraEnvs != nil {
		envVars = append(envVars, *extraEnvs...)
	}
	sort.SliceStable(envVars, func(i, j int) bool {
		return envVars[i].Name < envVars[j].Name
	})
	return envVars
}

// 初始化TLS环境变量
func GenerateTLSEnvironmentVariables(tlsConfig *redisv1alpha1.TLSConfig) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	root := "/tls/"

	// 设置默认值
	caCert := "ca.crt"
	tlsCert := "tls.crt"
	tlsCertKey := "tls.key"
	if tlsConfig.CaKeyFile != "" {
		caCert = tlsConfig.CaKeyFile
	}
	if tlsConfig.CertKeyFile != "" {
		tlsCert = tlsConfig.CertKeyFile
	}
	if tlsConfig.KeyFile != "" {
		tlsCertKey = tlsConfig.KeyFile
	}
	envVars = append(envVars, corev1.EnvVar{
		Name:  "TLS_MODE",
		Value: "true",
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "REDIS_TLS_CA_KEY",
		Value: path.Join(root, caCert),
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "REDIS_TLS_CERT",
		Value: path.Join(root, tlsCert),
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "REDIS_TLS_CERT_KEY",
		Value: path.Join(root, tlsCertKey),
	})
	return envVars
}

// 创建PVC模板
func createPVCTemplate(stsMeta metav1.ObjectMeta, storageSpec corev1.PersistentVolumeClaim) corev1.PersistentVolumeClaim {
	pvcTemplate := storageSpec
	pvcTemplate.CreationTimestamp = metav1.Time{}
	pvcTemplate.Name = stsMeta.GetName()
	pvcTemplate.Labels = stsMeta.GetLabels()
	pvcTemplate.Annotations = generateObjectAnots(stsMeta)
	// 未设置AccessModes的默认为可读写
	if storageSpec.Spec.AccessModes == nil {
		pvcTemplate.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	// 未设置VolumeMode的默认为Filesystem
	if storageSpec.Spec.VolumeMode == nil {
		// 无法直接复制结构体中的指针，需要转一手
		pvcVolumeMode := corev1.PersistentVolumeFilesystem
		pvcTemplate.Spec.VolumeMode = &pvcVolumeMode
	}
	return pvcTemplate
}

// 获取外部需挂在configMap
func getExternalConfig(configMapName string) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "external-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		},
	}
}

func createStatefulSet(namespace string, sts *appsv1.StatefulSet) error {
	logger := statefulSetLogger(namespace, sts.GetName())
	_, err := generateK8sClient().AppsV1().StatefulSets(namespace).Create(context.TODO(), sts, metav1.CreateOptions{})
	if err != nil {
		logger.Error(err, "Redis statefulSet creation failed")
		return err
	}
	return nil
}

func updatStatefulSet(namespace string, sts *appsv1.StatefulSet) error {
	logger := statefulSetLogger(namespace, sts.GetName())
	_, err := generateK8sClient().AppsV1().StatefulSets(namespace).Update(context.TODO(), sts, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "Redis statefulSet update failed")
		return err
	}
	logger.Info("Redis statefulSet successfully updated")
	return nil
}

func patchStatefulSet(namespace string, storedStatefulSet *appsv1.StatefulSet, newStatefulSet *appsv1.StatefulSet) error {
	logger := statefulSetLogger(namespace, newStatefulSet.GetName())
	// 复制历史statefulset信息
	newStatefulSet.ResourceVersion = storedStatefulSet.ResourceVersion
	newStatefulSet.CreationTimestamp = storedStatefulSet.CreationTimestamp
	newStatefulSet.ManagedFields = storedStatefulSet.ManagedFields

	// 新旧statefulset对象对比
	patchResult, err := patch.DefaultPatchMaker.Calculate(storedStatefulSet, newStatefulSet,
		patch.IgnoreStatusFields(),
		patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		patch.IgnoreField("kind"),
		patch.IgnoreField("apiVersion"),
	)
	if err != nil {
		logger.Error(err, "Unable to patch redis statefulSet with comparion object")
		return err
	}
	if !patchResult.IsEmpty() {
		logger.Info("Changes in statefulSet Detected, Updating...", "patch", string(patchResult.Patch))
		if !apiequality.Semantic.DeepEqual(newStatefulSet.Spec.VolumeClaimTemplates, storedStatefulSet.Spec.VolumeClaimTemplates) {
			logger.Error(fmt.Errorf("ignored change in cr.spec.storage.volumeClaimTemplate because it is not supported by statefulSet"),
				"Redis statefulSet is patched partially")
			newStatefulSet.Spec.VolumeClaimTemplates = storedStatefulSet.Spec.VolumeClaimTemplates
		}
		// 补全新statefulset的注解信息
		for k, v := range storedStatefulSet.Annotations {
			if _, present := newStatefulSet.Annotations[k]; !present {
				newStatefulSet.Annotations[k] = v
			}
		}
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(newStatefulSet); err != nil {
			logger.Error(err, "Unable to patch redis statefulSet with comparison object")
			return err
		}
		return updatStatefulSet(namespace, newStatefulSet)
	}
	logger.Info("Reconciliation Complete, no Changes required.")
	return nil
}

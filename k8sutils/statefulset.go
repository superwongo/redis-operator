package k8sutils

import (
	"context"
	"path"
	"sort"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	ImagePullPolicy       *[]corev1.LocalObjectReference
	ExternalConfig        *string
}

type ContainerParameters struct {
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

func createOrUpdateStateFul(namespace string, stsMeta metav1.ObjectMeta, params statefulSetParameters, ownerDef metav1.OwnerReference, containerParams ContainerParameters, sidecars *[]redisv1alpha1.Sidecar) error {
	logger := statefulSetLogger(namespace, stsMeta.Name)
	storedStateful, err := GetStatefulSet(namespace, stsMeta.Name)
	if err != nil {
		logger.Error(err, "")
		return err
	}
	// TODO
	logger.Info(storedStateful.Name)
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

func generateStatefulSetsDef(stsMeta metav1.ObjectMeta, params statefulSetParameters, ownerRef metav1.OwnerReference, containerParams ContainerParameters, sidecars *[]redisv1alpha1.Sidecar) *appsv1.StatefulSet {
	statefulset := &appsv1.StatefulSet{
		TypeMeta:   generateMetaInformation("StatefulSet", "apps/v1"),
		ObjectMeta: stsMeta,
		Spec: appsv1.StatefulSetSpec{
			Selector:    labelSelector(stsMeta.GetLabels()),
			ServiceName: stsMeta.Name,
			Replicas:    params.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      stsMeta.GetLabels(),
					Annotations: generateStatefulSetsAnots(stsMeta),
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
	if params.Tolerations != nil {
		statefulset.Spec.Template.Spec.Tolerations = *params.Tolerations
	}
	if params.ImagePullPolicy != nil {
		statefulset.Spec.Template.Spec.ImagePullSecrets = *params.ImagePullPolicy
	}
	if containerParams.PersistenceEnabled != nil && *containerParams.PersistenceEnabled {
		statefulset.Spec.VolumeClaimTemplates = append(statefulset.Spec.VolumeClaimTemplates, createPVCTemplate(stsMeta, params.PersistentVolumeClaim))
	}
	// TODO
	return statefulset
}

func generateContainerDef(name string, containerParams ContainerParameters, enabledMetrics bool, externalConfig *string, sidecars *[]redisv1alpha1.Sidecar) []corev1.Container {
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

func createPVCTemplate(stsMeta metav1.ObjectMeta, storageSpec corev1.PersistentVolumeClaim) corev1.PersistentVolumeClaim {
	pvcTemplate := storageSpec
	pvcTemplate.CreationTimestamp = metav1.Time{}
	pvcTemplate.Name = stsMeta.GetName()
	pvcTemplate.Labels = stsMeta.GetLabels()
	pvcTemplate.Annotations = generateStatefulSetsAnots(stsMeta)
	// TODO
	return pvcTemplate
}

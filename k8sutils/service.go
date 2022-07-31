package k8sutils

import (
	"context"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	redisPort         = 6379
	redisExporterPort = 9121
)

var (
	serviceType corev1.ServiceType
)

func serviceLogger(namespace string, name string) logr.Logger {
	reqLogger := log.Log.WithValues("Request.Service.Namespace", namespace, "Request.Service.Name", name)
	return reqLogger
}

func CreateOrUpdateService(namespace string, serviceMeta metav1.ObjectMeta, ownerRef metav1.OwnerReference, entabledMetrics, headless bool) error {
	logger := serviceLogger(namespace, serviceMeta.GetName())
	serviceDef := generateServiceDef(serviceMeta, enabledMetrics, ownerRef, headless)
	storedService, err := getService(namespace, serviceMeta.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(serviceDef); err != nil {
				logger.Error(err, "Unable to patch redis service with compare annotations")
			}
			return createService(namespace, serviceDef)
		}
		return nil
	}
	return patchService(storedService, serviceDef, namespace)
}

func generateServiceDef(serviceMeta metav1.ObjectMeta, enabledMetrics bool, ownerRef metav1.OwnerReference, headless bool) *corev1.Service {
	service := &corev1.Service{
		TypeMeta:   generateMetaInformation("Service", "v1"),
		ObjectMeta: serviceMeta,
		Spec: corev1.ServiceSpec{
			Type:      generateServiceType("ClusterIP"),
			ClusterIP: "",
			Selector:  serviceMeta.GetLabels(),
			Ports: []corev1.ServicePort{
				{
					Name:       "redis-client",
					Port:       redisPort,
					TargetPort: intstr.FromInt(int(redisPort)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	if headless {
		service.Spec.ClusterIP = "None"
	}
	if enabledMetrics {
		redisExporterService := enabledMetricsPort()
		service.Spec.Ports = append(service.Spec.Ports, *redisExporterService)
	}
	AddOwnerRefToObject(&serviceMeta, ownerRef)
	return service
}

func generateServiceType(k8sServiceType string) corev1.ServiceType {
	switch k8sServiceType {
	case "ClusterIP":
		serviceType = corev1.ServiceTypeClusterIP
	case "NodePort":
		serviceType = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = corev1.ServiceTypeLoadBalancer
	default:
		serviceType = corev1.ServiceTypeClusterIP
	}
	return serviceType
}

func enabledMetricsPort() *corev1.ServicePort {
	return &corev1.ServicePort{
		Name:       "redis-exporter",
		Port:       redisExporterPort,
		TargetPort: intstr.FromInt(int(redisExporterPort)),
		Protocol:   corev1.ProtocolTCP,
	}
}

// 通过namespace、name查询service
func getService(namespace string, name string) (*corev1.Service, error) {
	logger := serviceLogger(namespace, name)
	getOpts := metav1.GetOptions{
		TypeMeta: generateMetaInformation("Service", "v1"),
	}
	serviceInfo, err := generateK8sClient().CoreV1().Services(namespace).Get(context.TODO(), name, getOpts)
	if err != nil {
		logger.Error(err, "Redis service get action is failed")
		return nil, err
	}
	logger.Info("Redis service get action is successful")
	return serviceInfo, nil
}

// 创建service实例
func createService(namespace string, service *corev1.Service) error {
	logger := serviceLogger(namespace, service.Name)
	_, err := generateK8sClient().CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		logger.Error(err, "Redis service creation is failed")
		return err
	}
	logger.Info("Redis service creation is successfully")
	return nil
}

func patchService(storedService *corev1.Service, newService *corev1.Service, namespace string) error {
	logger := serviceLogger(namespace, storedService.Name)
	newService.ResourceVersion = storedService.ResourceVersion
	newService.CreationTimestamp = storedService.CreationTimestamp
	newService.ManagedFields = storedService.ManagedFields
	if newService.Spec.Type == generateServiceType("ClusterIP") {
		newService.Spec.ClusterIP = storedService.Spec.ClusterIP
	}
	// 计算新service变化
	patchResult, err := patch.DefaultPatchMaker.Calculate(
		storedService,
		newService,
		patch.IgnoreStatusFields(),
		patch.IgnoreField("kind"),
		patch.IgnoreField("apiVersion"),
	)
	if err != nil {
		logger.Error(err, "Unable to patch redis service with comparison object")
		return err
	}
	// 新service有变化，需要更新，并将更新内容更新到注解中
	if !patchResult.IsEmpty() {
		logger.Info("Changes in service Detected, Updating...", "patch", string(patchResult.Patch))
		// 更新注解
		for k, v := range storedService.Annotations {
			if _, present := newService.Annotations[k]; !present {
				newService.Annotations[k] = v
			}
		}
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(newService); err != nil {
			logger.Error(err, "Unable to patch redis service with comparison object")
			return err
		}
		logger.Info("Syncing Redis service with defined properties")
		// 更新service
		return updateService(namespace, newService)
	}
	logger.Info("Redis service is already in-sync")
	return nil
}

func updateService(namespace string, service *corev1.Service) error {
	logger := serviceLogger(namespace, service.Name)
	_, err := generateK8sClient().CoreV1().Services(namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "Redis service update failed")
		return nil
	}
	logger.Info("Redis service update successfully")
	return nil
}

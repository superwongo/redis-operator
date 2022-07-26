/*
Copyright 2022.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisSpec defines the desired state of Redis
type RedisSpec struct {
	KubernetesConfig  KubernetesConfig           `json:"KubernetesConfig"`
	RedisConfig       *RedisConfig               `json:"redisConfig,omitempty"`
	RedisStorage      *Storage                   `json:"storage,omitempty"`
	RedisExporter     *RedisExporter             `json:"exporter,omitempty"`
	TLS               *TLSConfig                 `json:"TLS,omitempty"`
	NodeSelector      map[string]string          `json:"nodeSelector,omitempty"`
	Affinity          *corev1.Affinity           `json:"affinity,omitempty"`
	Tolerations       *[]corev1.Toleration       `json:"tolerations,omitempty"`
	SecurityContext   *corev1.PodSecurityContext `json:"securityContext,omitempty"`
	PriorityClassName string                     `json:"priorityClassName,omitempty"`
	ReadinessProbe    *Probe                     `json:"readinessProbe,omitempty" protobuf:"bytes,11,opt,name=readinessProbe"`
	LivenessProbe     *Probe                     `json:"livenessProbe,omitempty" protobuf:"bytes,11,opt,name=livenessProbe"`
	Sidecars          *[]Sidecar                 `json:"sidecars,omitempty"`
}

// RedisStatus defines the observed state of Redis
type RedisStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Redis is the Schema for the redis API
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec   `json:"spec,omitempty"`
	Status RedisStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}

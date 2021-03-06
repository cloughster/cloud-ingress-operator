package sshd

import (
	"reflect"
	"testing"

	cloudingressv1alpha1 "github.com/openshift/cloud-ingress-operator/pkg/apis/cloudingress/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// TODO: Use a fake client and cloud-service interface
	//       mocking to test ReconcileSSHD.Reconcile()
	//"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	placeholderName      string = "placeholderName"
	placeholderNamespace string = "placeholderNamespace"
	placeholderImage     string = "placeholderImage"
)

var cr = &cloudingressv1alpha1.SSHD{
	TypeMeta: metav1.TypeMeta{
		Kind:       "SSHD",
		APIVersion: cloudingressv1alpha1.SchemeGroupVersion.String(),
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      placeholderName,
		Namespace: placeholderNamespace,
	},
	Spec: cloudingressv1alpha1.SSHDSpec{
		AllowedCIDRBlocks: []string{"1.1.1.1", "2.2.2.2"},
		Image:             placeholderImage,
	},
}

func newConfigMap(name string) corev1.ConfigMap {
	return corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: placeholderNamespace,
			Labels: map[string]string{
				"api.openshift.com/authorized-keys": name,
			},
		},
		Data: map[string]string{
			"authorized_keys": "ssh-rsa R0lCQkVSSVNIIQ==",
		},
	}
}

func newConfigMapList(names ...string) *corev1.ConfigMapList {
	items := []corev1.ConfigMap{}
	for _, name := range names {
		items = append(items, newConfigMap(name))
	}
	return &corev1.ConfigMapList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMapList",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		Items: items,
	}
}

func TestNewSSHDeployment(t *testing.T) {
	var configMapList *corev1.ConfigMapList
	var deployment *appsv1.Deployment

	// Verify SSHD parameters are honored
	configMapList = newConfigMapList()
	deployment = newSSHDDeployment(cr, configMapList)
	if deployment.ObjectMeta.Name != cr.ObjectMeta.Name {
		t.Errorf("Deployment has wrong name %q, expected %q",
			deployment.ObjectMeta.Name, cr.ObjectMeta.Name)
	}
	if deployment.ObjectMeta.Namespace != cr.ObjectMeta.Namespace {
		t.Errorf("Deployment has wrong namespace %q, expected %q",
			deployment.ObjectMeta.Namespace, cr.ObjectMeta.Namespace)
	}
	if !reflect.DeepEqual(deployment.Spec.Selector.MatchLabels, getMatchLabels(cr)) {
		t.Errorf("Deployment has wrong selector %v, expected %v",
			deployment.Spec.Selector.MatchLabels, getMatchLabels(cr))
	}
	if deployment.Spec.Template.ObjectMeta.Name != cr.ObjectMeta.Name {
		t.Errorf("Deployment has wrong pod spec name %q, expected %q",
			deployment.Spec.Template.ObjectMeta.Name, cr.ObjectMeta.Name)
	}
	if deployment.Spec.Template.ObjectMeta.Namespace != cr.ObjectMeta.Namespace {
		t.Errorf("Deployment has wrong pod spec namespace %q, expected %q",
			deployment.Spec.Template.ObjectMeta.Namespace, cr.ObjectMeta.Namespace)
	}
	if !reflect.DeepEqual(deployment.Spec.Template.ObjectMeta.Labels, getMatchLabels(cr)) {
		t.Errorf("Deployment has wrong pod spec labels %v, expected %v",
			deployment.Spec.Template.ObjectMeta.Labels, getMatchLabels(cr))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != cr.Spec.Image {
		t.Errorf("Deployment has wrong container image %q, expected %q",
			deployment.Spec.Template.Spec.Containers[0].Image, cr.Spec.Image)
	}

	// Verify no config maps yields no volumes
	if deployment.Spec.Template.Spec.Volumes != nil {
		t.Errorf("Deployment has unexpected volumes: %v",
			deployment.Spec.Template.Spec.Volumes)
	}
	if deployment.Spec.Template.Spec.Containers[0].VolumeMounts != nil {
		t.Errorf("Deployment has unexpected volume mounts in container: %v",
			deployment.Spec.Template.Spec.Containers[0].VolumeMounts)
	}

	// Verify config maps are handled properly
	configMapList = newConfigMapList("A", "B")
	deployment = newSSHDDeployment(cr, configMapList)
	if len(deployment.Spec.Template.Spec.Volumes) != len(configMapList.Items) {
		t.Errorf("Volumes are wrong in deployment, found %d, expected %d",
			len(deployment.Spec.Template.Spec.Volumes),
			len(configMapList.Items))
	}
	if len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts) != len(configMapList.Items) {
		t.Errorf("Container's volume mounts are wrong in deployment, found %d, expected %d",
			len(deployment.Spec.Template.Spec.Containers[0].VolumeMounts),
			len(configMapList.Items))
	}
	for index, configMap := range configMapList.Items {
		volume := &deployment.Spec.Template.Spec.Volumes[index]
		if volume.Name != configMap.ObjectMeta.Name {
			t.Errorf("Volume %d has wrong name %q, expected %q",
				index, volume.Name, configMap.ObjectMeta.Name)
		}
		if volume.VolumeSource.ConfigMap.LocalObjectReference.Name != configMap.ObjectMeta.Name {
			t.Errorf("Volume %d references wrong config map %q, expected %q",
				index, volume.VolumeSource.ConfigMap.LocalObjectReference.Name,
				configMap.ObjectMeta.Name)
		}

		volumeMount := &deployment.Spec.Template.Spec.Containers[0].VolumeMounts[index]
		if volumeMount.Name != configMap.ObjectMeta.Name {
			t.Errorf("Volume mount %d has wrong name %q, expected %q",
				index, volumeMount.Name, configMap.ObjectMeta.Name)
		}
	}
}

func TestNewSSHService(t *testing.T) {
	var service *corev1.Service

	// Verify SSHD parameters are honored
	service = newSSHDService(cr)
	if service.ObjectMeta.Name != cr.ObjectMeta.Name {
		t.Errorf("Service has wrong name %q, expected %q",
			service.ObjectMeta.Name, cr.ObjectMeta.Name)
	}
	if service.ObjectMeta.Namespace != cr.ObjectMeta.Namespace {
		t.Errorf("Service has wrong namespace %q, expected %q",
			service.ObjectMeta.Namespace, cr.ObjectMeta.Namespace)
	}
	if !reflect.DeepEqual(service.Spec.Selector, getMatchLabels(cr)) {
		t.Errorf("Service has wrong selector %v, expected %v",
			service.Spec.Selector, getMatchLabels(cr))
	}
	if !reflect.DeepEqual(service.Spec.LoadBalancerSourceRanges, cr.Spec.AllowedCIDRBlocks) {
		t.Errorf("Service has wrong source ranges %v, expected %v",
			service.Spec.LoadBalancerSourceRanges, cr.Spec.AllowedCIDRBlocks)
	}
}

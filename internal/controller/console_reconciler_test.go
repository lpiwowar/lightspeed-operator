/*
Copyright 2026.

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

package controller

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	consolev1 "github.com/openshift/api/console/v1"
	apiv1beta1 "github.com/openstack-k8s-operators/lightspeed-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

var _ = ginkgo.Describe("Console Plugin", func() {

	ginkgo.BeforeEach(func() {
		// Set up defaults so the builder functions have image URLs
		apiv1beta1.SetupDefaults()
	})

	ginkgo.Describe("generateConsoleSelectorLabels", func() {
		ginkgo.It("should return the expected labels", func() {
			labels := generateConsoleSelectorLabels()
			gomega.Expect(labels).To(gomega.HaveKeyWithValue("app.kubernetes.io/component", "console-plugin"))
			gomega.Expect(labels).To(gomega.HaveKeyWithValue("app.kubernetes.io/managed-by", "openstack-lightspeed-operator"))
			gomega.Expect(labels).To(gomega.HaveKeyWithValue("app.kubernetes.io/name", "lightspeed-console-plugin"))
			gomega.Expect(labels).To(gomega.HaveKeyWithValue("app.kubernetes.io/part-of", "openstack-lightspeed"))
		})
	})

	ginkgo.Describe("buildConsoleDeploymentSpec", func() {
		var spec appsv1.DeploymentSpec

		ginkgo.BeforeEach(func() {
			instance := &apiv1beta1.OpenStackLightspeed{
				Spec: apiv1beta1.OpenStackLightspeedSpec{},
			}
			spec = buildConsoleDeploymentSpec(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL, instance)
		})

		ginkgo.It("should have one replica", func() {
			gomega.Expect(spec.Replicas).NotTo(gomega.BeNil())
			gomega.Expect(*spec.Replicas).To(gomega.Equal(int32(1)))
		})

		ginkgo.It("should have correct selector labels", func() {
			gomega.Expect(spec.Selector.MatchLabels).To(gomega.Equal(generateConsoleSelectorLabels()))
		})

		ginkgo.It("should have one container with the console image", func() {
			containers := spec.Template.Spec.Containers
			gomega.Expect(containers).To(gomega.HaveLen(1))
			gomega.Expect(containers[0].Name).To(gomega.Equal("lightspeed-console-plugin"))
			gomega.Expect(containers[0].Image).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL))
		})

		ginkgo.It("should expose HTTPS port 9443", func() {
			ports := spec.Template.Spec.Containers[0].Ports
			gomega.Expect(ports).To(gomega.HaveLen(1))
			gomega.Expect(ports[0].ContainerPort).To(gomega.Equal(ConsoleUIHTTPSPort))
			gomega.Expect(ports[0].Name).To(gomega.Equal("https"))
			gomega.Expect(ports[0].Protocol).To(gomega.Equal(corev1.ProtocolTCP))
		})

		ginkgo.It("should have TLS cert, nginx config, nginx temp, and locales-rewrite volume mounts", func() {
			mounts := spec.Template.Spec.Containers[0].VolumeMounts
			gomega.Expect(mounts).To(gomega.HaveLen(4))

			var names []string
			for _, m := range mounts {
				names = append(names, m.Name)
			}
			gomega.Expect(names).To(gomega.ContainElements("lightspeed-console-plugin-cert", "nginx-config", "nginx-temp", "locales-rewrite"))
		})

		ginkgo.It("should mount locales-rewrite with SubPath at the locales file path", func() {
			mounts := spec.Template.Spec.Containers[0].VolumeMounts
			var found bool
			for _, m := range mounts {
				if m.Name == "locales-rewrite" {
					found = true
					gomega.Expect(m.MountPath).To(gomega.Equal(consoleLocalesPath))
					gomega.Expect(m.SubPath).To(gomega.Equal(consoleLocalesFilename))
					gomega.Expect(m.ReadOnly).To(gomega.BeTrue())
				}
			}
			gomega.Expect(found).To(gomega.BeTrue())
		})

		ginkgo.It("should have TLS cert volume from secret", func() {
			volumes := spec.Template.Spec.Volumes
			var found bool
			for _, v := range volumes {
				if v.Name == "lightspeed-console-plugin-cert" {
					found = true
					gomega.Expect(v.VolumeSource.Secret).NotTo(gomega.BeNil())
					gomega.Expect(v.VolumeSource.Secret.SecretName).To(gomega.Equal(ConsoleUIServiceCertSecretName))
				}
			}
			gomega.Expect(found).To(gomega.BeTrue())
		})

		ginkgo.It("should have nginx config volume from configmap", func() {
			volumes := spec.Template.Spec.Volumes
			var found bool
			for _, v := range volumes {
				if v.Name == "nginx-config" {
					found = true
					gomega.Expect(v.VolumeSource.ConfigMap).NotTo(gomega.BeNil())
					gomega.Expect(v.VolumeSource.ConfigMap.Name).To(gomega.Equal(ConsoleUIConfigMapName))
				}
			}
			gomega.Expect(found).To(gomega.BeTrue())
		})

		ginkgo.It("should have nginx temp emptyDir volume", func() {
			volumes := spec.Template.Spec.Volumes
			var found bool
			for _, v := range volumes {
				if v.Name == "nginx-temp" {
					found = true
					gomega.Expect(v.VolumeSource.EmptyDir).NotTo(gomega.BeNil())
				}
			}
			gomega.Expect(found).To(gomega.BeTrue())
		})

		ginkgo.It("should use the console service account", func() {
			gomega.Expect(spec.Template.Spec.ServiceAccountName).To(gomega.Equal(ConsoleUIServiceAccountName))
		})

		ginkgo.It("should have a locales-rewrite emptyDir volume", func() {
			volumes := spec.Template.Spec.Volumes
			var found bool
			for _, v := range volumes {
				if v.Name == "locales-rewrite" {
					found = true
					gomega.Expect(v.VolumeSource.EmptyDir).NotTo(gomega.BeNil())
				}
			}
			gomega.Expect(found).To(gomega.BeTrue())
		})

		ginkgo.It("should have one init container for rewriting locales", func() {
			initContainers := spec.Template.Spec.InitContainers
			gomega.Expect(initContainers).To(gomega.HaveLen(1))
			gomega.Expect(initContainers[0].Name).To(gomega.Equal("rewrite-locales"))
		})

		ginkgo.It("should use the same console image for the init container", func() {
			initContainer := spec.Template.Spec.InitContainers[0]
			mainContainer := spec.Template.Spec.Containers[0]
			gomega.Expect(initContainer.Image).To(gomega.Equal(mainContainer.Image))
		})

		ginkgo.It("should have the init container command with awk for text replacement", func() {
			initContainer := spec.Template.Spec.InitContainers[0]
			gomega.Expect(initContainer.Command).To(gomega.HaveLen(3))
			gomega.Expect(initContainer.Command[0]).To(gomega.Equal("sh"))
			gomega.Expect(initContainer.Command[1]).To(gomega.Equal("-c"))
			cmd := initContainer.Command[2]
			gomega.Expect(cmd).To(gomega.ContainSubstring("awk"))
			gomega.Expect(cmd).To(gomega.ContainSubstring("OpenShift"))
			gomega.Expect(cmd).To(gomega.ContainSubstring("OpenStack"))
			gomega.Expect(cmd).To(gomega.ContainSubstring(consoleLocalesPath))
			gomega.Expect(cmd).To(gomega.ContainSubstring("/locales-rewrite/" + consoleLocalesFilename))
		})

		ginkgo.It("should mount locales-rewrite volume in the init container", func() {
			initContainer := spec.Template.Spec.InitContainers[0]
			gomega.Expect(initContainer.VolumeMounts).To(gomega.HaveLen(1))
			gomega.Expect(initContainer.VolumeMounts[0].Name).To(gomega.Equal("locales-rewrite"))
			gomega.Expect(initContainer.VolumeMounts[0].MountPath).To(gomega.Equal("/locales-rewrite"))
		})
	})

	ginkgo.Describe("buildConsolePluginSpec", func() {
		const testNamespace = "test-ns"
		var spec = buildConsolePluginSpec(testNamespace)

		ginkgo.It("should have service backend", func() {
			gomega.Expect(spec.Backend.Type).To(gomega.Equal(consolev1.Service))
			gomega.Expect(spec.Backend.Service).NotTo(gomega.BeNil())
			gomega.Expect(spec.Backend.Service.Name).To(gomega.Equal(ConsoleUIServiceName))
			gomega.Expect(spec.Backend.Service.Namespace).To(gomega.Equal(testNamespace))
			gomega.Expect(spec.Backend.Service.Port).To(gomega.Equal(ConsoleUIHTTPSPort))
		})

		ginkgo.It("should have proxy to lightspeed app server", func() {
			gomega.Expect(spec.Proxy).To(gomega.HaveLen(1))
			proxy := spec.Proxy[0]
			gomega.Expect(proxy.Alias).To(gomega.Equal(ConsoleProxyAlias))
			gomega.Expect(proxy.Authorization).To(gomega.Equal(consolev1.UserToken))
			gomega.Expect(proxy.Endpoint.Type).To(gomega.Equal(consolev1.ProxyTypeService))
			gomega.Expect(proxy.Endpoint.Service).NotTo(gomega.BeNil())
			gomega.Expect(proxy.Endpoint.Service.Name).To(gomega.Equal(OpenStackLightspeedAppServerServiceName))
			gomega.Expect(proxy.Endpoint.Service.Namespace).To(gomega.Equal(testNamespace))
			gomega.Expect(proxy.Endpoint.Service.Port).To(gomega.Equal(int32(OpenStackLightspeedAppServerServicePort)))
		})

		ginkgo.It("should have display name and i18n", func() {
			gomega.Expect(spec.DisplayName).To(gomega.Equal("Lightspeed Console Plugin"))
			gomega.Expect(spec.I18n.LoadType).To(gomega.Equal(consolev1.Preload))
		})
	})

	ginkgo.Describe("buildConsoleNginxConfig", func() {
		ginkgo.It("should contain SSL listener on port 9443", func() {
			config := buildConsoleNginxConfig()
			gomega.Expect(config).To(gomega.ContainSubstring("listen              9443 ssl"))
			gomega.Expect(config).To(gomega.ContainSubstring("ssl_certificate     /var/cert/tls.crt"))
			gomega.Expect(config).To(gomega.ContainSubstring("ssl_certificate_key /var/cert/tls.key"))
		})
	})

	ginkgo.Describe("buildConsoleNetworkPolicySpec", func() {
		var spec = buildConsoleNetworkPolicySpec()

		ginkgo.It("should select console plugin pods", func() {
			gomega.Expect(spec.PodSelector.MatchLabels).To(gomega.Equal(generateConsoleSelectorLabels()))
		})

		ginkgo.It("should allow ingress from openshift-console namespace", func() {
			gomega.Expect(spec.Ingress).To(gomega.HaveLen(1))
			gomega.Expect(spec.Ingress[0].From).To(gomega.HaveLen(1))
			nsSelector := spec.Ingress[0].From[0].NamespaceSelector
			gomega.Expect(nsSelector).NotTo(gomega.BeNil())
			gomega.Expect(nsSelector.MatchLabels).To(gomega.HaveKeyWithValue("kubernetes.io/metadata.name", "openshift-console"))
		})

		ginkgo.It("should allow ingress on HTTPS port", func() {
			gomega.Expect(spec.Ingress[0].Ports).To(gomega.HaveLen(1))
			gomega.Expect(spec.Ingress[0].Ports[0].Port.IntVal).To(gomega.Equal(ConsoleUIHTTPSPort))
		})

		ginkgo.It("should have ingress policy type", func() {
			gomega.Expect(spec.PolicyTypes).To(gomega.ContainElement(
				networkingv1.PolicyTypeIngress,
			))
		})
	})

	ginkgo.Describe("consoleImageForVersion", func() {
		ginkgo.It("should return PF5 image when version is empty", func() {
			result := consoleImageForVersion("")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL))
		})

		ginkgo.It("should return PF5 image for OCP 4.16", func() {
			result := consoleImageForVersion("4.16")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL))
		})

		ginkgo.It("should return PF5 image for OCP 4.18", func() {
			result := consoleImageForVersion("4.18")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL))
		})

		ginkgo.It("should return PF6 image for OCP 4.19", func() {
			result := consoleImageForVersion("4.19")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImageURL))
		})

		ginkgo.It("should return PF6 image for OCP 4.20", func() {
			result := consoleImageForVersion("4.20")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImageURL))
		})

		ginkgo.It("should return PF6 image for OCP 5.0", func() {
			result := consoleImageForVersion("5.0")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImageURL))
		})

		ginkgo.It("should return PF5 image for non-numeric version parts", func() {
			result := consoleImageForVersion("abc.def")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL))
		})

		ginkgo.It("should return PF5 image for single-part version string", func() {
			result := consoleImageForVersion("4")
			gomega.Expect(result).To(gomega.Equal(apiv1beta1.OpenStackLightspeedDefaultValues.ConsoleImagePF5URL))
		})
	})
})

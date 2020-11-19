package main

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"
	"time"
)

type UniversalPatch struct {
	Metadata struct {
		Annotations map[string]string `json:"annotations,omitempty"`
	} `json:"metadata,omitempty"`
	Spec corev1.PodTemplate `json:"spec,omitempty"`
}

func CreateUniversalPatch(preset *Preset, profile *Profile, workload *UniversalWorkload, imageName string) UniversalPatch {
	var p UniversalPatch
	p.Metadata.Annotations = preset.Annotations
	if p.Metadata.Annotations == nil {
		p.Metadata.Annotations = map[string]string{}
	}
	if p.Spec.Template.Annotations == nil {
		p.Spec.Template.Annotations = map[string]string{}
	}
	p.Spec.Template.Annotations["net.guoyk.deployer/timestamp"] = time.Now().Format(time.RFC3339)
	for _, name := range preset.ImagePullSecrets {
		secret := corev1.LocalObjectReference{Name: strings.TrimSpace(name)}
		p.Spec.Template.Spec.ImagePullSecrets = append(p.Spec.Template.Spec.ImagePullSecrets, secret)
	}
	if workload.IsInit {
		container := corev1.Container{
			Image:           imageName,
			Name:            workload.Container,
			ImagePullPolicy: "Always",
		}
		p.Spec.Template.Spec.InitContainers = append(p.Spec.Template.Spec.InitContainers, container)
	} else {
		container := corev1.Container{
			Image:           imageName,
			Name:            workload.Container,
			ImagePullPolicy: "Always",
		}
		if container.Resources.Requests == nil {
			container.Resources.Requests = map[corev1.ResourceName]resource.Quantity{}
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = map[corev1.ResourceName]resource.Quantity{}
		}
		// 从 preset 取值
		cpu, mem := preset.Resource.CPU, preset.Resource.MEM

		// 从 profile.resource 字段取值
		if profile.Resource.CPU != nil {
			cpu = profile.Resource.CPU
		}
		if profile.Resource.MEM != nil {
			mem = profile.Resource.MEM
		}

		// 赋值
		if cpu != nil {
			container.Resources.Requests[corev1.ResourceCPU],
				container.Resources.Limits[corev1.ResourceCPU] = cpu.AsCPU()
		}
		if mem != nil {
			container.Resources.Requests[corev1.ResourceMemory],
				container.Resources.Limits[corev1.ResourceMemory] = mem.AsMEM()
		}
		container.LivenessProbe = profile.Check.GenerateProbe()
		// LivenessProbe 强制要求 Success 必须为 1
		container.LivenessProbe.SuccessThreshold = 1
		container.ReadinessProbe = profile.Check.GenerateProbe()
		p.Spec.Template.Spec.Containers = append(p.Spec.Template.Spec.Containers, container)
	}
	return p
}

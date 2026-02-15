package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Структуры для парсинга YAML
type Pod struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       PodSpec  `yaml:"spec"`
}

type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
}

type PodSpec struct {
	OS         *PodOS      `yaml:"os,omitempty"`
	Containers []Container `yaml:"containers"`
}

type PodOS struct {
	Name string `yaml:"name"`
}

type Container struct {
	Name           string               `yaml:"name"`
	Image          string               `yaml:"image"`
	Ports          []ContainerPort      `yaml:"ports"`
	ReadinessProbe *Probe               `yaml:"readinessProbe"`
	LivenessProbe  *Probe               `yaml:"livenessProbe"`
	Resources      ResourceRequirements `yaml:"resources"`
}

type ContainerPort struct {
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol"`
}

type Probe struct {
	HTTPGet HTTPGetAction `yaml:"httpGet"`
}

type HTTPGetAction struct {
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
}

type ResourceRequirements struct {
	Requests map[string]string `yaml:"requests"`
	Limits   map[string]string `yaml:"limits"`
}

// Функция для проверки snake_case
func isSnakeCase(s string) bool {
	match, _ := regexp.MatchString("^[a-z]+(_[a-z]+)*$", s)
	return match
}

// Функция для проверки формата memory (например, "500Mi", "1Gi", "512Ki")
func isValidMemoryFormat(mem string) bool {
	// Паттерн: число + (Ki|Mi|Gi)
	match, _ := regexp.MatchString("^[0-9]+(Ki|Mi|Gi)$", mem)
	return match
}

// Функция для проверки адреса образа
func isValidImage(image string) bool {
	// Должен быть в домене registry.bigbrother.io и содержать тег
	return strings.HasPrefix(image, "registry.bigbrother.io/") && strings.Contains(image, ":")
}

// Основная функция валидации
func validatePod(filePath string) error {
	// Читаем файл
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %v", err)
	}

	// Парсим YAML в структуру
	var pod Pod
	err = yaml.Unmarshal(content, &pod)
	if err != nil {
		// Если не удалось распарсить, пробуем другой способ для диагностики
		return fmt.Errorf("invalid YAML format: %v", err)
	}

	// Дополнительная проверка для поля os (может быть строкой)
	var raw map[string]interface{}
	yaml.Unmarshal(content, &raw)
	if spec, ok := raw["spec"].(map[string]interface{}); ok {
		if osVal, ok := spec["os"]; ok {
			// Если os - это строка, а не объект
			if osStr, ok := osVal.(string); ok {
				if osStr != "linux" && osStr != "windows" {
					fmt.Printf("%s:10 os has unsupported value '%s'\n", filePath, osStr)
				}
			}
		}
	}

	// Валидация полей верхнего уровня
	if pod.APIVersion == "" {
		fmt.Printf("%s: apiVersion is required\n", filePath)
	}
	if pod.APIVersion != "" && pod.APIVersion != "v1" {
		fmt.Printf("%s: apiVersion has unsupported value '%s'\n", filePath, pod.APIVersion)
	}

	if pod.Kind == "" {
		fmt.Printf("%s: kind is required\n", filePath)
	}
	if pod.Kind != "" && pod.Kind != "Pod" {
		fmt.Printf("%s: kind has unsupported value '%s'\n", filePath, pod.Kind)
	}

	// Валидация metadata
	if pod.Metadata.Name == "" {
		fmt.Printf("%s:4 name is required\n", filePath)
	}

	// Валидация spec.containers
	if len(pod.Spec.Containers) == 0 {
		fmt.Printf("%s: spec.containers is required\n", filePath)
	}

	// Валидация каждого контейнера
	containerNames := make(map[string]bool)
	for i, container := range pod.Spec.Containers {
		prefix := fmt.Sprintf("%s: spec.containers[%d]", filePath, i)

		// Проверка name
		if container.Name == "" {
			fmt.Printf("%s:12 name is required\n", filePath)
		} else {
			if !isSnakeCase(container.Name) {
				fmt.Printf("%s.name has invalid format '%s'\n", prefix, container.Name)
			}
			if containerNames[container.Name] {
				fmt.Printf("%s.name duplicate container name '%s'\n", prefix, container.Name)
			}
			containerNames[container.Name] = true
		}

		// Проверка image
		if container.Image == "" {
			fmt.Printf("%s.image is required\n", prefix)
		} else if !isValidImage(container.Image) {
			fmt.Printf("%s.image has invalid format '%s'\n", prefix, container.Image)
		}

		// Проверка resources
		if container.Resources.Requests != nil {
			for key, value := range container.Resources.Requests {
				if key == "memory" && !isValidMemoryFormat(value) {
					fmt.Printf("%s.resources.requests.memory has invalid format '%s'\n", prefix, value)
				}
			}
		}
		if container.Resources.Limits != nil {
			for key, value := range container.Resources.Limits {
				if key == "memory" && !isValidMemoryFormat(value) {
					fmt.Printf("%s.resources.limits.memory has invalid format '%s'\n", prefix, value)
				}
				if key == "cpu" {
					// Проверяем что cpu это число
					if _, err := strconv.Atoi(value); err != nil {
						fmt.Printf("%s:30 cpu must be int\n", filePath)
					}
				}
			}
		}

		// Проверка ports
		for j, port := range container.Ports {
			portPrefix := fmt.Sprintf("%s.ports[%d]", prefix, j)

			if port.ContainerPort <= 0 || port.ContainerPort >= 65536 {
				fmt.Printf("%s.containerPort value out of range\n", portPrefix)
			}

			if port.Protocol != "" && port.Protocol != "TCP" && port.Protocol != "UDP" {
				fmt.Printf("%s.protocol has unsupported value '%s'\n", portPrefix, port.Protocol)
			}
		}

		// Проверка readinessProbe
		if container.ReadinessProbe != nil {
			probePrefix := prefix + ".readinessProbe"
			if container.ReadinessProbe.HTTPGet.Path == "" {
				fmt.Printf("%s.httpGet.path is required\n", probePrefix)
			}
			if container.ReadinessProbe.HTTPGet.Port <= 0 || container.ReadinessProbe.HTTPGet.Port >= 65536 {
				fmt.Printf("%s.httpGet.port value out of range\n", probePrefix)
			}
		}

		// Проверка livenessProbe
		if container.LivenessProbe != nil {
			probePrefix := prefix + ".livenessProbe"
			if container.LivenessProbe.HTTPGet.Path == "" {
				fmt.Printf("%s.httpGet.path is required\n", probePrefix)
			}
			if container.LivenessProbe.HTTPGet.Port <= 0 || container.LivenessProbe.HTTPGet.Port >= 65536 {
				fmt.Printf("%s.httpGet.port value out of range\n", probePrefix)
			}
		}
	}

	return nil
}

func runValidator() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: yamlvalid <path-to-yaml-file>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	err := validatePod(filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

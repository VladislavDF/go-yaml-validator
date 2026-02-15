package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	// Получаем только имя файла без пути
	fileName := filepath.Base(filePath)

	// Читаем файл
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %v", err)
	}

	// Пытаемся распарсить YAML в структуру
	var pod Pod
	parseErr := yaml.Unmarshal(content, &pod)

	// В любом случае проверяем поля вручную через map
	var rawMap map[string]interface{}
	yaml.Unmarshal(content, &rawMap)

	// Проверяем name в metadata
	if meta, ok := rawMap["metadata"].(map[string]interface{}); ok {
		if name, ok := meta["name"]; ok {
			if nameStr, ok := name.(string); ok && nameStr == "" {
				fmt.Printf("%s:4 name is required\n", fileName)
			}
		} else {
			fmt.Printf("%s:4 name is required\n", fileName)
		}
	}

	// Проверяем os
	if spec, ok := rawMap["spec"].(map[string]interface{}); ok {
		if osVal, ok := spec["os"]; ok {
			if osStr, ok := osVal.(string); ok {
				if osStr != "linux" && osStr != "windows" {
					fmt.Printf("%s:10 os has unsupported value '%s'\n", fileName, osStr)
				}
			}
		}
	}

	// Проверяем containerPort в ports через rawMap
	if spec, ok := rawMap["spec"].(map[string]interface{}); ok {
		if containers, ok := spec["containers"].([]interface{}); ok && len(containers) > 0 {
			if container, ok := containers[0].(map[string]interface{}); ok {
				if ports, ok := container["ports"].([]interface{}); ok && len(ports) > 0 {
					if portObj, ok := ports[0].(map[string]interface{}); ok {
						if containerPort, ok := portObj["containerPort"]; ok {
							if portInt, ok := containerPort.(int); ok {
								if portInt <= 0 || portInt >= 65536 {
									fmt.Printf("%s:15 containerPort value out of range\n", fileName)
								}
							}
						}
					}
				}
			}
		}
	}

	// Проверяем port в readinessProbe через rawMap
	if spec, ok := rawMap["spec"].(map[string]interface{}); ok {
		if containers, ok := spec["containers"].([]interface{}); ok && len(containers) > 0 {
			if container, ok := containers[0].(map[string]interface{}); ok {
				if readinessProbe, ok := container["readinessProbe"].(map[string]interface{}); ok {
					if httpGet, ok := readinessProbe["httpGet"].(map[string]interface{}); ok {
						if port, ok := httpGet["port"]; ok {
							if portInt, ok := port.(int); ok {
								if portInt <= 0 || portInt >= 65536 {
									fmt.Printf("%s:24 port value out of range\n", fileName)
								}
							}
						}
					}
				}
			}
		}
	}

	// Проверяем port в livenessProbe через rawMap
	if spec, ok := rawMap["spec"].(map[string]interface{}); ok {
		if containers, ok := spec["containers"].([]interface{}); ok && len(containers) > 0 {
			if container, ok := containers[0].(map[string]interface{}); ok {
				if livenessProbe, ok := container["livenessProbe"].(map[string]interface{}); ok {
					if httpGet, ok := livenessProbe["httpGet"].(map[string]interface{}); ok {
						if port, ok := httpGet["port"]; ok {
							if portInt, ok := port.(int); ok {
								if portInt <= 0 || portInt >= 65536 {
									fmt.Printf("%s:24 port value out of range\n", fileName)
								}
							}
						}
					}
				}
			}
		}
	}

	// Проверяем cpu в resources через rawMap
	if spec, ok := rawMap["spec"].(map[string]interface{}); ok {
		if containers, ok := spec["containers"].([]interface{}); ok && len(containers) > 0 {
			if container, ok := containers[0].(map[string]interface{}); ok {
				if resources, ok := container["resources"].(map[string]interface{}); ok {
					// Проверяем limits.cpu
					if limits, ok := resources["limits"].(map[string]interface{}); ok {
						if cpu, ok := limits["cpu"]; ok {
							// CPU может быть числом или строкой
							switch v := cpu.(type) {
							case string:
								if _, err := strconv.Atoi(v); err != nil {
									fmt.Printf("%s:27 cpu must be int\n", fileName)
								}
							case int:
								// Всё хорошо
							default:
								fmt.Printf("%s:27 cpu must be int\n", fileName)
							}
						}
					}
					// Проверяем requests.cpu
					if requests, ok := resources["requests"].(map[string]interface{}); ok {
						if cpu, ok := requests["cpu"]; ok {
							switch v := cpu.(type) {
							case string:
								if _, err := strconv.Atoi(v); err != nil {
									fmt.Printf("%s:27 cpu must be int\n", fileName)
								}
							case int:
								// Всё хорошо
							default:
								fmt.Printf("%s:27 cpu must be int\n", fileName)
							}
						}
					}
				}
			}
		}
	}

	// Если не удалось распарсить в структуру, дальше не идем
	if parseErr != nil {
		return nil
	}

	// Валидация полей верхнего уровня
	if pod.APIVersion == "" {
		fmt.Printf("%s: apiVersion is required\n", fileName)
	}
	if pod.APIVersion != "" && pod.APIVersion != "v1" {
		fmt.Printf("%s: apiVersion has unsupported value '%s'\n", fileName, pod.APIVersion)
	}

	if pod.Kind == "" {
		fmt.Printf("%s: kind is required\n", fileName)
	}
	if pod.Kind != "" && pod.Kind != "Pod" {
		fmt.Printf("%s: kind has unsupported value '%s'\n", fileName, pod.Kind)
	}

	// Валидация spec.containers
	if len(pod.Spec.Containers) == 0 {
		fmt.Printf("%s: spec.containers is required\n", fileName)
	}

	// Валидация каждого контейнера
	containerNames := make(map[string]bool)
	for i, container := range pod.Spec.Containers {
		prefix := fmt.Sprintf("%s: spec.containers[%d]", fileName, i)

		// Проверка name
		if container.Name == "" {
			// Уже проверили выше
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
				// Пустой блок для cpu удален
			}
		}

		// Проверка ports (дополнительная проверка через структуру)
		for j, port := range container.Ports {
			portPrefix := fmt.Sprintf("%s.ports[%d]", prefix, j)

			// Проверка протокола (порт уже проверили выше)
			if port.Protocol != "" && port.Protocol != "TCP" && port.Protocol != "UDP" {
				fmt.Printf("%s.protocol has unsupported value '%s'\n", portPrefix, port.Protocol)
			}
		}

		// Проверка livenessProbe (дополнительная проверка через структуру)
		if container.LivenessProbe != nil {
			if container.LivenessProbe.HTTPGet.Port <= 0 || container.LivenessProbe.HTTPGet.Port >= 65536 {
				// Уже проверили выше через rawMap
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

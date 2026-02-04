package tools

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// getMapStringString 从参数中获取 map[string]string
func getMapStringString(args map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	if val, ok := args[key].(map[string]interface{}); ok {
		for k, v := range val {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
	}
	return result
}

// getStringArg 从参数中获取字符串，支持默认值
func getStringArg(args map[string]interface{}, key, defaultVal string) string {
	if val, ok := args[key].(string); ok && val != "" {
		return val
	}
	return defaultVal
}

// getStringSlice 从参数中获取字符串切片
func getStringSlice(args map[string]interface{}, key string) []string {
	result := []string{}
	if val, ok := args[key].([]interface{}); ok {
		for _, v := range val {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
	}
	return result
}

// getInt32Arg 从参数中获取 int32，支持默认值
func getInt32Arg(args map[string]interface{}, key string, defaultVal int32) int32 {
	if val, ok := args[key].(float64); ok {
		return int32(val)
	}
	if val, ok := args[key].(int); ok {
		return int32(val)
	}
	return defaultVal
}

// buildResourceRequirements 构建资源需求
func buildResourceRequirements(args map[string]interface{}) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	if limitsCPU := getStringArg(args, "limitsCPU", ""); limitsCPU != "" {
		if q, err := resource.ParseQuantity(limitsCPU); err == nil {
			requirements.Limits[corev1.ResourceCPU] = q
		}
	}
	if limitsMemory := getStringArg(args, "limitsMemory", ""); limitsMemory != "" {
		if q, err := resource.ParseQuantity(limitsMemory); err == nil {
			requirements.Limits[corev1.ResourceMemory] = q
		}
	}
	if requestsCPU := getStringArg(args, "requestsCPU", ""); requestsCPU != "" {
		if q, err := resource.ParseQuantity(requestsCPU); err == nil {
			requirements.Requests[corev1.ResourceCPU] = q
		}
	}
	if requestsMemory := getStringArg(args, "requestsMemory", ""); requestsMemory != "" {
		if q, err := resource.ParseQuantity(requestsMemory); err == nil {
			requirements.Requests[corev1.ResourceMemory] = q
		}
	}

	return requirements
}

// buildEnvVars 构建环境变量
func buildEnvVars(args map[string]interface{}) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	if envMap, ok := args["env"].(map[string]interface{}); ok {
		for k, v := range envMap {
			if str, ok := v.(string); ok {
				envVars = append(envVars, corev1.EnvVar{Name: k, Value: str})
			}
		}
	}
	return envVars
}

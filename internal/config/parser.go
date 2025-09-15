/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"sigs.k8s.io/yaml"
)

func ParseConfiguration(ctx context.Context, configPath string) *Configuration {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	envVarReplacedFile := replaceEnvVariables(file)
	var configuration Configuration
	err = yaml.Unmarshal(envVarReplacedFile, &configuration)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	if hasErrors := validateConfiguration(ctx, &configuration); hasErrors {
		return nil
	}
	return &configuration
}

func replaceEnvVariables(content []byte) []byte {
	var allEnvVariables = regexp.MustCompile(`\${(?P<name>\w+)}`)
	return allEnvVariables.ReplaceAllFunc(content, func(matched []byte) []byte {
		varName := allEnvVariables.ReplaceAllString(string(matched), `$1`)
		if value, ok := os.LookupEnv(varName); ok {
			return []byte(value)
		}
		return matched
	})
}

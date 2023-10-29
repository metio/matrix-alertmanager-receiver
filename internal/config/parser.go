/*
 * SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package config

import (
	"context"
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
)

func ParseConfiguration(ctx context.Context, configPath string) *Configuration {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	var configuration Configuration
	err = yaml.Unmarshal(file, &configuration)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}
	if hasErrors := validateConfiguration(ctx, &configuration); hasErrors {
		return nil
	}
	return &configuration
}

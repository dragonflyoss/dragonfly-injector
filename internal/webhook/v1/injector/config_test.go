/*
Copyright 2025.

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

package injector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestLoadInjectConfFromFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("load valid config file successfully", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "valid-config.yaml")
		configData := &InjectConf{
			Enable:          true,
			ProxyPort:       8080,
			CliToolsImage:   "test-image:latest",
			CliToolsDirPath: "/test/tools",
		}
		yamlData, err := yaml.Marshal(configData)
		assert.NoError(t, err)
		err = os.WriteFile(configPath, yamlData, 0644)
		assert.NoError(t, err)

		loadedConfig, err := LoadInjectConfFromFile(configPath)
		assert.NoError(t, err)
		assert.Equal(t, true, loadedConfig.Enable)
		assert.Equal(t, 8080, loadedConfig.ProxyPort)
		assert.Equal(t, "test-image:latest", loadedConfig.CliToolsImage)
		assert.Equal(t, "/test/tools", loadedConfig.CliToolsDirPath)
	})

	t.Run("return error for non-existent file", func(t *testing.T) {
		_, err := LoadInjectConfFromFile("non-existent-file.yaml")
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("return error for invalid YAML content", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "invalid.yaml")
		invalidYAML := "invalid: yaml: content: ["
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		assert.NoError(t, err)

		_, err = LoadInjectConfFromFile(configPath)
		assert.Error(t, err)
	})

	t.Run("handle partial config with zero values", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "partial-config.yaml")
		partialConfig := &InjectConf{Enable: true}
		yamlData, err := yaml.Marshal(partialConfig)
		assert.NoError(t, err)
		err = os.WriteFile(configPath, yamlData, 0644)
		assert.NoError(t, err)

		loadedConfig, err := LoadInjectConfFromFile(configPath)
		assert.NoError(t, err)
		assert.Equal(t, true, loadedConfig.Enable)
		assert.Equal(t, 0, loadedConfig.ProxyPort)
		assert.Equal(t, "", loadedConfig.CliToolsImage)
		assert.Equal(t, "", loadedConfig.CliToolsDirPath)
	})
}

func TestLoadInjectConf(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("load config from file when file exists", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "existing-config.yaml")
		configData := &InjectConf{
			Enable:    false,
			ProxyPort: 1234,
		}
		yamlData, err := yaml.Marshal(configData)
		assert.NoError(t, err)
		err = os.WriteFile(configPath, yamlData, 0644)
		assert.NoError(t, err)

		loadedConfig := LoadInjectConf(configPath)
		assert.Equal(t, false, loadedConfig.Enable)
		assert.Equal(t, 1234, loadedConfig.ProxyPort)
	})

	t.Run("return default config when file does not exist", func(t *testing.T) {
		loadedConfig := LoadInjectConf("non-existent-file.yaml")
		expected := NewDefaultInjectConf()
		assert.Equal(t, expected.Enable, loadedConfig.Enable)
		assert.Equal(t, expected.ProxyPort, loadedConfig.ProxyPort)
		assert.Equal(t, expected.CliToolsImage, loadedConfig.CliToolsImage)
		assert.Equal(t, expected.CliToolsDirPath, loadedConfig.CliToolsDirPath)
	})

	t.Run("return default config when file is invalid", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "invalid.yaml")
		invalidYAML := "invalid: yaml: content: ["
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		assert.NoError(t, err)

		loadedConfig := LoadInjectConf(configPath)
		expected := NewDefaultInjectConf()
		assert.Equal(t, expected.Enable, loadedConfig.Enable)
		assert.Equal(t, expected.ProxyPort, loadedConfig.ProxyPort)
		assert.Equal(t, expected.CliToolsImage, loadedConfig.CliToolsImage)
		assert.Equal(t, expected.CliToolsDirPath, loadedConfig.CliToolsDirPath)
	})
}

func TestNewDefaultInjectConf(t *testing.T) {
	defaultConfig := NewDefaultInjectConf()
	assert.Equal(t, true, defaultConfig.Enable)
	assert.Equal(t, 4001, defaultConfig.ProxyPort)
	assert.Equal(t, "dragonflyoss/cli-tools:latest", defaultConfig.CliToolsImage)
	assert.Equal(t, "/dragonfly-tools", defaultConfig.CliToolsDirPath)
}

func TestConfigManager(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建初始配置
	initialConfig := &InjectConf{
		Enable:          true,
		ProxyPort:       3000,
		CliToolsImage:   "initial:latest",
		CliToolsDirPath: "/initial",
	}
	data, err := yaml.Marshal(initialConfig)
	assert.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	assert.NoError(t, err)

	// 创建ConfigManager
	configManager := NewConfigManager(tempDir)
	assert.NotNil(t, configManager)

	// 测试GetConfig
	config := configManager.GetConfig()
	assert.Equal(t, initialConfig.Enable, config.Enable)
	assert.Equal(t, initialConfig.ProxyPort, config.ProxyPort)
	assert.Equal(t, initialConfig.CliToolsImage, config.CliToolsImage)
	assert.Equal(t, initialConfig.CliToolsDirPath, config.CliToolsDirPath)

	// 测试配置重载
	updatedConfig := &InjectConf{
		Enable:    false,
		ProxyPort: 9999,
	}
	data, err = yaml.Marshal(updatedConfig)
	assert.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	assert.NoError(t, err)

	// 手动触发重载
	configManager.reload()

	// 验证更新后的配置
	config = configManager.GetConfig()
	assert.Equal(t, updatedConfig.Enable, config.Enable)
	assert.Equal(t, updatedConfig.ProxyPort, config.ProxyPort)
}

func TestConfigManagerStart(t *testing.T) {
	tempDir := t.TempDir()

	// 创建ConfigManager
	configManager := NewConfigManager(tempDir)

	// 创建可取消的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动ConfigManager
	done := make(chan error)
	go func() {
		done <- configManager.Start(ctx)
	}()

	// 等待一小段时间确保启动
	time.Sleep(100 * time.Millisecond)

	// 取消context
	cancel()

	// 等待退出
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ConfigManager did not stop within timeout")
	}
}

func TestConfigManagerFileNotExist(t *testing.T) {
	tempDir := t.TempDir()

	// 创建ConfigManager，但配置文件不存在
	configManager := NewConfigManager(tempDir)

	// 应该使用默认配置
	config := configManager.GetConfig()
	expected := NewDefaultInjectConf()
	assert.Equal(t, expected.Enable, config.Enable)
	assert.Equal(t, expected.ProxyPort, config.ProxyPort)
	assert.Equal(t, expected.CliToolsImage, config.CliToolsImage)
	assert.Equal(t, expected.CliToolsDirPath, config.CliToolsDirPath)
}

func TestConfigManagerConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建初始配置
	configData := &InjectConf{
		Enable:          true,
		ProxyPort:       3000,
		CliToolsImage:   "initial:latest",
		CliToolsDirPath: "/initial",
	}
	yamlData, err := yaml.Marshal(configData)
	assert.NoError(t, err)
	err = os.WriteFile(configPath, yamlData, 0644)
	assert.NoError(t, err)

	configManager := NewConfigManager(tempDir)

	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			config := configManager.GetConfig()
			assert.NotNil(t, config)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			configManager.reload()
		}
		done <- true
	}()

	// 等待两个goroutine完成
	<-done
	<-done
}

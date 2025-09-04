package injector

import (
	"context"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Config", func() {
	var (
		tempDir string
	)

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
	})

	Describe("LoadInjectConfFromFile", func() {
		Context("when loading configuration from file", func() {
			It("should load valid config file successfully", func() {
				By("creating a valid config file")
				configPath := filepath.Join(tempDir, "valid-config.yaml")
				configData := &InjectConf{
					Enable:          true,
					ProxyPort:       8080,
					CliToolsImage:   "test-image:latest",
					CliToolsDirPath: "/test/tools",
				}
				yamlData, err := yaml.Marshal(configData)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(configPath, yamlData, 0644)
				Expect(err).NotTo(HaveOccurred())

				By("loading the config from file")
				loadedConfig, err := LoadInjectConfFromFile(configPath)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the loaded configuration")
				Expect(loadedConfig.Enable).To(BeTrue())
				Expect(loadedConfig.ProxyPort).To(Equal(8080))
				Expect(loadedConfig.CliToolsImage).To(Equal("test-image:latest"))
				Expect(loadedConfig.CliToolsDirPath).To(Equal("/test/tools"))
			})

			It("should return error for non-existent file", func() {
				By("attempting to load a non-existent file")
				_, err := LoadInjectConfFromFile("non-existent-file.yaml")
				Expect(err).To(HaveOccurred())
				Expect(os.IsNotExist(err)).To(BeTrue())
			})

			It("should return error for invalid YAML content", func() {
				By("creating a file with invalid YAML content")
				configPath := filepath.Join(tempDir, "invalid.yaml")
				invalidYAML := "invalid: yaml: content: ["
				err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
				Expect(err).NotTo(HaveOccurred())

				By("attempting to load the invalid YAML file")
				_, err = LoadInjectConfFromFile(configPath)
				Expect(err).To(HaveOccurred())
			})

			It("should handle partial config with zero values", func() {
				By("creating a partial config file")
				configPath := filepath.Join(tempDir, "partial-config.yaml")
				partialConfig := &InjectConf{Enable: true}
				yamlData, err := yaml.Marshal(partialConfig)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(configPath, yamlData, 0644)
				Expect(err).NotTo(HaveOccurred())

				By("loading the partial config")
				loadedConfig, err := LoadInjectConfFromFile(configPath)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the loaded configuration with zero values")
				Expect(loadedConfig.Enable).To(BeTrue())
				Expect(loadedConfig.ProxyPort).To(Equal(0))
				Expect(loadedConfig.CliToolsImage).To(BeEmpty())
				Expect(loadedConfig.CliToolsDirPath).To(BeEmpty())
			})
		})
	})

	Describe("LoadInjectConf", func() {
		Context("when loading configuration with fallback behavior", func() {
			It("should load config from file when file exists", func() {
				By("creating an existing config file")
				configPath := filepath.Join(tempDir, "existing-config.yaml")
				configData := &InjectConf{
					Enable:    false,
					ProxyPort: 1234,
				}
				yamlData, err := yaml.Marshal(configData)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(configPath, yamlData, 0644)
				Expect(err).NotTo(HaveOccurred())

				By("loading the config")
				loadedConfig := LoadInjectConf(configPath)
				Expect(loadedConfig.Enable).To(BeFalse())
				Expect(loadedConfig.ProxyPort).To(Equal(1234))
			})

			It("should return default config when file does not exist", func() {
				By("loading a non-existent file")
				loadedConfig := LoadInjectConf("non-existent-file.yaml")
				expected := NewDefaultInjectConf()

				By("verifying the default configuration is returned")
				Expect(loadedConfig.Enable).To(Equal(expected.Enable))
				Expect(loadedConfig.ProxyPort).To(Equal(expected.ProxyPort))
				Expect(loadedConfig.CliToolsImage).To(Equal(expected.CliToolsImage))
				Expect(loadedConfig.CliToolsDirPath).To(Equal(expected.CliToolsDirPath))
			})

			It("should return default config when file is invalid", func() {
				By("creating an invalid config file")
				configPath := filepath.Join(tempDir, "invalid.yaml")
				invalidYAML := "invalid: yaml: content: ["
				err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
				Expect(err).NotTo(HaveOccurred())

				By("loading the invalid file")
				loadedConfig := LoadInjectConf(configPath)
				expected := NewDefaultInjectConf()

				By("verifying the default configuration is returned")
				Expect(loadedConfig.Enable).To(Equal(expected.Enable))
				Expect(loadedConfig.ProxyPort).To(Equal(expected.ProxyPort))
				Expect(loadedConfig.CliToolsImage).To(Equal(expected.CliToolsImage))
				Expect(loadedConfig.CliToolsDirPath).To(Equal(expected.CliToolsDirPath))
			})
		})
	})

	Describe("NewDefaultInjectConf", func() {
		It("should return the correct default configuration", func() {
			By("creating a new default config")
			defaultConfig := NewDefaultInjectConf()

			By("verifying the default values")
			Expect(defaultConfig.Enable).To(BeTrue())
			Expect(defaultConfig.ProxyPort).To(Equal(4001))
			Expect(defaultConfig.CliToolsImage).To(Equal("dragonflyoss/cli-tools:latest"))
			Expect(defaultConfig.CliToolsDirPath).To(Equal("/dragonfly-tools"))
		})
	})

	Describe("ConfigManager", func() {
		var (
			configManager *ConfigManager
		)

		Context("with basic functionality", func() {
			BeforeEach(func() {
				By("creating initial configuration")
				configPath := filepath.Join(tempDir, "config.yaml")
				initialConfig := &InjectConf{
					Enable:          true,
					ProxyPort:       3000,
					CliToolsImage:   "initial:latest",
					CliToolsDirPath: "/initial",
				}
				data, err := yaml.Marshal(initialConfig)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(configPath, data, 0644)
				Expect(err).NotTo(HaveOccurred())

				By("creating the ConfigManager")
				configManager = NewConfigManager(tempDir)
				Expect(configManager).NotTo(BeNil())
			})

			It("should get the correct configuration", func() {
				By("retrieving the configuration")
				config := configManager.GetConfig()

				By("verifying the configuration values")
				Expect(config.Enable).To(BeTrue())
				Expect(config.ProxyPort).To(Equal(3000))
				Expect(config.CliToolsImage).To(Equal("initial:latest"))
				Expect(config.CliToolsDirPath).To(Equal("/initial"))
			})

			It("should reload configuration correctly", func() {
				By("updating the configuration file")
				updatedConfig := &InjectConf{
					Enable:    false,
					ProxyPort: 9999,
				}
				data, err := yaml.Marshal(updatedConfig)
				Expect(err).NotTo(HaveOccurred())
				configPath := filepath.Join(tempDir, "config.yaml")
				err = os.WriteFile(configPath, data, 0644)
				Expect(err).NotTo(HaveOccurred())

				By("triggering configuration reload")
				configManager.reload()

				By("verifying the updated configuration")
				config := configManager.GetConfig()
				Expect(config.Enable).To(BeFalse())
				Expect(config.ProxyPort).To(Equal(9999))
			})
		})

		Context("when configuration file does not exist", func() {
			It("should use default configuration", func() {
				By("creating ConfigManager without config file")
				configManager := NewConfigManager(tempDir)

				By("verifying default configuration is used")
				config := configManager.GetConfig()
				expected := NewDefaultInjectConf()
				Expect(config.Enable).To(Equal(expected.Enable))
				Expect(config.ProxyPort).To(Equal(expected.ProxyPort))
				Expect(config.CliToolsImage).To(Equal(expected.CliToolsImage))
				Expect(config.CliToolsDirPath).To(Equal(expected.CliToolsDirPath))
			})
		})

		Context("Start and Stop functionality", func() {
			It("should start and stop gracefully", func() {
				By("creating ConfigManager")
				configManager := NewConfigManager(tempDir)

				By("creating a cancellable context")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("starting the ConfigManager")
				done := make(chan error)
				go func() {
					done <- configManager.Start(ctx)
				}()

				By("waiting for startup")
				time.Sleep(100 * time.Millisecond)

				By("cancelling the context")
				cancel()

				By("waiting for graceful shutdown")
				Eventually(done, 5*time.Second).Should(Receive(BeNil()))
			})
		})

		Context("concurrent access", func() {
			BeforeEach(func() {
				By("creating initial configuration for concurrent testing")
				configPath := filepath.Join(tempDir, "config.yaml")
				configData := &InjectConf{
					Enable:          true,
					ProxyPort:       3000,
					CliToolsImage:   "initial:latest",
					CliToolsDirPath: "/initial",
				}
				yamlData, err := yaml.Marshal(configData)
				Expect(err).NotTo(HaveOccurred())
				err = os.WriteFile(configPath, yamlData, 0644)
				Expect(err).NotTo(HaveOccurred())

				configManager = NewConfigManager(tempDir)
			})

			It("should handle concurrent access safely", func() {
				By("starting concurrent readers")
				done := make(chan bool)
				go func() {
					defer GinkgoRecover()
					for i := 0; i < 100; i++ {
						config := configManager.GetConfig()
						Expect(config).NotTo(BeNil())
					}
					done <- true
				}()

				By("starting concurrent reloaders")
				go func() {
					defer GinkgoRecover()
					for i := 0; i < 100; i++ {
						configManager.reload()
					}
					done <- true
				}()

				By("waiting for all goroutines to complete")
				Eventually(done).Should(Receive())
				Eventually(done).Should(Receive())
			})
		})
	})
})

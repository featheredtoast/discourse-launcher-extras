package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"context"
	"github.com/discourse/discourse_docker/launcher_go/v2/config"
	"github.com/discourse/discourse_docker/launcher_go/v2/utils"
	ddocker "github.com/featheredtoast/launcher-extras"
	"os"
)

var _ = Describe("Generate", func() {
	var testDir string
	var out *bytes.Buffer
	var cli *ddocker.Cli
	var ctx context.Context

	BeforeEach(func() {
		utils.DockerPath = "docker"
		out = &bytes.Buffer{}
		utils.Out = out
		testDir, _ = os.MkdirTemp("", "ddocker-test")

		ctx = context.Background()

		cli = &ddocker.Cli{
			ConfDir:      "./test/containers",
			TemplatesDir: "./test",
		}
	})
	AfterEach(func() {
		os.RemoveAll(testDir)
	})

	It("should output docker compose cmd to config name's subdir", func() {
		runner := ddocker.DockerComposeCmd{Config: []string{"test"},
			OutputDir: testDir}
		err := runner.Run(cli, &ctx)
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/test/config.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("should force create output parent folders", func() {
		runner := ddocker.DockerComposeCmd{Config: []string{"test"},
			OutputDir: testDir + "/subfolder/sub-subfolder"}
		err := runner.Run(cli, &ctx)
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/subfolder/sub-subfolder/test/config.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("can write a docker compose setup", func() {
		conf, _ := config.LoadConfig("./test/containers", "test", true, "./test")
		ddocker.WriteDockerCompose(*conf, testDir, false)
		out, err := os.ReadFile(testDir + "/.envrc")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("export DISCOURSE_HOSTNAME"))
		out, err = os.ReadFile(testDir + "/config.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
		out, err = os.ReadFile(testDir + "/test.dockerfile")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("RUN cat /temp-config.yaml"))

		out, err = os.ReadFile(testDir + "/docker-compose.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("build:"))
		Expect(string(out[:])).To(ContainSubstring("dockerfile: ./test.dockerfile"))
		Expect(string(out[:])).To(ContainSubstring("image: local_discourse/test"))
	})
})

package main_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bytes"
	"context"
	"os"

	"github.com/discourse/launcher/v2/config"
	"github.com/discourse/launcher/v2/utils"
	ddocker "github.com/featheredtoast/discourse-launcher-extras"
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
		os.RemoveAll(testDir) //nolint:errcheck
	})

	It("should output docker compose cmd to config name's subdir", func() {
		runner := ddocker.DockerComposeCmd{Config: []string{"test"},
			OutputDir: testDir}
		err := runner.Run(cli, &ctx)
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/test/test.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("should force create output parent folders", func() {
		runner := ddocker.DockerComposeCmd{Config: []string{"test"},
			OutputDir: testDir + "/subfolder/sub-subfolder"}
		err := runner.Run(cli, &ctx)
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/subfolder/sub-subfolder/test/test.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring("DISCOURSE_DEVELOPER_EMAILS: 'me@example.com,you@example.com'"))
	})

	It("can write a docker compose setup", func() {
		conf, _ := config.LoadConfig("./test/containers", "test", true, "./test")
		err := ddocker.WriteDockerCompose([]config.Config{*conf}, testDir)
		Expect(err).To(BeNil())
		out, err := os.ReadFile(testDir + "/.envrc")
		Expect(err).To(BeNil())
		// envrc does not export secrets since we don't build with them
		Expect(string(out[:])).ToNot(ContainSubstring("export DISCOURSE_DB_PASSWORD"))
		Expect(string(out[:])).To(ContainSubstring("export RAILS_ENV"))
		out, err = os.ReadFile(testDir + "/test.yaml")
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

	It("can write multiple containers to a single compose", func() {
		runner := ddocker.DockerComposeCmd{Config: []string{"web_only", "data"},
			OutputDir: testDir}
		err := runner.Run(cli, &ctx)
		Expect(err).To(BeNil())

		//expect envrc to be concatenated, with web_only last, as last-write-wins
		out, err := os.ReadFile(testDir + "/web_only/.envrc")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring(`export LANG="en_US.UTF-8"
export LANGUAGE="en_US.UTF-8"
export LC_ALL="en_US.UTF-8"
export LANG="en_US.UTF-8"
export LANGUAGE="en_US.UTF-8"
export LC_ALL="en_US.UTF-8"
export RAILS_ENV="production"
export RUBY_GC_HEAP_GROWTH_MAX_SLOTS="40000"
export RUBY_GC_HEAP_INIT_SLOTS="400000"
export RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR="1.5"
export UNICORN_SIDEKIQS="1"
export UNICORN_WORKERS="3"`))

		// expect web template to be printed here
		out, err = os.ReadFile(testDir + "/web_only/web_only.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring(`env:
  # You can have redis on a different box
  RAILS_ENV: 'production'
  UNICORN_WORKERS: 3
  UNICORN_SIDEKIQS: 1
  # stop heap doubling in size so aggressively, this conserves memory
  RUBY_GC_HEAP_GROWTH_MAX_SLOTS: 40000
  RUBY_GC_HEAP_INIT_SLOTS: 400000
  RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR: 1.5

  DISCOURSE_DB_SOCKET: /var/run/postgresql
  DISCOURSE_DB_HOST:
  DISCOURSE_DB_PORT:`))

		out, err = os.ReadFile(testDir + "/web_only/web_only.dockerfile")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring(`FROM ${dockerfile_from_image}
ARG LANG
ARG LANGUAGE
ARG LC_ALL
ARG RAILS_ENV
ARG RUBY_GC_HEAP_GROWTH_MAX_SLOTS
ARG RUBY_GC_HEAP_INIT_SLOTS
ARG RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR
ARG UNICORN_SIDEKIQS
ARG UNICORN_WORKERS
ENV RAILS_ENV=${RAILS_ENV}
ENV RUBY_GC_HEAP_GROWTH_MAX_SLOTS=${RUBY_GC_HEAP_GROWTH_MAX_SLOTS}
ENV RUBY_GC_HEAP_INIT_SLOTS=${RUBY_GC_HEAP_INIT_SLOTS}
ENV RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR=${RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR}
ENV UNICORN_SIDEKIQS=${UNICORN_SIDEKIQS}
ENV UNICORN_WORKERS=${UNICORN_WORKERS}
EXPOSE 443
EXPOSE 80
COPY web_only.yaml /temp-config.yaml
RUN cat /temp-config.yaml | /usr/local/bin/pups --skip-tags=precompile,migrate,db --stdin && rm /temp-config.yaml
CMD ["/sbin/boot"]`))

		out, err = os.ReadFile(testDir + "/web_only/docker-compose.yaml")
		Expect(err).To(BeNil())
		Expect(string(out[:])).To(ContainSubstring(`services:
  data:
    image: local_discourse/data
    build:
      dockerfile: ./data.dockerfile
      labels: {}
      shm_size: 512m
      args:
        - LANG
        - LANGUAGE
        - LC_ALL
      no_cache: true
    volumes:
      - /var/discourse/shared/data/log/var-log:/var/log
      - /var/discourse/shared/data:/shared
    links: []
    environment:
      CREATE_DB_ON_BOOT: "1"
      LANG: en_US.UTF-8
      LANGUAGE: en_US.UTF-8
      LC_ALL: en_US.UTF-8
      MIGRATE_ON_BOOT: "1"
      PRECOMPILE_ON_BOOT: "1"
    ports: []
  web_only:
    image: local_discourse/web_only
    build:
      dockerfile: ./web_only.dockerfile
      labels: {}
      shm_size: 512m
      args:
        - LANG
        - LANGUAGE
        - LC_ALL
        - RAILS_ENV
        - RUBY_GC_HEAP_GROWTH_MAX_SLOTS
        - RUBY_GC_HEAP_INIT_SLOTS
        - RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR
        - UNICORN_SIDEKIQS
        - UNICORN_WORKERS
      no_cache: true
    volumes:
      - /var/discourse/shared/web-only/log/var-log:/var/log
      - /var/discourse/shared/web-only:/shared
    links:
      - data:data
    environment:
      CREATE_DB_ON_BOOT: "1"
      DISCOURSE_DB_HOST: data
      DISCOURSE_DB_PASSWORD: SOME_SECRET
      DISCOURSE_DB_PORT: ""
      DISCOURSE_DB_SOCKET: ""
      DISCOURSE_DEVELOPER_EMAILS: me@example.com,you@example.com
      DISCOURSE_HOSTNAME: discourse.example.com
      DISCOURSE_REDIS_HOST: data
      DISCOURSE_SMTP_ADDRESS: smtp.example.com
      DISCOURSE_SMTP_PASSWORD: pa$$word
      DISCOURSE_SMTP_USER_NAME: user@example.com
      LANG: en_US.UTF-8
      LANGUAGE: en_US.UTF-8
      LC_ALL: en_US.UTF-8
      MIGRATE_ON_BOOT: "1"
      PRECOMPILE_ON_BOOT: "1"
      RAILS_ENV: production
      RUBY_GC_HEAP_GROWTH_MAX_SLOTS: "40000"
      RUBY_GC_HEAP_INIT_SLOTS: "400000"
      RUBY_GC_HEAP_OLDOBJECT_LIMIT_FACTOR: "1.5"
      UNICORN_SIDEKIQS: "1"
      UNICORN_WORKERS: "3"
    ports:
      - 443:443
      - 80:80
volumes: {}`))
	})
})

# Launcher Extras

Extra scripts for other possibly useful commands from launcher2 rewrite

## Docker compose generation.

Allows easier exporting of configuration from discourse's pups configuration to a docker compose configuration.

Still probably needs some way of exporting multi containers...

TODO: Maybe compose can take in multiple containers, and combine them all in one.
Downside, this means all env is shared between containers, probably. and value of all env needs to be the same.

## Concourse generation

Generates a concourse directory for build jobs.

it generates a yaml file with 3 key/value pairs:

* the dockerfile used to build
* the source pups yaml config (as a string)
* concourse job. This can be used in a concourse job in the following example:

Let's assume we ran the following: `launcher-extras concourse-job --conf-dir {path-to-container-config} --templates-dir discourse_docker --output job-config/config.yaml {site-name}` and have job-config/config.yaml be the output of the `concourse-job` command

We can then use a job skeleton to run the included task:

contents of `job-skeleton/skeleton.yaml`:
```
resource_types:
- name: static
  type: docker-image
  source: { repository: ktchen14/static-resource }
resources:
  - name: docker-from-image
    type: registry-image
    source:
      repository: ((from_namespace))
      tag: ((from_tag))
  - name: dockerhub-image
    type: registry-image
    source:
      repository: (docker repository)
      tag: latest
      username: (username)
      password: (password)
  - name: docker-config
    type: static
    source:
      concourse_task.yaml: ((concourse_task))
      Dockerfile: ((dockerfile))
      config.yaml: ((config))
jobs:
  - name: build
    serial: true
    plan:
      - in_parallel:
        - get: docker-config
        - get: docker-from-image
          params:
            format: oci
      - task: build-base
        privileged: true
        params:
          CONTEXT: docker-config
          IMAGE_ARG_dockerfile_from_image: docker-from-image/image.tar
        file: docker-config/concourse_task.yaml
        input_mapping:
          docker-config: docker-config
          docker-from-image: docker-from-image
        output_mapping:
          image: image
      - put: dockerhub-image
        params: {image: image/image.tar}
        no_get: true
```
We may have a set pipeline task that may be set with var_files to generate the job:

```
- set_pipeline: build-job-test
  file: job-skeleton/skeleton.yaml
  var_files: ["job-config/config.yaml"]
```

### Why so complicated?

TLDR: newline support.

The passing of build args is only supported via:
* hardcoded env vars in the job in the job yaml, and using set-env from other variables.
* an env file, with `key=value` on each line.

Hardcoding env isn't an option because we don't know all variables.
Env file is close, but doesn't support newlines.

I've opened a PR against the oci-images that will allow for passing key/value in newline format, using a yaml format.

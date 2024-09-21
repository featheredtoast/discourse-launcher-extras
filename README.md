# Launcher Extras

Extra scripts for other possibly useful commands from launcher2 rewrite

## Docker compose generation.

Allows easier exporting of configuration from discourse's pups configuration to a docker compose configuration.

Allows multiple build containers: `launcher-extras compose app data`

### Limitations

compose's build uses a common env - all shared env keys need to be the same (EG LANG).

The written .envrc will only work with the same key/values, but won't work if each container are assumng different values.

## Concourse generation

Generates a concourse directory to build an image for a job.

it generates a yaml file with the following key/value pairs:

* from_namespace the namespace to build from, eg: discourse/base
* from_tag: the tag to build from eg: 2.0.20240825-0027
* the dockerfile used to build
* the source pups yaml config (as a string)
* concourse job. This can be used in a concourse job in the following example:

Let's assume we ran the following: `launcher-extras concourse-job --conf-dir {path-to-container-config} --templates-dir discourse_docker --output job-config/config.yaml {site-name}` and have job-config/config.yaml be the output of the `concourse-job` command

We can then use a job skeleton to run the included task:
(note: additional vars are needed: docker_repository eg `discourse/fully_powered`, docker username and password)

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
      repository: ((docker_repository))
      tag: latest
      username: ((username))
      password: ((password))
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

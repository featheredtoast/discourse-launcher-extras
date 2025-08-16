# Launcher Extras

Extra scripts for other possibly useful commands from launcher2 rewrite

## Docker compose generation.

Allows easier exporting of configuration from discourse's pups configuration to a docker compose configuration.

Allows multiple build containers: `launcher-extras compose app data`

### Limitations

compose's build uses a common env - all shared env keys need to be the same (EG LANG).

The written .envrc will only work with the same key/values, but won't work if each container are assumng different values.

## Exports/prints

Export or prints the Dockerfile, env (in yaml format), or pups config.

Intended for use with concourse concourse/oci-build-task. An example job snippet is below:

Note, the job config that the dockerfile expects is currently hardcoded as `config.yaml`

```
      - task: gen-dockerfiles
        config:
          platform: linux
          image_resource:
            type: registry-image
            source:
              repository: busybox
          inputs:
            - name: launcher-extras
            - name: discourse_docker
          outputs:
            - name: job-config
          run:
            path: sh
            args:
              - -exc
              - |
                launcher-extras/launcher-extras --conf-dir discourse_docker/containers --templates-dir discourse_docker print dockerfile web_only > job-config/Dockerfile &&
                launcher-extras/launcher-extras --conf-dir discourse_docker/containers --templates-dir discourse_docker print config web_only > job-config/config.yaml &&
                launcher-extras/launcher-extras --conf-dir discourse_docker/containers --templates-dir discourse_docker print env web_only > job-config/env.yaml
      - task: build-image
        privileged: true
        config:
          platform: linux
          image_resource:
            type: registry-image
            source:
              repository: concourse/oci-build-task
          inputs:
            - name: job-config
          outputs:
            - name: image
          params:
            CONTEXT: job-config
            BUILD_ARGS_FILE: job-config/env.yaml
          run:
            path: build
```

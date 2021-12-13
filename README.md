Simpler version of https://github.com/dmacvicar/terraform-provider-libvirt.

* Takes libvirt XML as input.
* Define and undefine domains only - guests will not be started or stopped.

`GO111MODULE=on GOOS=linux go install`

### Env

```
podman run -it --rm \
  -v $(pwd):/go/src/github.com/randomcoww/terraform-provider-libvirt \
  -w /go/src/github.com/randomcoww/terraform-provider-libvirt \
   golang:alpine sh
```
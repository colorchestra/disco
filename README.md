# 🪩 disco - disposable containers with podman
easily create podman containers that only exist until you exit.

## features / usage
- automatically mounts a temporary directory in /tmp
- name your container and temp dir with `--name`
- select your favourite distro with `--image`
- no network by default; enable with `--network`
- clean up your temp dir after exiting with `--cleanup`

yay!

## to do
- [ ] arg shorthands
- [ ] parse image URLs, download from specific registry
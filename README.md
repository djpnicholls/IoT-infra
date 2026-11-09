# IoT-infra
A place for me to experiment with IoT - creates the OS-level configuration that can be flashed to a raspberry Pi or similar.
Also, to discover how much of my (cloud) SRE experience carries over.

Current project: a probe to measure temperature and volume levels, so I can correlate whether temperature has an effect on waking.

Note: Since I currently lack the hardware to measure temperature and volume, these readings are simulated for now.

## Architecture

The probe project consists of two halves: a probe and an analyzer.

The probe is deployed as a systemd service running on a Raspberry Pi B, taking (or simulating) measurements and exposing these in Prometheus format on a TCP socket.

The analyzer is deployed as a set of containers (podman or k8s), running on a separate Raspberry Pi 5, collecting these measurements and running analysis.

> Design note: originally, I intended to run these two halves on the same machine.  The probe would be deployed as systemd services at the OS level, since I don't anticipate it changing once it's done, and the analyzer would run on k3s, since it's the item most likely to change and be iterated on.  The probe would have exposed the readings on a unix socket.  But the Raspberry Pi B doesn't have the resources to do this, so the analyzer now runs on a separate host.

## Probe

This is a Go [programme](./drivers/measurement/).

This is written in Go because it can be deployed as a single binary, without worrying about interpreters or dependencies.

## Systemd files

There are two [systemd unit files](./systemd/unitFiles/).
These create the socket and service for the probe.

We use socket activation, to avoid wasting work and taking measurements while there's nothing listening.

## Ansible

There is an [ansible playbook](./ansible/) for deploying the probe to the host it runs on (the Raspberry Pi B).

## Testing

The [test script](build-test.py) is used to test the deployment of the probe.
It does the following:
1. Build the probe application
2. Create a new [container](Containerfile.rpi), simulating a Raspberry Pi B (running 32-bit, arm v6, running a slimmed down version of Raspbian, including Python and systemd).
3. Run the [ansible playbook](./ansible/) to deploy the probe application properly to the container.
4. Test that it can reach the correct sockets and receive the correct measurements.
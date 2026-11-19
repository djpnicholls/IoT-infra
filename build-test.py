#! /bin/env python3

import subprocess
import os
import requests

def podman(*args, check=True):
	subprocess.run(["podman", *args],
	 	check=check)

print("Building measurement application...")
subprocess.run(["go", "build", "-o", "../../dist/measurement_arm_v6", "."], 
	check=True,
	cwd="drivers/measurement",
	env={**os.environ, "GOARCH": "arm", "GOARM": "6", "GOOS": "linux"})
print("Measurement application built successfully.")

print("Building container image...")
podman("build", "--platform", "linux/arm/v6", "-t", 
	"local/pi-b-rootfs", "-f", "Containerfile.rpi", ".")
print("Container image built successfully.")	

print("Starting container...")
podman("run", "-d", "--name", "pi-build-env",
	"--replace", "--platform", "linux/arm/v6",
	"--systemd=always", "--security-opt", "label=disable",
	"-p", "8091:8091", "-p", "8092:8092",
	"localhost/local/pi-b-rootfs")
print("Container running.")

print("Running Ansible playbook...")
subprocess.run(["ansible-playbook", "-i", "inventory.ini", "playbook.yaml"],
	check=True,
	cwd="ansible")
print("Ansible playbook run successfully.")

print("Starting test...")
resp = requests.get(
	"http://127.0.0.1:8092/",
	timeout=10,
)
resp.raise_for_status()

assert "room_temperature_celsius" in resp.text
assert "room_volume_decibels" in resp.text

print("Test finished.  Check result.")

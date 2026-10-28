#! /bin/env python3

import subprocess
import os
import socket

print("Building measurement application...")
subprocess.run(["go", "build", "-o", "../../dist/measurement_arm_v6", "."], 
	check=True,
	cwd="drivers/measurement",
	env={"GOARCH": "arm", "GOARM": "6", "HOME": os.environ.get("HOME")})
print("Measurement application built successfully.")

print("\n\n")
print("Building container image...")
subprocess.run(["podman", "build", "--platform", "linux/arm/v6", "-t", 
	"local/pi-b-rootfs", "-f", "Containerfile.rpi", "."],
	 check=True)
print("Container image built successfully.")	

print("\n\n")
print("Starting container...")
subprocess.run(["podman", "run", "-d", "--name", "pi-build-env",
	"--replace", "--platform", "linux/arm/v6",
	"--systemd=always", "--security-opt", "label=disable",
	"-p", "8091:8091", "-p", "8092:8092",
	"localhost/local/pi-b-rootfs"],
	check=True)
print("Container running.")

print("\n\n")
print("Running Ansible playbook...")
subprocess.run(["ansible-playbook", "-i", "inventory.ini", "playbook.yaml"],
	check=True,
	cwd="ansible")
print("Ansible playbook run successfully.")

print("\n\n")
print("Starting test...")
sock1 = socket.create_connection(("127.0.0.1", 8091))
with sock1.makefile() as s:
	print(s.readline())
sock1.close()

sock2 = socket.create_connection(("127.0.0.1", 8092))
with sock2.makefile() as s:
	print(s.readline())
sock2.close()

print("Test finished.  Check result.")

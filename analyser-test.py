#! /bin/env python3

import subprocess
import os
import socket

print("Building measurement application...")
subprocess.run(["go", "build", "-o", "../../dist/measurement_x86", "."], 
	check=True,
	cwd="drivers/measurement",
	env={"GOOS": "linux", "HOME": os.environ.get("HOME")})
print("Measurement application built successfully.")

print("\n\n")
print("Building container image...")
subprocess.run(["podman", "build",  "-t", 
	"local/probe", "-f", "Containerfile.probe", "."],
	 check=True)
print("Container image built successfully.")	

print("\n\n")
print("Starting probe container...")
subprocess.run(["podman", "run", "-d", "--rm", "--name", "probe",
	"--replace", "-e", "MEASUREMENT_SOURCE=temp",
	"-p", "8091:8090",
	"localhost/local/probe"],
	check=True)
print("Probe container running.  Listening on port 8091")

print("\n\n")
print("Starting test...")

print("Test finished.  Check result.")

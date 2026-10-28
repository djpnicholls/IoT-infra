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
print("Starting temp probe container...")
subprocess.run(["podman", "run", "-d", "--rm", "--name", "probe-temp",
	"--replace", "-e", "MEASUREMENT_SOURCE=temp",
	"-p", "8091:8090",
	"localhost/local/probe"],
	check=True)
print("Temp probe container running.  Listening on port 8091")

print("\n\n")
print("Starting volume probe container...")
subprocess.run(["podman", "run", "-d", "--rm", "--name", "probe-volume",
	"--replace", "-e", "MEASUREMENT_SOURCE=volume",
	"-p", "8092:8090",
	"localhost/local/probe"],
	check=True)
print("Volume probe container running.  Listening on port 8092")

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

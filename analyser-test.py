#! /bin/env python3

import subprocess
import os
import requests

print("Building measurement application...")
subprocess.run(["go", "build", "-o", "../../dist/measurement_x86", "."], 
	check=True,
	cwd="drivers/measurement",
	env={**os.environ, "GOOS": "linux"})
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

resp = requests.get(
	"http://127.0.0.1:8091/",
	timeout=10,
)
resp.raise_for_status()

assert "room_temperature_celsius" in resp.text
assert "room_volume_decibels" in resp.text

print("Test passed.")

print("Pulling victoria metrics container")

subprocess.run(["podman", "pull", "victoriametrics/victoria-metrics:v1.156.0"],check=True)

print("Running victoria metrics container")
subprocess.run(["podman", "run", "-d", "--rm", "--name", "vmetrics",
	"--replace", "-v", "test-scrape.yaml:/test-scrape.yaml",
	"-p", "8428:8428",
	 "victoriametrics/victoria-metrics:v1.156.0",
     "-promscrape.config=/test-scrape.yaml",
     "-selfScrapeInterval=10s"],
	check=True)
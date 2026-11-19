#! /bin/env python3

import subprocess
import os
import requests

PODMAN_NET_NAME="test-net"

def podman(*args, check=True):
	subprocess.run(["podman", *args],
	 	check=check)

print("Building measurement application...")
subprocess.run(["go", "build", "-o", "../../dist/measurement_x86", "."], 
	check=True,
	cwd="drivers/measurement",
	env={**os.environ, "GOOS": "linux"})
print("Measurement application built successfully.")

print("Creating network")
podman("network", "create", PODMAN_NET_NAME, "--ignore")

print("Building container image...")
podman ("build",  "-t", 
	"local/probe", "-f", "Containerfile.probe", ".")
print("Container image built successfully.")	

print("Starting probe container...")
podman( "run", "-d", "--rm", "--name", "probe",
	"--replace", 
	"-p", "8090:8090",
	"--network", PODMAN_NET_NAME,
	"localhost/local/probe")
print("Probe container running.  Listening on port 8091")

print("Starting initial query test...")

resp = requests.get(
	"http://127.0.0.1:8091/",
	timeout=10,
)
resp.raise_for_status()

assert "room_temperature_celsius" in resp.text
assert "room_volume_decibels" in resp.text

print("Test passed.")

print("Pulling victoria metrics container")

podman( "pull", "victoriametrics/victoria-metrics:v1.156.0")

print("Running victoria metrics container")
podman("run", "-d", "--rm", "--name", "vmetrics",
	"--replace", "-v", "test-scrape.yaml:/test-scrape.yaml",
	"-p", "8428:8428",
	"--network", PODMAN_NET_NAME,
	 "victoriametrics/victoria-metrics:v1.156.0",
     "-promscrape.config=/test-scrape.yaml",
     "-selfScrapeInterval=10s")
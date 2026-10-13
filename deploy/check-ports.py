#!/usr/bin/env python3
import sys


def listening_ports():
    ports = set()
    for path in ("/proc/net/tcp", "/proc/net/tcp6"):
        try:
            with open(path, encoding="ascii") as connections:
                next(connections)
                for line in connections:
                    fields = line.split()
                    if fields[3] == "0A":
                        ports.add(int(fields[1].rsplit(":", 1)[1], 16))
        except FileNotFoundError:
            continue
    return ports


try:
    ports = [int(value) for value in sys.argv[1:]]
    if len(ports) != 2 or ports[0] == ports[1]:
        raise ValueError("HTTP_PORT and HTTPS_PORT must be distinct valid ports")
    for port in ports:
        if not 1 <= port <= 65535:
            raise ValueError(f"invalid host port {port}")
    occupied = listening_ports().intersection(ports)
    if occupied:
        raise ValueError(f"host port(s) already in use: {', '.join(map(str, sorted(occupied)))}")
except ValueError as error:
    print(f"Port check failed: {error}", file=sys.stderr)
    sys.exit(1)

print("HTTP and HTTPS host ports are available.")
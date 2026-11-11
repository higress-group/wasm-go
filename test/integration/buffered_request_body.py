#!/usr/bin/env python3
# Copyright (c) 2026 Alibaba Group Holding Ltd.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Verify buffered bodies and protocol handshakes in an isolated Higress Envoy.

Requires Go, Docker, redis-server and redis-cli. Builds real community plugins
using temporary module files; leaves both source checkouts untouched. Listeners
bind to loopback, and only processes/containers created here are stopped.
"""
import argparse
import base64
import hashlib
import http.client
import http.server
import json
import os
import pathlib
import shutil
import socket
import subprocess
import threading
import time


def port():
    """Choose an unused loopback port without touching existing listeners."""
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def wait_port(n, p=None):
    """Wait for a newly started test process and fail if it exits early."""
    for _ in range(200):
        if p is not None and p.poll() is not None:
            raise RuntimeError(f"process exited {p.returncode}")
        try:
            with socket.create_connection(("127.0.0.1", n), 0.1):
                return
        except OSError:
            time.sleep(0.05)
    raise TimeoutError(n)


def cluster(name, n):
    """Route one static cluster to a loopback test service."""
    return {
        "name": name,
        "type": "STATIC",
        "connect_timeout": "1s",
        "load_assignment": {
            "cluster_name": name,
            "endpoints": [
                {
                    "lb_endpoints": [
                        {
                            "endpoint": {
                                "address": {
                                    "socket_address": {
                                        "address": "127.0.0.1",
                                        "port_value": n,
                                    }
                                }
                            }
                        }
                    ]
                }
            ],
        },
    }


def wasm(name, path, config):
    """Load one separately compiled plugin with its own VM and configuration."""
    return {
        "name": "envoy.filters.http.wasm",
        "typed_config": {
            "@type": "type.googleapis.com/envoy.extensions.filters.http.wasm.v3.Wasm",
            "config": {
                "name": name,
                "root_id": name,
                "vm_config": {
                    "vm_id": name,
                    "runtime": "envoy.wasm.runtime.v8",
                    "code": {"local": {"filename": "/case/" + path}},
                },
                "configuration": {
                    "@type": "type.googleapis.com/google.protobuf.StringValue",
                    "value": json.dumps(config),
                },
            },
        },
    }


def run(root, image, mode, framing="length", kind="json", deny=False):
    """Check a split upload against real transformer and rate limiter filters."""
    echo_port, redis_port, admin_port, listener_port = port(), port(), port(), port()
    events = []
    requests = []
    lock = threading.Lock()

    class Echo(http.server.BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def log_message(self, *a):
            pass

        def do_POST(self):
            with lock:
                events.append(("upstream_headers", time.monotonic()))
            body = b""
            if self.headers.get("transfer-encoding", "").lower() == "chunked":
                while True:
                    size = int(self.rfile.readline().split(b";", 1)[0], 16)
                    if not size:
                        while self.rfile.readline() != b"\r\n":
                            pass
                        break
                    body += self.rfile.read(size)
                    self.rfile.read(2)
                    with lock:
                        events.append(("upstream_body", time.monotonic()))
            else:
                remaining = int(self.headers.get("content-length", "0"))
                while remaining:
                    data = self.rfile.read1(min(remaining, 65536))
                    if not data:
                        break
                    body += data
                    remaining -= len(data)
                    with lock:
                        events.append(("upstream_body", time.monotonic()))
            requests.append(body)
            self.send_response(200)
            self.send_header("content-length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    server = http.server.ThreadingHTTPServer(("127.0.0.1", echo_port), Echo)
    server.daemon_threads = True
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    processes = []
    files = []
    client = None
    label = f"{mode}-{framing}-{kind}-" + ("deny" if deny else "allow")
    try:
        redis = subprocess.Popen(
            [
                "redis-server",
                "--bind",
                "127.0.0.1",
                "--port",
                str(redis_port),
                "--save",
                "",
                "--appendonly",
                "no",
                "--logfile",
                str(root / (label + "-redis.log")),
            ],
            stdout=subprocess.DEVNULL,
        )
        processes.append(redis)
        wait_port(redis_port, redis)
        monitor_process = subprocess.Popen(
            ["redis-cli", "-p", str(redis_port), "MONITOR"],
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            bufsize=1,
        )
        processes.append(monitor_process)
        assert monitor_process.stdout.readline().strip() == "OK"

        def monitor():
            for line in monitor_process.stdout:
                if '"eval"' in line.lower():
                    with lock:
                        events.append(("redis_eval", time.monotonic()))

        threading.Thread(target=monitor, daemon=True).start()
        rule = "body-eval-" + label
        if deny:
            subprocess.run(
                [
                    "redis-cli",
                    "-p",
                    str(redis_port),
                    "SET",
                    "higress-token-ratelimit:{" + rule + "}:global_threshold:60",
                    "1001",
                ],
                stdout=subprocess.DEVNULL,
                check=True,
            )
        transforms = {
            "reqRules": [
                {
                    "operate": "replace",
                    "body": [{"key": "model", "newValue": "transformed"}],
                }
            ]
        }
        limits = {
            "rule_name": rule,
            "global_threshold": {"token_per_minute": 1000},
            "redis": {
                "service_name": "redis.static",
                "service_port": redis_port,
                "timeout": 2000,
            },
        }
        config = {
            "static_resources": {
                "listeners": [
                    {
                        "name": "ingress",
                        "address": {
                            "socket_address": {
                                "address": "127.0.0.1",
                                "port_value": listener_port,
                            }
                        },
                        "filter_chains": [
                            {
                                "filters": [
                                    {
                                        "name": "envoy.filters.network.http_connection_manager",
                                        "typed_config": {
                                            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                                            "stat_prefix": "body_test",
                                            "route_config": {
                                                "name": "local",
                                                "virtual_hosts": [
                                                    {
                                                        "name": "all",
                                                        "domains": ["*"],
                                                        "routes": [
                                                            {
                                                                "match": {
                                                                    "prefix": "/"
                                                                },
                                                                "route": {
                                                                    "cluster": "echo",
                                                                    "timeout": "10s",
                                                                },
                                                            }
                                                        ],
                                                    }
                                                ],
                                            },
                                            "http_filters": [
                                                wasm(
                                                    "transformer",
                                                    "transformer-" + mode + ".wasm",
                                                    transforms,
                                                ),
                                                wasm(
                                                    "rate-limit",
                                                    "ai-token-ratelimit.wasm",
                                                    limits,
                                                ),
                                                {
                                                    "name": "envoy.filters.http.router",
                                                    "typed_config": {
                                                        "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
                                                    },
                                                },
                                            ],
                                        },
                                    }
                                ]
                            }
                        ],
                    }
                ],
                "clusters": [
                    cluster("echo", echo_port),
                    cluster(
                        "outbound|" + str(redis_port) + "||redis.static", redis_port
                    ),
                ],
            },
            "admin": {
                "address": {
                    "socket_address": {"address": "127.0.0.1", "port_value": admin_port}
                }
            },
        }
        config_path = root / (label + ".json")
        config_path.write_text(json.dumps(config))
        log = open(root / (label + "-envoy.log"), "w")
        files.append(log)
        name = "wasm-body-eval-" + str(os.getpid())
        envoy = subprocess.Popen(
            [
                "docker",
                "run",
                "--rm",
                "--name",
                name,
                "--cpus",
                "2",
                "--memory",
                "1536m",
                "--network",
                "host",
                "-v",
                str(root) + ":/case:ro",
                "--entrypoint",
                "/usr/local/bin/envoy",
                image,
                "--disable-hot-restart",
                "-c",
                "/case/" + config_path.name,
                "--concurrency",
                "1",
                "--log-level",
                "debug",
            ],
            stdout=log,
            stderr=subprocess.STDOUT,
        )
        processes.append(envoy)
        wait_port(listener_port, envoy)
        body = (
            b'{"model":"original","message":"fragmented-request"}'
            if kind == "json"
            else b"first-part-second-part"
        )
        first, rest = body[:12], body[12:]
        client = socket.create_connection(("127.0.0.1", listener_port), 2)
        client.settimeout(10)
        ct = "application/json" if kind == "json" else "application/octet-stream"
        framing_header = (
            "Content-Length: " + str(len(body))
            if framing == "length"
            else "Transfer-Encoding: chunked"
        )
        client.sendall(
            (
                "POST / HTTP/1.1\r\nHost: test.local\r\nContent-Type: "
                + ct
                + "\r\n"
                + framing_header
                + "\r\nConnection: close\r\n\r\n"
            ).encode()
        )
        client.sendall(
            first
            if framing == "length"
            else ("%x\r\n" % len(first)).encode() + first + b"\r\n"
        )
        # Wait for the asynchronous result in baseline/skip cases. The fixed
        # buffering case must have no successor activity during this interval.
        expect_early = mode == "before" or kind != "json"
        deadline = time.monotonic() + (5 if expect_early else 0.25)
        while time.monotonic() < deadline:
            with lock:
                if expect_early and any(n == "upstream_body" for n, _ in events):
                    break
            time.sleep(0.01)
        with lock:
            early = [x[0] for x in events]
        if mode == "before" or kind != "json":
            assert "redis_eval" in early and "upstream_body" in early, (
                label,
                early,
                "successor did not forward the partial body",
            )
        else:
            assert not early, (label, early, "successor ran before complete body")
        sent_end = time.monotonic()
        client.sendall(
            rest
            if framing == "length"
            else ("%x\r\n" % len(rest)).encode()
            + rest
            + b"\r\n0\r\nx-checksum: ok\r\n\r\n"
        )
        response = http.client.HTTPResponse(client)
        response.begin()
        result = response.read()
        status = response.status
        client.close()
        deadline = time.monotonic() + 1
        while time.monotonic() < deadline:
            with lock:
                if any(n == "redis_eval" for n, _ in events):
                    break
            time.sleep(0.01)
        assert any(n == "redis_eval" for n, _ in events), (
            label,
            "no Redis call observed",
        )
        if deny:
            assert status == 429, (label, status, result)
            assert not requests, (label, "denied request reached upstream")
        else:
            assert status == 200, (label, status, result)
            if mode == "fixed" and kind == "json":
                assert json.loads(result)["model"] == "transformed", (label, result)
            else:
                assert result == body, (label, result)
        summary = {
            "case": label,
            "status": status,
            "early_events": early,
            "body": result.decode(errors="replace"),
            "upstream_requests": len(requests),
            "redis_after_body": all(
                t >= sent_end for n, t in events if n == "redis_eval"
            ),
        }
        print(json.dumps(summary), flush=True)
        return summary
    finally:
        if client is not None:
            client.close()
        if "name" in locals():
            subprocess.run(
                ["docker", "stop", "-t", "1", name],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        for p in reversed(processes):
            p.terminate()
            try:
                p.wait(timeout=3)
            except subprocess.TimeoutExpired:
                p.kill()
                p.wait()
        server.shutdown()
        server.server_close()
        for f in files:
            f.close()


def run_handshake(root, image, mode, protocol, required=False):
    """Require 100/101/CONNECT 200 before sending any request data."""
    hits = []

    class Upstream(http.server.BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def log_message(self, *args):
            pass

        def handle_expect_100(self):
            hits.append("expect_at_upstream")
            return super().handle_expect_100()

        def do_POST(self):
            hits.append("post_at_upstream")
            if self.headers.get("transfer-encoding", "").lower() == "chunked":
                body = b""
                while True:
                    size = int(self.rfile.readline().strip(), 16)
                    if not size:
                        self.rfile.readline()
                        break
                    body += self.rfile.read(size)
                    self.rfile.read(2)
            else:
                body = self.rfile.read(int(self.headers.get("content-length", "0")))
            self.send_response(200)
            self.send_header("content-length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            hits.append("websocket")
            self.send_response(101)
            self.send_header("Connection", "upgrade")
            self.send_header("Upgrade", "websocket")
            key = (
                self.headers["Sec-WebSocket-Key"]
                + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
            )
            self.send_header(
                "Sec-WebSocket-Accept",
                base64.b64encode(hashlib.sha1(key.encode()).digest()).decode(),
            )
            self.end_headers()
            self.wfile.flush()
            time.sleep(0.2)
            self.close_connection = True

        def do_CONNECT(self):
            hits.append("CONNECT")
            self.send_response(200)
            self.end_headers()
            self.wfile.flush()
            time.sleep(0.2)
            self.close_connection = True

    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Upstream)
    server.daemon_threads = True
    threading.Thread(target=server.serve_forever, daemon=True).start()
    lp, ap = port(), port()
    up = server.server_address[1]
    hcm = {
        "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
        "stat_prefix": "review",
        "proxy_100_continue": protocol != "expect-local",
        "upgrade_configs": [{"upgrade_type": "websocket"}, {"upgrade_type": "CONNECT"}],
        "route_config": {
            "name": "local",
            "virtual_hosts": [
                {
                    "name": "all",
                    "domains": ["*"],
                    "routes": [
                        {
                            "match": {"connect_matcher": {}},
                            "route": {"cluster": "upstream", "timeout": "10s"},
                        },
                        {
                            "match": {"prefix": "/"},
                            "route": {"cluster": "upstream", "timeout": "10s"},
                        },
                    ],
                }
            ],
        },
        "http_filters": [
            wasm("handshake-regression", "handshake-" + mode + ".wasm", {}),
            {
                "name": "envoy.filters.http.router",
                "typed_config": {
                    "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
                },
            },
        ],
    }
    config = {
        "static_resources": {
            "listeners": [
                {
                    "name": "review",
                    "address": {
                        "socket_address": {"address": "127.0.0.1", "port_value": lp}
                    },
                    "filter_chains": [
                        {
                            "filters": [
                                {
                                    "name": "envoy.filters.network.http_connection_manager",
                                    "typed_config": hcm,
                                }
                            ]
                        }
                    ],
                }
            ],
            "clusters": [cluster("upstream", up)],
        },
        "admin": {
            "address": {"socket_address": {"address": "127.0.0.1", "port_value": ap}}
        },
    }
    label = mode + "-" + protocol + "-" + str(required)
    cp = root / (label + ".json")
    cp.write_text(json.dumps(config))
    name = "wasm-handshake-eval-" + str(os.getpid())
    log = open(root / (label + ".log"), "w")
    p = None
    client = None
    try:
        p = subprocess.Popen(
            [
                "docker",
                "run",
                "--rm",
                "--name",
                name,
                "--cpus",
                "1",
                "--memory",
                "768m",
                "--network",
                "host",
                "-v",
                str(root) + ":/case:ro",
                "--entrypoint",
                "/usr/local/bin/envoy",
                image,
                "--disable-hot-restart",
                "-c",
                "/case/" + cp.name,
                "--concurrency",
                "1",
                "--log-level",
                "debug",
            ],
            stdout=log,
            stderr=subprocess.STDOUT,
        )
        wait_port(lp, p)
        client = socket.create_connection(("127.0.0.1", lp), 2)
        client.settimeout(2)
        body = b'{"model":"original"}'
        if protocol.startswith("expect"):
            request = (
                "POST / HTTP/1.1\r\nHost: test.local\r\nContent-Type: application/json\r\nContent-Length: "
                + str(len(body))
                + "\r\nExpect: 100-continue\r\nConnection: close\r\n"
            )
        elif protocol.startswith("websocket"):
            connection = (
                "keep-alive, UpGrAdE" if protocol == "websocket-tokens" else "Upgrade"
            )
            request = (
                "GET / HTTP/1.1\r\nHost: test.local\r\nConnection: "
                + connection
                + "\r\nUpgrade: websocket\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n"
            )
        else:
            request = "CONNECT test.local:443 HTTP/1.1\r\nHost: test.local:443\r\n"
        if required:
            request += "x-require-body: 1\r\n"
        client.sendall((request + "\r\n").encode())
        response = b""
        while b"\r\n\r\n" not in response:
            data = client.recv(65536)
            assert data, (label, "connection closed before handshake")
            response += data
        status_line = response.decode().split("\r\n")[0]
        before_body = list(hits)
        expected = (
            100
            if protocol.startswith("expect")
            else (101 if protocol.startswith("websocket") else 200)
        )
        assert int(status_line.split()[1]) == expected, (label, status_line)
        if protocol == "expect-proxy":
            assert "expect_at_upstream" in before_body, (label, before_body)
        if protocol == "expect-local" and mode == "fixed":
            # HCM sends local 100 before filter iteration. Allow an accidentally
            # continued header enough time to reach the upstream before checking.
            time.sleep(0.25)
            before_body = list(hits)
            assert not before_body, (
                label,
                "local 100 should keep the complete-body header barrier",
            )
        if protocol.startswith("expect"):
            client.sendall(body)
            final = http.client.HTTPResponse(client)
            final.begin()
            assert final.status == 200, (label, final.status)
            assert final.read() == body, (label, "body delivery changed")
        r = {
            "case": "handshake-" + label,
            "response": status_line,
            "upstream_before_body": before_body,
            "required_body": required,
        }
        print(json.dumps(r), flush=True)
        return r
    finally:
        if client:
            client.close()
        subprocess.run(
            ["docker", "stop", "-t", "1", name],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        if p:
            try:
                p.wait(timeout=3)
            except subprocess.TimeoutExpired:
                p.kill()
                p.wait()
        log.close()
        server.shutdown()
        server.server_close()


def build_plugin(higress_root, sdk_root, root, plugin, output):
    """Compile community sources using a temporary module file for this SDK."""
    source = higress_root / "plugins/wasm-go/extensions" / plugin
    modfile = root / (output + ".mod")
    shutil.copy2(source / "go.mod", modfile)
    shutil.copy2(source / "go.sum", modfile.with_suffix(".sum"))
    subprocess.run(
        [
            "go",
            "mod",
            "edit",
            "-modfile=" + str(modfile),
            "-replace=github.com/higress-group/wasm-go=" + str(sdk_root),
        ],
        cwd=source,
        check=True,
    )
    env = dict(os.environ, GOOS="wasip1", GOARCH="wasm", GOMAXPROCS="4")
    print("Building " + output, flush=True)
    subprocess.run(
        [
            "go",
            "build",
            "-p",
            "4",
            "-mod=mod",
            "-modfile=" + str(modfile),
            "-buildmode=c-shared",
            "-o",
            str(root / (output + ".wasm")),
            ".",
        ],
        cwd=source,
        env=env,
        check=True,
    )


def build_handshake_fixture(sdk_root, root, output):
    """Build the same handshake fixture against the selected SDK checkout."""
    source = root / (output + "-src")
    source.mkdir(parents=True, exist_ok=True)
    shutil.copy2(
        pathlib.Path(__file__).parent / "handshake/main.go", source / "main.go"
    )
    (source / "go.mod").write_text(
        "module handshake_regression\n\ngo 1.24.1\n\n"
        "require github.com/higress-group/wasm-go v0.0.0\n\n"
        "replace github.com/higress-group/wasm-go => " + str(sdk_root) + "\n"
    )
    env = dict(os.environ, GOOS="wasip1", GOARCH="wasm", GOMAXPROCS="4")
    print("Building " + output, flush=True)
    subprocess.run(
        [
            "go",
            "build",
            "-p",
            "4",
            "-mod=mod",
            "-buildmode=c-shared",
            "-o",
            str(root / (output + ".wasm")),
            ".",
        ],
        cwd=source,
        env=env,
        check=True,
    )


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--higress-root", type=pathlib.Path, required=True)
    parser.add_argument("--envoy-image", required=True)
    parser.add_argument("--work-dir", type=pathlib.Path, required=True)
    parser.add_argument(
        "--baseline-sdk",
        type=pathlib.Path,
        help="Optionally reproduce the bug with an unpatched SDK checkout",
    )
    args = parser.parse_args()
    root = args.work_dir.resolve()
    root.mkdir(parents=True, exist_ok=True)
    image = args.envoy_image
    sdk = pathlib.Path(__file__).resolve().parents[2]
    higress = args.higress_root.resolve()
    build_plugin(higress, sdk, root, "transformer", "transformer-fixed")
    build_plugin(higress, sdk, root, "ai-token-ratelimit", "ai-token-ratelimit")
    build_handshake_fixture(sdk, root, "handshake-fixed")
    modes = ["fixed"]
    cases = []
    if args.baseline_sdk:
        build_plugin(
            higress,
            args.baseline_sdk.resolve(),
            root,
            "transformer",
            "transformer-before",
        )
        cases.append(("before", "length", "json", False))
        build_handshake_fixture(args.baseline_sdk.resolve(), root, "handshake-before")
        modes.insert(0, "before")
    cases.extend(
        [
            ("fixed", "length", "json", False),
            ("fixed", "trailers", "json", False),
            ("fixed", "length", "binary", False),
            ("fixed", "length", "json", True),
        ]
    )
    results = [run(root, image, *case) for case in cases]
    for mode in modes:
        for protocol, required in [
            ("expect-proxy", False),
            ("expect-local", False),
            ("websocket", False),
            ("websocket", True),
            ("websocket-tokens", False),
            ("websocket-tokens", True),
            ("connect", False),
            ("connect", True),
        ]:
            results.append(run_handshake(root, image, mode, protocol, required))
    (root / "results.json").write_text(json.dumps(results, indent=2) + "\n")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Collect the local, explicit v0.8.0 lifecycle experiment baseline."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import platform
import re
import socket
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


METRICS = (
    "new_seconds",
    "new_to_ready_seconds",
    "check_fast_seconds",
    "check_deep_seconds",
    "upgrade_check_json_diff_seconds",
)


def command(
    args: list[str],
    *,
    cwd: Path,
    env: dict[str, str] | None = None,
    timeout: float,
) -> tuple[float, str, str]:
    started = time.perf_counter()
    completed = subprocess.run(
        args,
        cwd=cwd,
        env=env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=timeout,
        check=False,
    )
    elapsed = time.perf_counter() - started
    if completed.returncode != 0:
        raise RuntimeError(
            f"{' '.join(args)} failed with exit {completed.returncode}:\n"
            f"{completed.stdout}{completed.stderr}"
        )
    return elapsed, completed.stdout, completed.stderr


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return int(listener.getsockname()[1])


def wait_for_ready(process: subprocess.Popen[str], url: str, timeout: float) -> float:
    started = time.perf_counter()
    while time.perf_counter() - started < timeout:
        if process.poll() is not None:
            stdout, stderr = process.communicate(timeout=1)
            raise RuntimeError(f"generated server exited early:\n{stdout}{stderr}")
        try:
            with urllib.request.urlopen(url, timeout=0.3) as response:
                if 200 <= response.status < 300:
                    return time.perf_counter() - started
        except (urllib.error.URLError, TimeoutError):
            time.sleep(0.05)
    raise RuntimeError(f"generated server did not become ready within {timeout:.1f}s")


def run_once(args: argparse.Namespace, run_number: int) -> dict[str, Any]:
    framework_root = args.framework_root.resolve()
    jimu = args.jimu.resolve()
    with tempfile.TemporaryDirectory(prefix="jimu-v080-benchmark-") as raw_temp:
        temp = Path(raw_temp)
        project = temp / "basic-api"
        env = os.environ.copy()
        env["GOWORK"] = "off"
        started = time.perf_counter()
        new_seconds, _, _ = command(
            [
                str(jimu),
                "new",
                str(project),
                "--template=basic-api",
                "--module=example.com/jimu-benchmark",
                f"--from={framework_root}",
            ],
            cwd=temp,
            env=env,
            timeout=args.timeout,
        )

        server_binary = temp / "server"
        _, _, _ = command(
            ["go", "build", "-o", str(server_binary), "./cmd/server"],
            cwd=project,
            env=env,
            timeout=args.timeout,
        )
        http_port = free_port()
        management_port = free_port()
        while management_port == http_port:
            management_port = free_port()
        server_env = env.copy()
        server_env.update(
            {
                "HTTP_HOST": "127.0.0.1",
                "HTTP_PORT": str(http_port),
                "MANAGEMENT_HOST": "127.0.0.1",
                "MANAGEMENT_PORT": str(management_port),
                "APP_ENV": "test",
                "LOG_LEVEL": "error",
            }
        )
        process = subprocess.Popen(
            [str(server_binary)],
            cwd=project,
            env=server_env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        try:
            readiness_seconds = wait_for_ready(
                process,
                f"http://127.0.0.1:{management_port}/readyz",
                args.timeout,
            )
            new_to_ready_seconds = time.perf_counter() - started
        finally:
            process.terminate()
            try:
                process.communicate(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.communicate()

        check_args = [
            str(jimu),
            "check",
            str(project),
            "--json",
            f"--from={framework_root}",
        ]
        check_fast_seconds, first_check, _ = command(
            check_args, cwd=framework_root, env=env, timeout=args.timeout
        )
        _, second_check, _ = command(
            check_args, cwd=framework_root, env=env, timeout=args.timeout
        )
        check_deep_seconds, _, _ = command(
            check_args + ["--deep"],
            cwd=framework_root,
            env=env,
            timeout=args.timeout,
        )

        upgrade_args = [
            str(jimu),
            "upgrade",
            str(project),
            "--check",
            "--json",
            "--diff",
            f"--from={framework_root}",
        ]
        upgrade_seconds, first_upgrade, _ = command(
            upgrade_args, cwd=framework_root, env=env, timeout=args.timeout
        )
        _, second_upgrade, _ = command(
            upgrade_args, cwd=framework_root, env=env, timeout=args.timeout
        )
        return {
            "run": run_number,
            "success": True,
            "new_seconds": new_seconds,
            "new_to_ready_seconds": new_to_ready_seconds,
            "readiness_seconds": readiness_seconds,
            "check_fast_seconds": check_fast_seconds,
            "check_deep_seconds": check_deep_seconds,
            "upgrade_check_json_diff_seconds": upgrade_seconds,
            "check_json_deterministic": first_check == second_check,
            "upgrade_json_deterministic": first_upgrade == second_upgrade,
        }


def summarize(values: list[float]) -> dict[str, float]:
    return {
        "median_seconds": round(statistics.median(values), 3),
        "min_seconds": round(min(values), 3),
        "max_seconds": round(max(values), 3),
        "spread_seconds": round(max(values) - min(values), 3),
    }


def tool_version(args: list[str], cwd: Path) -> str:
    completed = subprocess.run(
        args,
        cwd=cwd,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if completed.returncode != 0:
        return "unknown"
    return completed.stdout.strip().splitlines()[0]


def sanitize_error(error: Exception, framework_root: Path, temp_root: Path) -> str:
    text = str(error)
    text = (
        text.replace(str(temp_root), "<temp>")
        .replace(str(framework_root), "<repository>")
        .replace(str(Path.home()), "<home>")
    )
    return re.sub(r"jimu-v080-benchmark-[^/\s]+", "<temp>", text)


def git_provenance(root: Path) -> dict[str, Any]:
    revision = subprocess.run(
        ["git", "rev-parse", "HEAD"],
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    status = subprocess.run(
        ["git", "status", "--porcelain"],
        cwd=root,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    return {
        "commit": revision.stdout.strip() if revision.returncode == 0 else "unknown",
        "dirty": bool(status.stdout.strip()) if status.returncode == 0 else None,
    }


def validate_runs(runs: int) -> None:
    if runs < 3:
        raise ValueError("--runs must be at least 3")


def self_test() -> None:
    assert summarize([1.0, 2.0, 3.0]) == {
        "median_seconds": 2.0,
        "min_seconds": 1.0,
        "max_seconds": 3.0,
        "spread_seconds": 2.0,
    }
    try:
        validate_runs(2)
    except ValueError:
        pass
    else:
        raise AssertionError("fewer than three runs must be rejected")
    print("experiment script self-test passed")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--jimu", type=Path, help="v0.8.0 development CLI binary")
    parser.add_argument(
        "--framework-root",
        type=Path,
        default=Path(__file__).resolve().parent.parent,
        help="Jimu checkout used as the local template source",
    )
    parser.add_argument("--runs", type=int, default=3)
    parser.add_argument("--timeout", type=float, default=180)
    parser.add_argument("--output", type=Path, help="explicit local JSON output path")
    parser.add_argument("--version", default="v0.8.0")
    parser.add_argument("--self-test", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.self_test:
        self_test()
        return 0
    validate_runs(args.runs)
    if args.jimu is None:
        raise SystemExit("--jimu is required")
    if args.output is None:
        raise SystemExit("--output is required")

    framework_root = args.framework_root.resolve()
    records: list[dict[str, Any]] = []
    for number in range(1, args.runs + 1):
        try:
            records.append(run_once(args, number))
        except Exception as error:  # noqa: BLE001 - the run record must capture failures.
            records.append(
                {
                    "run": number,
                    "success": False,
                    "error": sanitize_error(error, framework_root, framework_root),
                }
            )
    successful = [record for record in records if record["success"]]
    report = {
        "schema_version": 1,
        "version": args.version,
        "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "environment": {
            "platform": platform.platform(),
            "system": platform.system(),
            "machine": platform.machine(),
            "python": platform.python_version(),
            "go": tool_version(["go", "version"], framework_root),
            "jimu": tool_version([str(args.jimu.resolve()), "version"], framework_root),
            "cpu_count": os.cpu_count(),
            "git": git_provenance(framework_root),
        },
        "inputs": {
            "template": "basic-api",
            "runs": args.runs,
            "network_upload": False,
        },
        "runs": records,
        "summary": {
            metric: summarize([float(record[metric]) for record in successful])
            for metric in METRICS
        }
        if successful
        else {},
        "success_rate": round(len(successful) / len(records), 3),
        "check_json_deterministic_rate": round(
            sum(bool(record.get("check_json_deterministic")) for record in records)
            / len(records),
            3,
        ),
        "upgrade_json_deterministic_rate": round(
            sum(bool(record.get("upgrade_json_deterministic")) for record in records)
            / len(records),
            3,
        ),
        "anomalies": [
            {"run": record["run"], "summary": record["error"]}
            for record in records
            if not record["success"]
        ],
    }
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    if len(successful) != len(records):
        return 1
    if any(not record["check_json_deterministic"] for record in successful):
        return 1
    if any(not record["upgrade_json_deterministic"] for record in successful):
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

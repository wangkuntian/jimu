#!/usr/bin/env python3
"""Keep a managed release Issue current and close it after publication."""

import argparse
from datetime import datetime
import json
import os
import re
import subprocess
import sys
from urllib.parse import quote, urlencode


VERSION_PATTERN = r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
STATUS_START = "<!-- jimu-release-status:start -->"
STATUS_END = "<!-- jimu-release-status:end -->"
ACTIVE_LABELS = {"release: collecting", "release: candidate"}
TERMINAL_LABELS = {"release: blocked", "release: published"}


class ReleaseError(Exception):
    pass


class GitHub:
    def __init__(self):
        self.repo = os.environ.get("GH_REPO", "")
        if not re.fullmatch(r"[\w.-]+/[\w.-]+", self.repo):
            raise ReleaseError("GH_REPO must be owner/repository")

    def request(self, path, method="GET", payload=None):
        command = ["gh", "api", f"repos/{self.repo}/{path}", "--method", method]
        if payload is not None:
            command += ["--input", "-"]
        result = subprocess.run(command, input=json.dumps(payload) if payload is not None else None,
                                capture_output=True, text=True)
        if result.returncode:
            raise ReleaseError(f"GitHub {method} {path} failed: {result.stderr.strip()}")
        try:
            return json.loads(result.stdout)
        except json.JSONDecodeError as error:
            raise ReleaseError(f"GitHub {method} {path} returned invalid JSON") from error

    def pages(self, path, **query):
        page = 1
        while True:
            batch = self.request(path + "?" + urlencode(dict(query, per_page=100, page=page)))
            if not isinstance(batch, list):
                raise ReleaseError(f"GitHub {path} did not return a list")
            yield from batch
            if len(batch) < 100:
                break
            page += 1


def labels(issue):
    return {label["name"] for label in issue.get("labels", [])}


def timestamp(value):
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except (AttributeError, ValueError) as error:
        raise ReleaseError(f"invalid GitHub timestamp: {value}") from error


def find_cycle(github, version=None):
    candidates = []
    for issue in github.pages("issues", state="all" if version else "open"):
        if "pull_request" in issue or issue.get("author_association") != "OWNER":
            continue
        match = re.fullmatch(r"release: (" + VERSION_PATTERN + r")", issue.get("title", ""))
        if not match or (version and match[1] != version):
            continue
        if not version and (issue.get("state") != "open" or
                            not labels(issue) & ACTIVE_LABELS or labels(issue) & TERMINAL_LABELS):
            continue
        marker = f"<!-- jimu-release-automation:{match[1]} -->"
        comments = [comment for comment in github.pages(f"issues/{issue['number']}/comments")
                    if marker in (comment.get("body") or "")]
        if comments:
            started = min(timestamp(comment.get("created_at")) for comment in comments)
            candidates.append((issue, match[1], started))
    if len(candidates) > 1:
        raise ReleaseError("multiple managed release Issues match; refusing an ambiguous cycle")
    if not candidates and version:
        raise ReleaseError(f"no managed release Issue matches {version}")
    return candidates[0] if candidates else None


def write_output(**values):
    result = "".join(f"{key}={value}\n" for key, value in values.items())
    print(result, end="")
    output = os.environ.get("GITHUB_OUTPUT")
    if output:
        with open(output, "a", encoding="utf-8") as stream:
            stream.write(result)


def active(github):
    cycle = find_cycle(github)
    if cycle:
        write_output(active="true", issue_number=cycle[0]["number"], version=cycle[1])
    else:
        write_output(active="false")


def in_cycle(pull, started, issue):
    created = timestamp(pull.get("created_at"))
    return created >= started and (not issue.get("closed_at") or created <= timestamp(issue["closed_at"]))


def pull_category(pull, version, started, issue, repo):
    head = pull.get("head") or {}
    base = pull.get("base") or {}
    if (base.get("repo") or {}).get("full_name") != repo:
        return None
    release_branch = "release/" + version
    if base.get("ref") == release_branch:
        if head.get("ref") == "dependabot-updates":
            return "aggregate"
        if re.match(r"fix(?:[/( :]|$)", head.get("ref", "")) or re.match(r"fix(?:[(: ]|$)", pull.get("title", "")):
            return "fix"
        if re.match(r"(?:feat|feature)(?:[/( :]|$)", head.get("ref", "")) or re.match(r"feat(?:[(: ]|$)", pull.get("title", "")):
            return "feature"
        return "change"
    if head.get("ref") == release_branch and base.get("ref") == "master" and (head.get("repo") or {}).get("full_name") == repo:
        return "final"
    author = (pull.get("user") or {}).get("login")
    if author == "dependabot[bot]" and base.get("ref") in {"dependabot-updates", "master"}:
        if in_cycle(pull, started, issue):
            return "dependency" if base["ref"] == "dependabot-updates" else "security"
    app_slug = os.environ.get("RELEASE_APP_SLUG")
    if app_slug and author == app_slug + "[bot]" and base.get("ref") == "dependabot-updates":
        if (head.get("ref", "").startswith(f"dependabot-cli/{version}/") and
                (head.get("repo") or {}).get("full_name") == repo and
                re.search(r"<!-- jimu-release-automation:" + re.escape(version) + r" dependency:[^\s]+ -->", pull.get("body") or "")):
            return "dependency"
    return None


def cycle_status(issue):
    current = labels(issue)
    for status in ("published", "blocked", "candidate", "collecting"):
        if "release: " + status in current:
            return status
    return issue.get("state", "unknown")


def status_block(github, cycle, release_url=None, published=False):
    issue, version, started = cycle
    lines = [STATUS_START, f"## Release automation · {version}", "",
             f"- Status: `{ 'published' if published else cycle_status(issue) }`",
             f"- Release branch: `release/{version}`", "- Dependency branch: `dependabot-updates`"]
    if release_url:
        lines.append(f"- Release: [{version}]({release_url})")
    else:
        lines.append("- Release: pending")
    lines += ["", "| Kind | Pull request | Status |", "| --- | --- | --- |"]
    related = {}
    for pull in github.pages("pulls", state="all"):
        category = pull_category(pull, version, started, issue, github.repo)
        if category:
            related[pull["number"]] = (category, pull)
    for number, (category, pull) in sorted(related.items()):
        state = "merged" if pull.get("merged_at") else pull.get("state", "unknown")
        title = (pull.get("title") or "").replace("\n", " ").replace("\r", " ")
        title = title.replace("|", r"\|").replace("[", r"\[").replace("]", r"\]")
        lines.append(f"| {category} | [#{number} {title}]({pull['html_url']}) | {state} |")
    if not related:
        lines.append("| — | No related pull requests | — |")
    lines += ["", STATUS_END]
    return "\n".join(lines)


def replace_status(body, block):
    body = body or ""
    if STATUS_START not in body and STATUS_END not in body:
        return body + ("\n\n" if body else "") + block
    if body.count(STATUS_START) != 1 or body.count(STATUS_END) != 1:
        raise ReleaseError("release status markers are incomplete or duplicated")
    start = body.index(STATUS_START)
    end = body.index(STATUS_END)
    if start > end:
        raise ReleaseError("release status markers are out of order")
    return body[:start] + block + body[end + len(STATUS_END):]


def fresh_issue(github, cycle):
    issue = github.request(f"issues/{cycle[0]['number']}")
    if (issue.get("title") != "release: " + cycle[1] or
            issue.get("author_association") != "OWNER" or "pull_request" in issue):
        raise ReleaseError("managed release Issue identity changed during synchronization")
    return issue


def update_body(github, cycle, release_url=None, published=False):
    block = status_block(github, cycle, release_url, published)
    issue = fresh_issue(github, cycle)
    body = replace_status(issue.get("body"), block)
    if body != (issue.get("body") or ""):
        github.request(f"issues/{issue['number']}", "PATCH", {"body": body})


def published_release(github, version, release_url=None):
    release = github.request("releases/tags/" + version)
    if (not isinstance(release, dict) or release.get("tag_name") != version or
            release.get("draft") is not False or not release.get("published_at") or
            not release.get("html_url") or (release_url and release["html_url"] != release_url)):
        raise ReleaseError(f"{version} does not have a matching published GitHub Release")
    return release["html_url"]


def sync(github, version=None):
    cycle = find_cycle(github, version)
    if not cycle:
        write_output(active="false")
        return
    release_url = None
    if "release: published" in labels(cycle[0]):
        release_url = published_release(github, cycle[1])
    update_body(github, cycle, release_url)
    write_output(issue_number=cycle[0]["number"], version=cycle[1])


def publish(github, version, release_url):
    cycle = find_cycle(github, version)
    verified_url = published_release(github, version, release_url)
    issue = cycle[0]
    update_body(github, cycle, verified_url, published=True)
    path = f"issues/{issue['number']}"
    current = labels(fresh_issue(github, cycle))
    if "release: published" not in current:
        github.request(path + "/labels", "POST", {"labels": ["release: published"]})
    for label in sorted(current & (ACTIVE_LABELS | {"release: blocked"})):
        github.request(path + "/labels/" + quote(label, safe=""), "DELETE")
    if fresh_issue(github, cycle).get("state") != "closed":
        github.request(path, "PATCH", {"state": "closed", "state_reason": "completed"})
    write_output(issue_number=issue["number"], version=version, release_url=verified_url)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("active")
    sync_parser = commands.add_parser("sync")
    sync_parser.add_argument("version", nargs="?")
    publish_parser = commands.add_parser("published")
    publish_parser.add_argument("version")
    publish_parser.add_argument("release_url")
    args = parser.parse_args()
    if getattr(args, "version", None) and not re.fullmatch(VERSION_PATTERN, args.version):
        raise ReleaseError(f"invalid release version: {args.version}")
    github = GitHub()
    if args.command == "active":
        active(github)
    elif args.command == "sync":
        sync(github, args.version)
    else:
        publish(github, args.version, args.release_url)


if __name__ == "__main__":
    try:
        main()
    except (ReleaseError, OSError) as error:
        print(f"::error::{error}", file=sys.stderr)
        sys.exit(1)

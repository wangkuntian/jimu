#!/usr/bin/env python3
"""Prepare official Dependabot jobs and publish their JSONL results via GitHub REST."""

import argparse
import base64
import binascii
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess
import sys
from urllib.parse import urlencode


BASE_BRANCH = "dependabot-updates"
ECOSYSTEMS = {"go": "go_modules", "github-actions": "github_actions", "docker": "docker"}
SHA = re.compile(r"[0-9a-f]{40}\Z")
TREE_MARKER = re.compile(r"<!-- jimu-dependabot-tree:([0-9a-f]{40}) -->")
METADATA_EVENTS = {"update_dependency_list", "create_dependency_submission", "record_ecosystem_versions",
                   "record_ecosystem_meta", "record_update_job_warning", "increment_metric"}


class Error(Exception):
    pass


class ApiError(Error):
    def __init__(self, message, missing=False):
        super().__init__(message)
        self.missing = missing


def require(condition, message):
    if not condition:
        raise Error(message)


class GitHub:
    def __init__(self, repo):
        self.prefix = "repos/" + repo + "/"

    def api(self, endpoint, method="GET", payload=None, query=None):
        if query:
            endpoint += "?" + urlencode(query)
        args = ["gh", "api", self.prefix + endpoint, "--method", method]
        data = None
        if payload is not None:
            args += ["--input", "-"]
            data = json.dumps(payload, ensure_ascii=False)
        result = subprocess.run(args, input=data, text=True, capture_output=True)
        if result.returncode:
            raise ApiError("GitHub API failed: " + method + " " + endpoint,
                           missing="HTTP 404" in result.stderr)
        try:
            return json.loads(result.stdout)
        except ValueError as error:
            raise Error("GitHub API returned invalid JSON: " + endpoint) from error

    def list(self, endpoint, **query):
        result = []
        page = 1
        while True:
            batch = self.api(endpoint, query=dict(query, per_page=100, page=page))
            require(isinstance(batch, list), "GitHub list response is not an array")
            result.extend(batch)
            if len(batch) < 100:
                return result
            page += 1

    def ref(self, branch, optional=False):
        try:
            result = self.api("git/ref/heads/" + branch)
        except ApiError as error:
            if optional and error.missing:
                return None
            raise
        sha = result.get("object", {}).get("sha", "")
        require(SHA.fullmatch(sha), "GitHub ref has invalid commit SHA")
        return sha

    def tree(self, sha):
        commit = self.api("git/commits/" + sha)
        tree_sha = commit.get("tree", {}).get("sha", "")
        require(SHA.fullmatch(tree_sha), "GitHub commit has invalid tree SHA")
        tree = self.api("git/trees/" + tree_sha, query={"recursive": 1})
        require(tree.get("truncated") is False, "GitHub recursive tree is truncated")
        return commit, {e["path"]: e for e in tree["tree"] if e["type"] != "tree"}


class Context:
    def __init__(self, publishing=False):
        self.repo = os.environ.get("GH_REPO", "")
        self.version = os.environ.get("RELEASE_VERSION", "")
        self.issue = os.environ.get("RELEASE_ISSUE_NUMBER", "")
        self.slug = os.environ.get("RELEASE_APP_SLUG", "")
        require(re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", self.repo), "GH_REPO is required")
        require(re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", self.version), "RELEASE_VERSION must be vMAJOR.MINOR.PATCH")
        require(re.fullmatch(r"[1-9][0-9]*", self.issue), "RELEASE_ISSUE_NUMBER is required")
        if publishing:
            require(re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", self.slug), "RELEASE_APP_SLUG is required")
        self.github = GitHub(self.repo)

    def active(self):
        issue = self.github.api("issues/" + self.issue)
        require(issue.get("state") == "open" and "pull_request" not in issue,
                "release Issue must be open")
        require(issue.get("author_association") == "OWNER", "release Issue must belong to the repository owner")
        require(issue.get("title") == "release: " + self.version, "release Issue title does not match version")
        labels = {label["name"] for label in issue.get("labels", [])}
        require(not labels & {"release: published", "release: blocked"}, "release Issue is terminal or blocked")
        require(labels & {"release: collecting", "release: candidate"}, "release Issue is not collecting or candidate")
        marker = "<!-- jimu-release-automation:" + self.version + " -->"
        if self.slug:
            comments = self.github.list("issues/" + self.issue + "/comments")
            require(any(marker in (comment.get("body") or "") and
                        comment.get("user", {}).get("login") == self.slug + "[bot]" for comment in comments),
                    "release Issue is not managed by the release App")

    def guard(self, base):
        self.active()
        require(self.github.ref(BASE_BRANCH) == base, "Dependabot base ref changed; run a fresh scan")

    def author(self):
        return {"name": self.slug + "[bot]", "email": self.slug + "[bot]@users.noreply.github.com"}


def output(key, value):
    line = key + "=" + str(value) + "\n"
    sys.stdout.write(line)
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as handle:
            handle.write(line)


def prepare(config_path, input_path):
    context = Context()
    config = json.loads(Path(config_path).read_text(encoding="utf-8"))
    require(isinstance(config, dict) and isinstance(config.get("job"), dict), "config must contain one official job")
    job = config["job"]
    require(job.get("package-manager") in ECOSYSTEMS.values(), "unsupported package manager")
    require(isinstance(job.get("source"), dict) and job["source"].get("directory") == "/", "job source directory must be /")
    context.active()
    base = context.github.ref(BASE_BRANCH)
    job["source"].update(provider="github", repo=context.repo, branch=BASE_BRANCH, commit=base)
    # Recompute from the base daily. Supplying existing PRs can suppress new create events
    # when an open PR needs a newer version; apply owns the idempotent reconciliation.
    job.pop("existing-pull-requests", None)
    job.pop("existing-group-pull-requests", None)
    config["credentials"] = [{"type": "git_source", "host": "github.com", "username": "x-access-token",
                              "password": "$LOCAL_GITHUB_ACCESS_TOKEN"}]
    Path(input_path).write_text(json.dumps(config, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    output("base_sha", base)


def dependency_names(data, kind):
    if kind == "create_pull_request":
        dependencies = data.get("dependencies")
        require(isinstance(dependencies, list), "create event has no dependencies")
        names = [dependency.get("name") for dependency in dependencies]
    else:
        names = data.get("dependency-names")
    require(isinstance(names, list) and names and all(isinstance(n, str) and n and n.isascii()
            and not any(ord(c) < 32 or ord(c) == 127 for c in n) for n in names), "invalid dependency names")
    require(len(set(names)) == len(names), "duplicate dependency names")
    return sorted(names)


def file_change(file, ecosystem):
    require(isinstance(file, dict), "dependency file must be an object")
    directory, name = file.get("directory"), file.get("name")
    require(isinstance(directory, str) and isinstance(name, str) and name, "invalid dependency file path")
    # Dependabot directories are repo-root-relative and conventionally start with /.
    # Names must remain relative; normalization must never silently remove traversal.
    require(not directory.startswith("//") and not name.startswith("/") and "\\" not in directory + name,
            "absolute or invalid dependency file path")
    relative_dir = directory[1:] if directory.startswith("/") else directory
    components = relative_dir.split("/") + name.split("/")
    require(not any(part in {".", "..", ".git"} for part in components), "dependency file path traversal")
    require(not any(ord(c) < 32 or ord(c) == 127 for c in directory + name), "invalid dependency file path")
    path = str(PurePosixPath(relative_dir) / name)
    if ecosystem == "go":
        allowed = path in {"go.mod", "go.sum"}
    elif ecosystem == "github-actions":
        allowed = bool(re.fullmatch(r"\.github/workflows/[^/]+\.ya?ml", path))
    else:
        allowed = bool(re.fullmatch(r"(?:[^/]+/)*(?:Dockerfile(?:\.[^/]+)?|(?:docker-)?compose(?:\.[^/]+)?\.ya?ml)", path))
    require(allowed, "dependency file is outside ecosystem scope: " + path)
    require(file.get("type", "file") == "file" and not file.get("symlink_target") and not file.get("support_file", False),
            "only regular dependency files can be published")
    require(file.get("mode", "100644") in {"", "100644"}, "dependency file mode is unsupported")
    deleted = file.get("deleted", False)
    require(isinstance(deleted, bool), "deleted must be boolean")
    operation = file.get("operation", "delete" if deleted else "update")
    require(operation in {"update", "create", "delete"} and (operation == "delete") == deleted,
            "dependency file operation is inconsistent")
    if deleted:
        return path, None
    content = file.get("content")
    require(isinstance(content, str), "dependency file has no content")
    encoding = file.get("content_encoding", "utf-8")
    if encoding == "utf-8":
        content = content.encode("utf-8")
    elif encoding == "base64":
        try:
            content = base64.b64decode(content, validate=True)
        except (ValueError, binascii.Error) as error:
            raise Error("invalid base64 dependency content") from error
    else:
        raise Error("unsupported dependency content encoding")
    return path, content


def events_from(path, ecosystem, base):
    updates = []
    completed = False
    seen = set()
    for line in Path(path).read_text(encoding="utf-8").splitlines():
        if not line.strip():
            continue
        event = json.loads(line)
        require(isinstance(event, dict), "JSONL event must be an object")
        kind, data = event.get("type"), event.get("data")
        if kind in {"record_update_job_error", "record_update_job_unknown_error"}:
            raise Error("Dependabot job reported an error; no updates will be published")
        if kind in METADATA_EVENTS:
            continue
        require(isinstance(data, dict), "JSONL event data must be an object")
        if kind == "mark_as_processed":
            require(data.get("base-commit-sha") == base, "completion event has wrong base SHA")
            completed = True
            continue
        require(kind in {"create_pull_request", "update_pull_request"}, "unsupported Dependabot event: " + str(kind))
        require(data.get("base-commit-sha") == base, "dependency event has wrong base SHA")
        names = dependency_names(data, kind)
        key = hashlib.sha256("\0".join(names).encode()).hexdigest()[:16]
        require(key not in seen, "duplicate dependency set in CLI results")
        seen.add(key)
        files = data.get("updated-dependency-files")
        require(isinstance(files, list) and files, "dependency event has no updated files")
        changes = [file_change(file, ecosystem) for file in files]
        require(len({path for path, _ in changes}) == len(changes), "duplicate dependency file path")
        title = data.get("pr-title")
        require(isinstance(title, str) and title.strip() and len(title) <= 256 and title.isascii()
                and not any(ord(c) < 32 or ord(c) == 127 for c in title), "invalid English PR title")
        body = data.get("pr-body", "")
        require(isinstance(body, str) and "<!-- jimu-release-automation:" not in body
                and "<!-- jimu-dependabot-tree:" not in body, "invalid PR body")
        summary = title[0].lower() + title[1:]
        if not summary.startswith(("bump ", "update ")):
            summary = "update " + ", ".join(names)
        updates.append({"kind": kind, "key": key, "names": names, "changes": changes,
                        "title": title, "body": body, "message": "chore(deps): " + summary})
    require(completed, "Dependabot JSONL is missing mark_as_processed; refusing partial results")
    return updates


def git_blob_sha(content):
    return hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()


def same_tree(left, right):
    def normalized(entries):
        return {p: (e["mode"], e["type"], e["sha"]) for p, e in entries.items()}
    return normalized(left) == normalized(right)


def app_commit(context, commit, message):
    return (commit.get("message") in {message, message.partition("\n\n")[0]} and
            all(commit.get(role, {}).get(field) == value for role in ("author", "committer")
                for field, value in context.author().items()))


def validate_recovery_commit(context, commit, tree, update, marker, base):
    message = commit.get("message", "")
    require(isinstance(message, str) and app_commit(context, commit, message),
            "existing dependency branch commit is not authored by the release App")
    legacy_message = update["message"].partition("\n\n")[0]
    identity = message == legacy_message
    if "\n\n" in message:
        summary, body = message.split("\n\n", 1)
        identity = body == marker and bool(re.fullmatch(r"chore\(deps\): (?:bump|update) [ -~]+", summary))
    elif len(update["names"]) == 1:
        # Earlier publisher commits have no identity marker. Their exact
        # dependency name still identifies a single update across versions.
        identity = bool(re.fullmatch(r"chore\(deps\): (?:bump|update) " + re.escape(update["names"][0]) +
                                     r"(?: from [ -~]+ to [ -~]+)?", message))
    require(identity, "existing dependency branch has different dependency identities")
    parents = [parent.get("sha") for parent in commit.get("parents", [])]
    require(len(parents) in {1, 2} and all(SHA.fullmatch(parent or "") for parent in parents),
            "interrupted dependency branch has unexpected parents")
    old_base = parents[-1]
    comparison = context.github.api("compare/" + old_base + "..." + base)
    require(comparison.get("merge_base_commit", {}).get("sha") == old_base,
            "orphan dependency baseline is not an ancestor of the current base")
    _, old_tree = context.github.tree(old_base)
    changed = {path for path in old_tree.keys() | tree.keys()
               if not same_tree({path: old_tree[path]} if path in old_tree else {},
                                {path: tree[path]} if path in tree else {})}
    require(changed and changed <= {path for path, _ in update["changes"]},
            "orphan dependency branch changes files outside the scanned update")
    for path in changed:
        for entries in (old_tree, tree):
            entry = entries.get(path)
            require(entry is None or (entry["type"] == "blob" and entry["mode"] == "100644"),
                    "orphan dependency path is not a regular file")
    return parents


def recover_orphan(context, commit, tree, update, marker, base):
    parents = validate_recovery_commit(context, commit, tree, update, marker, base)
    if len(parents) == 2:
        # A retry can itself stop after advancing the ref. Verify its immediate
        # predecessor as another App update, without walking arbitrary history.
        previous, previous_tree = context.github.tree(parents[0])
        validate_recovery_commit(context, previous, previous_tree, update, marker, base)


def validate_pull(context, pull, branch, marker, state="open"):
    require(pull.get("state") == state and pull.get("user", {}).get("login") == context.slug + "[bot]",
            "existing dependency PR has an unmanaged author or state")
    require(marker in (pull.get("body") or ""), "existing dependency PR is missing managed marker")
    require(pull.get("head", {}).get("ref") == branch and pull.get("base", {}).get("ref") == BASE_BRANCH,
            "existing dependency PR has unexpected branches")
    require(all(pull.get(side, {}).get("repo", {}).get("full_name") == context.repo for side in ("head", "base")),
            "dependency PR must belong to this repository")


def plan_updates(context, ecosystem, updates, base):
    github = context.github
    base_commit, base_tree = github.tree(base)
    pulls = github.list("pulls", state="all")
    plans = []
    for update in updates:
        branch = "dependabot-cli/" + context.version + "/" + ecosystem + "/" + update["key"]
        marker = "<!-- jimu-release-automation:" + context.version + " dependency:" + ecosystem + ":" + update["key"] + " -->"
        update = dict(update, message=update["message"] + "\n\n" + marker)
        matching = [p for p in pulls if p.get("head", {}).get("ref") == branch or marker in (p.get("body") or "")]
        candidates = [p for p in matching if p.get("state") == "open"]
        require(len(candidates) <= 1, "multiple PRs claim the same managed dependency set")
        pull = candidates[0] if candidates else None
        previous = None
        head = github.ref(branch, optional=True)
        if pull:
            validate_pull(context, pull, branch, marker)
        else:
            require(update["kind"] == "create_pull_request", "update event has no managed open PR")
            if matching:
                previous = max(matching, key=lambda p: p["number"])
                validate_pull(context, previous, branch, marker, state="closed")
                require(previous.get("merged_at") or head is None or head != previous.get("head", {}).get("sha"),
                        "dependency PR was closed without merging; refusing to reopen it")
                if not previous.get("merged_at"):
                    # A replacement ref must pass orphan validation, including its
                    # App identity and ancestry, before recovering an interrupted PR.
                    previous = None
        require(not pull or head == pull.get("head", {}).get("sha"), "dependency PR head ref changed")
        desired = dict(base_tree)
        entries = []
        for path, content in update["changes"]:
            original = base_tree.get(path)
            require(original is None or (original["type"] == "blob" and original["mode"] == "100644"),
                    "base dependency path is not a regular file")
            require(original is not None or (ecosystem == "go" and path == "go.sum" and content is not None),
                    "dependency update must target an existing repository file: " + path)
            sha = git_blob_sha(content) if content is not None else None
            entry = {"path": path, "mode": "100644", "type": "blob", "sha": sha}
            entries.append((entry, content))
            if sha is None:
                desired.pop(path, None)
            else:
                desired[path] = entry
        require(not same_tree(base_tree, desired), "dependency event makes no change to the base tree")
        if head:
            head_commit, head_tree = github.tree(head)
            managed = pull or previous
            if managed:
                tree_markers = TREE_MARKER.findall(managed.get("body") or "")
                require(len(tree_markers) == 1, "managed dependency PR is missing its tree marker")
                if tree_markers[0] != head_commit["tree"]["sha"]:
                    # A non-forced ref update may succeed immediately before the
                    # PR body write is interrupted. Match its App dependency
                    # identity and file scope before advancing from a newer base.
                    commit, tree = head_commit, head_tree
                    visited = set()
                    while commit["tree"]["sha"] != tree_markers[0]:
                        require(commit["sha"] not in visited, "dependency recovery history contains a cycle")
                        visited.add(commit["sha"])
                        parents = validate_recovery_commit(context, commit, tree, update, marker, base)
                        require(len(parents) == 2, "interrupted dependency PR has unexpected parents")
                        commit, tree = github.tree(parents[0])
            else:
                # Recover a ref created before an interrupted PR API call, without
                # trusting a branch name as ownership of existing repository work.
                recover_orphan(context, head_commit, head_tree, update, marker, base)
            unchanged = same_tree(head_tree, desired)
        else:
            unchanged = False
        plans.append(dict(update, branch=branch, marker=marker, pull=pull, head=head,
                          unchanged=unchanged, entries=entries, base_tree=base_commit["tree"]["sha"]))
    return plans


def publish(context, plan, base):
    github = context.github
    context.guard(base)
    require(github.ref(plan["branch"], optional=True) == plan["head"], "dependency head ref changed while planning")
    if plan["pull"]:
        current = github.api("pulls/" + str(plan["pull"]["number"]))
        validate_pull(context, current, plan["branch"], plan["marker"])
        require(current.get("head", {}).get("sha") == plan["head"], "dependency PR head changed while planning")
    if plan["unchanged"]:
        sha = plan["head"]
        tree_sha = github.api("git/commits/" + sha)["tree"]["sha"]
    else:
        entries = []
        for entry, content in plan["entries"]:
            if content is not None:
                blob = github.api("git/blobs", "POST", {"encoding": "base64", "content": base64.b64encode(content).decode("ascii")})
                require(blob.get("sha") == entry["sha"], "GitHub blob SHA differs from prepared content")
            entries.append(entry)
        tree_sha = github.api("git/trees", "POST", {"base_tree": plan["base_tree"], "tree": entries})["sha"]
        parents = [plan["head"]] if plan["head"] else [base]
        if plan["head"] and base != plan["head"]:
            parents.append(base)
        commit = github.api("git/commits", "POST", {"message": plan["message"], "tree": tree_sha,
                           "parents": parents, "author": context.author(), "committer": context.author()})
        sha = commit["sha"]
        context.guard(base)
        require(github.ref(plan["branch"], optional=True) == plan["head"], "dependency head ref changed before update")
        if plan["head"]:
            github.api("git/refs/heads/" + plan["branch"], "PATCH", {"sha": sha, "force": False})
        else:
            github.api("git/refs", "POST", {"ref": "refs/heads/" + plan["branch"], "sha": sha})
    body = plan["body"].rstrip() + "\n\n" + plan["marker"] + "\n<!-- jimu-dependabot-tree:" + tree_sha + " -->\nTracks #" + context.issue + "\n"
    context.guard(base)
    require(github.ref(plan["branch"]) == sha, "dependency branch changed before PR publication")
    if plan["pull"]:
        pull = github.api("pulls/" + str(plan["pull"]["number"]))
        validate_pull(context, pull, plan["branch"], plan["marker"])
        require(pull.get("head", {}).get("sha") == sha, "dependency PR changed before publication")
        if pull.get("body") != body or pull.get("title") != plan["title"]:
            github.api("pulls/" + str(pull["number"]), "PATCH", {"title": plan["title"], "body": body})
        return not plan["unchanged"]
    pull = github.api("pulls", "POST", {"head": plan["branch"], "base": BASE_BRANCH,
                                         "title": plan["title"], "body": body})
    validate_pull(context, pull, plan["branch"], plan["marker"])
    return True


def apply(ecosystem, result_path, base):
    require(ecosystem in ECOSYSTEMS, "unsupported ecosystem")
    require(SHA.fullmatch(base), "invalid base SHA")
    updates = events_from(result_path, ecosystem, base)
    context = Context(publishing=True)
    context.guard(base)
    plans = plan_updates(context, ecosystem, updates, base) if updates else []
    published, unchanged = 0, 0
    for plan in plans:
        if publish(context, plan, base):
            published += 1
        else:
            unchanged += 1
    output("published", published)
    output("unchanged", unchanged)


def main(args=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    prepare_parser = commands.add_parser("prepare")
    prepare_parser.add_argument("config_path")
    prepare_parser.add_argument("input_path")
    apply_parser = commands.add_parser("apply")
    apply_parser.add_argument("ecosystem", choices=ECOSYSTEMS)
    apply_parser.add_argument("result_path")
    apply_parser.add_argument("base_sha")
    options = parser.parse_args(args)
    try:
        if options.command == "prepare":
            prepare(options.config_path, options.input_path)
        else:
            apply(options.ecosystem, options.result_path, options.base_sha)
    except (Error, ValueError, OSError, KeyError, TypeError) as error:
        # Never print subprocess stdin or raw CLI payloads; these may contain secrets.
        print("Dependabot publisher: " + str(error), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

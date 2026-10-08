#!/usr/bin/env python3
"""Exercise the publisher with the real CLI JSONL schema and a GitHub API double."""

import base64
import copy
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit


SCRIPT = Path(__file__).with_name("dependabot_cli.py")
if SCRIPT.exists():
    SPEC = importlib.util.spec_from_file_location("dependabot_cli", SCRIPT)
    CLI = importlib.util.module_from_spec(SPEC)
    SPEC.loader.exec_module(CLI)
else:
    CLI = None

BASE = "a" * 40
REPO = "wangkuntian/jimu"
SLUG = "jimu-release-automation"


def blob_sha(content):
    return hashlib.sha1(b"blob " + str(len(content)).encode() + b"\0" + content).hexdigest()


def dependency_file(name="go.mod", content="module example.test/app\n", directory="/", **extra):
    return dict(content=content, content_encoding="utf-8", deleted=False,
                directory=directory, name=name, operation="update", support_file=False,
                type="file", **extra)


def create_event(content="module example.test/app\nrequire example.test/lib v1.2.0\n"):
    return {"type": "create_pull_request", "data": {
        "base-commit-sha": BASE,
        "pr-title": "Bump example.test/lib from 1.1.0 to 1.2.0",
        "pr-body": "Updates example.test/lib.\n\nRelease notes: https://example.test/releases",
        "commit-message": "Bump example.test/lib from 1.1.0 to 1.2.0\n\nUpstream details",
        "dependencies": [{"name": "example.test/lib", "version": "1.2.0",
                          "previous-version": "1.1.0", "requirements": []}],
        "updated-dependency-files": [dependency_file(content=content)],
    }}


class GitHub:
    """Return REST response shapes, while retaining observable repository state."""

    def __init__(self):
        self.refs = {"dependabot-updates": BASE}
        self.blobs = {}
        self.trees = {}
        self.commits = {}
        self.pulls = []
        self.issue = {"number": 73, "title": "release: v0.3.5", "state": "open", "author_association": "OWNER",
                      "body": "Release scope", "labels": [{"name": "release: collecting"}]}
        self.comments = [{"id": 1, "user": {"login": SLUG + "[bot]"},
                          "body": "<!-- jimu-release-automation:v0.3.5 -->\nrelease base: `" + BASE + "`"}]
        entries = []
        for path, content in {"go.mod": b"module example.test/app\nrequire example.test/lib v1.1.0\n",
                              "go.sum": b"original sums\n", "README.md": b"user documentation\n",
                              ".github/workflows/ci.yml": b"name: CI\n",
                              "Dockerfile": b"FROM alpine:3.20\n",
                              "docker-compose.yml": b"services: {}\n"}.items():
            sha = blob_sha(content)
            self.blobs[sha] = content
            entries.append({"path": path, "mode": "100644", "type": "blob", "sha": sha})
        tree = self.store_tree(entries)
        self.commits[BASE] = {"sha": BASE, "tree": {"sha": tree, "url": "https://api.github.com/git/trees/" + tree}, "parents": [],
                              "message": "initial commit", "author": {"name": "User", "email": "user@example.test", "date": "2026-10-08T00:00:00Z"}}
        self.writes = []
        self.issue_reads = 0
        self.close_on_issue_read = None
        self.change_base_on_read = None
        self.base_reads = 0
        self.fail_pr_once = False
        self.fail_patch_pr_once = False
        self.current_author = SLUG + "[bot]"

    def store_tree(self, entries):
        ordered = sorted(copy.deepcopy(entries), key=lambda item: item["path"])
        sha = hashlib.sha1(json.dumps(ordered, sort_keys=True).encode()).hexdigest()
        self.trees[sha] = ordered
        return sha

    def content(self, branch, path):
        tree = self.trees[self.commits[self.refs[branch]]["tree"]["sha"]]
        entry = next(item for item in tree if item["path"] == path)
        return self.blobs[entry["sha"]]

    def run(self, args, *, input=None, text=False, capture_output=False, **kwargs):
        if args[:2] != ["gh", "api"]:
            raise AssertionError("only gh api is allowed")
        endpoint = args[2]
        if "--method" not in args:
            raise AssertionError("API method must be explicit")
        method = args[args.index("--method") + 1]
        query = parse_qs(urlsplit(endpoint).query)
        path = urlsplit(endpoint).path.removeprefix("repos/" + REPO + "/")
        payload = json.loads(input) if input else None
        if method != "GET":
            if "--input" not in args or args[args.index("--input") + 1] != "-":
                raise AssertionError("mutations must send JSON via stdin")
            self.writes.append((method, path, copy.deepcopy(payload)))

        try:
            result = self.request(method, path, query, payload)
            return subprocess.CompletedProcess(args, 0, json.dumps(result), "")
        except LookupError as error:
            return subprocess.CompletedProcess(args, 1, "", "gh: " + str(error) + " (HTTP 404)")

    def request(self, method, path, query, payload):
        if method == "GET" and path == "issues/73":
            self.issue_reads += 1
            if self.close_on_issue_read == self.issue_reads:
                self.issue["state"] = "closed"
            return self.issue
        if method == "GET" and path == "issues/73/comments":
            return self.comments if query.get("page", ["1"])[0] == "1" else []
        if method == "GET" and path.startswith("git/ref/heads/"):
            branch = path.removeprefix("git/ref/heads/")
            if branch == "dependabot-updates":
                self.base_reads += 1
                if self.change_base_on_read == self.base_reads:
                    self.refs[branch] = "f" * 40
            if branch not in self.refs:
                raise LookupError("reference not found")
            return {"ref": "refs/heads/" + branch, "object": {"type": "commit", "sha": self.refs[branch]}}
        if method == "GET" and path.startswith("git/commits/"):
            return self.commits[path.removeprefix("git/commits/")]
        if method == "GET" and path.startswith("git/trees/"):
            sha = path.removeprefix("git/trees/")
            return {"sha": sha, "tree": self.trees[sha], "truncated": False}
        if method == "GET" and path == "pulls":
            result = [p for p in self.pulls if (query.get("state", ["open"])[0] == "all" or p["state"] == query.get("state", ["open"])[0])
                      and ("base" not in query or p["base"]["ref"] == query["base"][0])]
            start = (int(query.get("page", ["1"])[0]) - 1) * 100
            return result[start:start + 100]
        if method == "GET" and path.startswith("pulls/"):
            return next(p for p in self.pulls if p["number"] == int(path.split("/")[1]))
        if method == "POST" and path == "git/blobs":
            content = base64.b64decode(payload["content"], validate=True)
            sha = blob_sha(content)
            self.blobs[sha] = content
            return {"sha": sha}
        if method == "POST" and path == "git/trees":
            entries = {e["path"]: copy.deepcopy(e) for e in self.trees[payload["base_tree"]]}
            for entry in payload["tree"]:
                if entry["sha"] is None:
                    entries.pop(entry["path"], None)
                else:
                    entries[entry["path"]] = entry
            return {"sha": self.store_tree(list(entries.values()))}
        if method == "POST" and path == "git/commits":
            sha = hashlib.sha1(json.dumps(payload, sort_keys=True).encode()).hexdigest()
            self.commits[sha] = {"sha": sha, "tree": {"sha": payload["tree"], "url": "https://api.github.com/git/trees/" + payload["tree"]},
                                 "message": payload["message"], "parents": [{"sha": s, "url": "https://api.github.com/git/commits/" + s, "html_url": "https://github.com/" + REPO + "/commit/" + s} for s in payload["parents"]],
                                 "author": dict(payload["author"], date="2026-10-08T00:00:00Z"), "committer": dict(payload.get("committer", payload["author"]), date="2026-10-08T00:00:00Z")}
            return self.commits[sha]
        if method == "POST" and path == "git/refs":
            branch = payload["ref"].removeprefix("refs/heads/")
            if branch in self.refs:
                raise AssertionError("ref overwrite")
            self.refs[branch] = payload["sha"]
            return {"ref": payload["ref"], "object": {"sha": payload["sha"]}}
        if method == "PATCH" and path.startswith("git/refs/heads/"):
            branch = path.removeprefix("git/refs/heads/")
            if payload.get("force") is not False:
                raise AssertionError("force overwrite")
            if self.refs[branch] not in [p["sha"] for p in self.commits[payload["sha"]]["parents"]]:
                raise AssertionError("non-fast-forward ref update")
            self.refs[branch] = payload["sha"]
            for pull in self.pulls:
                if pull["head"]["ref"] == branch:
                    pull["head"]["sha"] = payload["sha"]
            return {"object": {"sha": payload["sha"]}}
        if method == "POST" and path == "pulls":
            if self.fail_pr_once:
                self.fail_pr_once = False
                raise LookupError("simulated API interruption")
            number = len(self.pulls) + 1
            pull = {"number": number, "state": "open", "title": payload["title"], "body": payload["body"],
                    "user": {"login": self.current_author, "type": "Bot"},
                    "head": {"ref": payload["head"], "sha": self.refs[payload["head"]], "repo": {"full_name": REPO}},
                    "base": {"ref": payload["base"], "repo": {"full_name": REPO}},
                    "html_url": "https://github.com/" + REPO + "/pull/" + str(number)}
            self.pulls.append(pull)
            return pull
        if method == "PATCH" and path.startswith("pulls/"):
            if self.fail_patch_pr_once:
                self.fail_patch_pr_once = False
                raise LookupError("simulated API interruption")
            pull = next(p for p in self.pulls if p["number"] == int(path.split("/")[1]))
            pull.update(payload)
            return pull
        raise AssertionError("unhandled API request: " + method + " " + path)


class PublisherTests(unittest.TestCase):
    def setUp(self):
        self.assertIsNotNone(CLI, "Dependabot publisher has not been implemented")
        self.github = GitHub()
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = {"GH_REPO": REPO, "RELEASE_VERSION": "v0.3.5", "RELEASE_ISSUE_NUMBER": "73",
                    "RELEASE_APP_SLUG": SLUG}

    def invoke(self, args):
        with patch.dict(os.environ, self.env, clear=True), patch.object(CLI.subprocess, "run", self.github.run), \
                patch("sys.stdout", new_callable=io.StringIO) as output, patch("sys.stderr", new_callable=io.StringIO) as errors:
            code = CLI.main(args)
            return code, output.getvalue(), errors.getvalue()

    def apply(self, events=None, ecosystem="go", base=BASE):
        result = self.root / "result.jsonl"
        events = [create_event()] if events is None else copy.deepcopy(events)
        if not any(e["type"] == "mark_as_processed" for e in events):
            events.append({"type": "mark_as_processed", "data": {"base-commit-sha": base}})
        result.write_text("".join(json.dumps(e) + "\n" for e in events))
        return self.invoke(["apply", ecosystem, str(result), base])

    def test_prepare_pins_source_and_read_only_credential_without_app_token(self):
        del self.env["RELEASE_APP_SLUG"]
        config, output = self.root / "job.yml", self.root / "input.yml"
        config.write_text(json.dumps({"job": {"package-manager": "go_modules", "source": {"provider": "github", "directory": "/", "repo": "wrong/repo"}}, "credentials": [{"password": "do-not-retain"}]}))
        self.assertEqual(0, self.invoke(["prepare", str(config), str(output)])[0])
        job = json.loads(output.read_text())
        self.assertEqual({"provider": "github", "directory": "/", "repo": REPO,
                          "branch": "dependabot-updates", "commit": BASE}, job["job"]["source"])
        self.assertEqual([{"type": "git_source", "host": "github.com", "username": "x-access-token",
                           "password": "$LOCAL_GITHUB_ACCESS_TOKEN"}], job["credentials"])
        self.assertEqual([], self.github.writes)

    def test_create_pr_preserves_bytes_and_conventional_commit(self):
        code, out, errors = self.apply()
        self.assertEqual(0, code, errors)
        self.assertIn("published=1", out)
        pull = self.github.pulls[0]
        self.assertEqual("dependabot-updates", pull["base"]["ref"])
        self.assertEqual(SLUG + "[bot]", pull["user"]["login"])
        self.assertIn("<!-- jimu-release-automation:v0.3.5 dependency:go:", pull["body"])
        self.assertIn("Tracks #73", pull["body"])
        self.assertEqual(create_event()["data"]["updated-dependency-files"][0]["content"].encode(), self.github.content(pull["head"]["ref"], "go.mod"))
        commit = self.github.commits[pull["head"]["sha"]]
        self.assertEqual("chore(deps): bump example.test/lib from 1.1.0 to 1.2.0", commit["message"])
        self.assertEqual([BASE], [p["sha"] for p in commit["parents"]])
        self.assertEqual(b"user documentation\n", self.github.content(pull["head"]["ref"], "README.md"))

    def test_identical_retry_has_no_duplicate_commit_or_pr(self):
        self.assertEqual(0, self.apply()[0])
        old_head = self.github.pulls[0]["head"]["sha"]
        commits = len(self.github.commits)
        code, out, errors = self.apply()
        self.assertEqual(0, code, errors)
        self.assertIn("unchanged=1", out)
        self.assertEqual(1, len(self.github.pulls))
        self.assertEqual(commits, len(self.github.commits))
        self.assertEqual(old_head, self.github.pulls[0]["head"]["sha"])

    def test_new_version_reuses_pr_and_advances_ref_without_force(self):
        self.assertEqual(0, self.apply()[0])
        old = self.github.pulls[0]["head"]["sha"]
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        event["data"]["pr-title"] = "Bump example.test/lib from 1.1.0 to 1.3.0"
        event["data"]["dependencies"][0]["version"] = "1.3.0"
        code, _, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        self.assertEqual(1, len(self.github.pulls))
        pull = self.github.pulls[0]
        self.assertNotEqual(old, pull["head"]["sha"])
        self.assertIn(old, [p["sha"] for p in self.github.commits[pull["head"]["sha"]]["parents"]])
        self.assertEqual(event["data"]["updated-dependency-files"][0]["content"].encode(), self.github.content(pull["head"]["ref"], "go.mod"))

    def test_unmanaged_author_marker_or_branch_is_never_overwritten(self):
        for field in ("author", "marker", "head", "base", "repo"):
            with self.subTest(field=field):
                self.github = GitHub()
                self.assertEqual(0, self.apply()[0])
                pull = self.github.pulls[0]
                if field == "author":
                    pull["user"]["login"] = "human"
                elif field == "marker":
                    pull["body"] = "User PR"
                elif field == "head":
                    pull["head"]["ref"] = "feature/user"
                elif field == "base":
                    pull["base"]["ref"] = "master"
                else:
                    pull["head"]["repo"]["full_name"] = "attacker/jimu"
                old_refs = copy.deepcopy(self.github.refs)
                old_count = len(self.github.commits)
                self.assertNotEqual(0, self.apply()[0])
                self.assertEqual(old_refs, self.github.refs)
                self.assertEqual(old_count, len(self.github.commits))

    def test_user_commit_tree_change_is_rejected(self):
        self.assertEqual(0, self.apply()[0])
        pull = self.github.pulls[0]
        parent = pull["head"]["sha"]
        tree = self.github.trees[self.github.commits[parent]["tree"]["sha"]]
        content = b"Human edits\n"
        sha = blob_sha(content)
        self.github.blobs[sha] = content
        tree = [dict(e, sha=sha) if e["path"] == "README.md" else e for e in tree]
        head = "c" * 40
        self.github.commits[head] = {"sha": head, "tree": {"sha": self.github.store_tree(tree)},
                                     "parents": [{"sha": parent}], "author": {"name": "Human", "email": "user@example.test"}}
        self.github.refs[pull["head"]["ref"]] = head
        pull["head"]["sha"] = head
        self.assertNotEqual(0, self.apply()[0])
        self.assertEqual(head, self.github.refs[pull["head"]["ref"]])

    def test_orphan_branch_from_pr_api_failure_can_be_recovered(self):
        self.github.fail_pr_once = True
        self.assertNotEqual(0, self.apply()[0])
        count = len(self.github.commits)
        code, _, errors = self.apply()
        self.assertEqual(0, code, errors)
        self.assertEqual(count, len(self.github.commits))
        self.assertEqual(1, len(self.github.pulls))

    def test_new_batch_reuses_merged_dependency_branch_safely(self):
        self.assertEqual(0, self.apply()[0])
        prior = self.github.pulls[0]
        old_head = prior["head"]["sha"]
        prior.update(state="closed", merged_at="2026-10-08T09:00:00Z")
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        event["data"]["pr-title"] = "Bump example.test/lib from 1.1.0 to 1.3.0"
        code, _, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        self.assertEqual(2, len(self.github.pulls))
        self.assertEqual(prior["head"]["ref"], self.github.pulls[1]["head"]["ref"])
        self.assertIn(old_head, [p["sha"] for p in self.github.commits[self.github.pulls[1]["head"]["sha"]]["parents"]])

    def test_new_batch_after_base_advances_preserves_other_merged_files(self):
        self.assertEqual(0, self.apply()[0])
        prior = self.github.pulls[0]
        prior.update(state="closed", merged_at="2026-10-08T09:00:00Z")
        old_head = prior["head"]["sha"]
        old_tree = self.github.trees[self.github.commits[old_head]["tree"]["sha"]]
        content = b"FROM alpine:3.21\n"
        sha = blob_sha(content)
        self.github.blobs[sha] = content
        entries = [dict(e, sha=sha) if e["path"] == "Dockerfile" else e for e in old_tree]
        new_base = "b" * 40
        self.github.commits[new_base] = {"sha": new_base, "tree": {"sha": self.github.store_tree(entries)},
                                       "parents": [{"sha": BASE}], "author": {"name": "User", "email": "user@example.test"}}
        self.github.refs["dependabot-updates"] = new_base
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        event["data"]["base-commit-sha"] = new_base
        event["data"]["pr-title"] = "Bump example.test/lib from 1.2.0 to 1.3.0"
        code, _, errors = self.apply([event], base=new_base)
        self.assertEqual(0, code, errors)
        current = self.github.pulls[1]
        self.assertEqual(content, self.github.content(current["head"]["ref"], "Dockerfile"))
        self.assertEqual([old_head, new_base], [p["sha"] for p in self.github.commits[current["head"]["sha"]]["parents"]])

    def test_grouped_dependency_order_is_stable_across_retries(self):
        event = create_event()
        event["data"]["dependencies"].append({"name": "example.test/second", "version": "2.0.0", "previous-version": "1.0.0", "requirements": []})
        event["data"]["pr-title"] = "Bump the library group with 2 updates"
        event["data"]["dependency-group"] = {"name": "libraries", "rules": {"patterns": ["example.test/*"]}}
        self.assertEqual(0, self.apply([event])[0])
        branch = self.github.pulls[0]["head"]["ref"]
        count = len(self.github.commits)
        event["data"]["dependencies"].reverse()
        self.assertEqual(0, self.apply([event])[0])
        self.assertEqual(branch, self.github.pulls[0]["head"]["ref"])
        self.assertEqual(count, len(self.github.commits))

    def test_wrong_token_author_fails_publication(self):
        self.github.current_author = "human"
        code, _, _ = self.apply()
        self.assertNotEqual(0, code)
        self.assertEqual("human", self.github.pulls[0]["user"]["login"])

    def test_retry_recovers_updated_ref_before_pr_body_was_saved(self):
        self.assertEqual(0, self.apply()[0])
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        event["data"]["pr-title"] = "Bump example.test/lib from 1.1.0 to 1.3.0"
        self.github.fail_patch_pr_once = True
        self.assertNotEqual(0, self.apply([event])[0])
        count = len(self.github.commits)
        head = self.github.pulls[0]["head"]["sha"]
        code, out, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        self.assertIn("unchanged=1", out)
        self.assertEqual(count, len(self.github.commits))
        self.assertEqual(head, self.github.pulls[0]["head"]["sha"])
        self.assertEqual(event["data"]["pr-title"], self.github.pulls[0]["title"])

    def test_merged_branch_retry_recovers_new_pr_creation_interruption(self):
        self.assertEqual(0, self.apply()[0])
        self.github.pulls[0].update(state="closed", merged_at="2026-10-08T09:00:00Z")
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        event["data"]["pr-title"] = "Bump example.test/lib from 1.1.0 to 1.3.0"
        self.github.fail_pr_once = True
        self.assertNotEqual(0, self.apply([event])[0])
        count = len(self.github.commits)
        code, _, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        self.assertEqual(2, len(self.github.pulls))
        self.assertEqual(count, len(self.github.commits))

    def test_user_change_cannot_pose_as_interrupted_publisher(self):
        self.assertEqual(0, self.apply()[0])
        pull = self.github.pulls[0]
        old = pull["head"]["sha"]
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        self.github.fail_patch_pr_once = True
        self.assertNotEqual(0, self.apply([event])[0])
        head = pull["head"]["sha"]
        self.github.commits[head]["committer"]["name"] = "Human"
        self.assertNotEqual(0, self.apply([event])[0])
        self.assertEqual(head, self.github.refs[pull["head"]["ref"]])
        self.assertNotEqual(old, head)

    def test_owner_and_app_comment_are_required_before_publishing(self):
        for invalid in ("non_owner", "human_comment", "body_only"):
            with self.subTest(invalid=invalid):
                self.github = GitHub()
                if invalid == "non_owner":
                    self.github.issue["author_association"] = "CONTRIBUTOR"
                elif invalid == "human_comment":
                    self.github.comments[0]["user"]["login"] = "human"
                else:
                    self.github.comments = []
                    self.github.issue["body"] = "<!-- jimu-release-automation:v0.3.5 -->"
                self.assertNotEqual(0, self.apply()[0])
                self.assertEqual([], self.github.writes)

    def test_pr_pagination_finds_managed_pr_beyond_first_page(self):
        self.assertEqual(0, self.apply()[0])
        managed = self.github.pulls[0]
        fillers = []
        for n in range(100):
            filler = copy.deepcopy(managed)
            filler.update(number=n + 100, body="Unrelated PR")
            filler["head"]["ref"] = "feature/pr-" + str(n)
            fillers.append(filler)
        self.github.pulls = fillers + [managed]
        count = len(self.github.commits)
        code, _, errors = self.apply()
        self.assertEqual(0, code, errors)
        self.assertEqual(count, len(self.github.commits))
        self.assertEqual(101, len(self.github.pulls))

    def test_api_ref_failure_is_not_treated_as_missing_branch(self):
        original = self.github.run
        def run(args, **kwargs):
            if "/git/ref/heads/dependabot-cli/" in args[2]:
                return subprocess.CompletedProcess(args, 1, "", "gh: rate limited (HTTP 403)")
            return original(args, **kwargs)
        result = self.root / "result.jsonl"
        result.write_text(json.dumps(create_event()) + "\n" + json.dumps({"type": "mark_as_processed", "data": {"base-commit-sha": BASE}}))
        with patch.dict(os.environ, self.env, clear=True), patch.object(CLI.subprocess, "run", run), \
                patch("sys.stderr", new_callable=io.StringIO):
            self.assertNotEqual(0, CLI.main(["apply", "go", str(result), BASE]))
        self.assertEqual([], self.github.writes)

    def test_user_head_race_is_refused_without_force(self):
        original = self.github.run
        def run(args, **kwargs):
            result = original(args, **kwargs)
            if args[2].endswith("/git/commits") and args[args.index("--method") + 1] == "POST":
                branch = self.github.pulls[0]["head"]["ref"]
                self.github.refs[branch] = "d" * 40
                self.github.pulls[0]["head"]["sha"] = "d" * 40
            return result
        self.assertEqual(0, self.apply()[0])
        event = create_event("module example.test/app\nrequire example.test/lib v1.3.0\n")
        result = self.root / "result.jsonl"
        result.write_text(json.dumps(event) + "\n" + json.dumps({"type": "mark_as_processed", "data": {"base-commit-sha": BASE}}))
        count = len(self.github.writes)
        with patch.dict(os.environ, self.env, clear=True), patch.object(CLI.subprocess, "run", run), \
                patch("sys.stderr", new_callable=io.StringIO):
            self.assertNotEqual(0, CLI.main(["apply", "go", str(result), BASE]))
        self.assertEqual("d" * 40, self.github.refs[self.github.pulls[0]["head"]["ref"]])
        self.assertFalse(any(method == "PATCH" for method, _, _ in self.github.writes[count:]))

    def test_manually_closed_dependency_pr_does_not_reopen(self):
        self.assertEqual(0, self.apply()[0])
        self.github.pulls[0].update(state="closed", merged_at=None)
        count = len(self.github.writes)
        self.assertNotEqual(0, self.apply()[0])
        self.assertEqual(count, len(self.github.writes))

    def test_empty_jsonl_and_invalid_later_file_block_entire_batch(self):
        invalid = create_event()
        invalid["data"]["dependencies"][0]["name"] = "other.test/lib"
        invalid["data"]["updated-dependency-files"][0]["name"] = "README.md"
        self.assertNotEqual(0, self.apply([create_event(), invalid])[0])
        self.assertEqual([], self.github.writes)

    def test_deleted_dependency_file_uses_tree_deletion(self):
        event = create_event()
        event["data"]["updated-dependency-files"].append(dict(dependency_file(name="go.sum"), deleted=True, operation="delete"))
        code, _, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        head = self.github.pulls[0]["head"]["sha"]
        paths = [e["path"] for e in self.github.trees[self.github.commits[head]["tree"]["sha"]]]
        self.assertNotIn("go.sum", paths)

    def test_truncated_tree_is_not_used_for_publishing(self):
        original = self.github.run
        def run(args, **kwargs):
            result = original(args, **kwargs)
            if "/git/trees/" in args[2]:
                data = json.loads(result.stdout)
                data["truncated"] = True
                result.stdout = json.dumps(data)
            return result
        with patch.dict(os.environ, self.env, clear=True), patch.object(CLI.subprocess, "run", run):
            result = self.root / "result.jsonl"
            result.write_text(json.dumps(create_event()) + "\n" + json.dumps({"type": "mark_as_processed", "data": {"base-commit-sha": BASE}}))
            with patch("sys.stderr", new_callable=io.StringIO):
                self.assertNotEqual(0, CLI.main(["apply", "go", str(result), BASE]))
        self.assertEqual([], self.github.writes)

    def test_closed_issue_and_terminal_labels_are_rejected_without_writes(self):
        for state, labels in [("closed", ["release: collecting"]), ("open", ["release: published"]),
                              ("open", ["release: blocked", "release: collecting"]), ("open", [])]:
            with self.subTest(state=state, labels=labels):
                self.github = GitHub()
                self.github.issue["state"] = state
                self.github.issue["labels"] = [{"name": n} for n in labels]
                self.assertNotEqual(0, self.apply()[0])
                self.assertEqual([], self.github.writes)

    def test_candidate_issue_remains_eligible(self):
        self.github.issue["labels"] = [{"name": "release: candidate"}]
        self.assertEqual(0, self.apply()[0])

    def test_missing_issue_marker_wrong_title_and_non_issue_are_rejected(self):
        for invalid in ("marker", "title", "pull_request"):
            with self.subTest(invalid=invalid):
                self.github = GitHub()
                if invalid == "marker":
                    self.github.comments = []
                elif invalid == "title":
                    self.github.issue["title"] = "release: v0.3.6"
                else:
                    self.github.issue["pull_request"] = {"url": "https://api.github.com/repos/x/y/pulls/73"}
                self.assertNotEqual(0, self.apply()[0])
                self.assertEqual([], self.github.writes)

    def test_issue_closing_after_planning_blocks_pr_creation(self):
        self.github.close_on_issue_read = 2
        self.assertNotEqual(0, self.apply()[0])
        self.assertEqual([], self.github.writes)

    def test_stale_expected_or_event_base_blocks_all_writes(self):
        self.assertNotEqual(0, self.apply(base="b" * 40)[0])
        self.assertEqual([], self.github.writes)
        self.github.change_base_on_read = 2
        self.github.base_reads = 0
        self.assertNotEqual(0, self.apply()[0])
        self.assertEqual([], self.github.writes)

    def test_failure_after_valid_create_event_blocks_entire_batch(self):
        for kind in ("record_update_job_error", "record_update_job_unknown_error"):
            with self.subTest(kind=kind):
                self.github = GitHub()
                failure = {"type": kind, "data": {"error-type": "dependency_file_not_found", "error-details": {}}}
                self.assertNotEqual(0, self.apply([create_event(), failure])[0])
                self.assertEqual([], self.github.writes)

    def test_unknown_mutation_and_close_events_fail_without_publishing_partial_batch(self):
        for kind in ("refresh_pull_request", "close_pull_request"):
            with self.subTest(kind=kind):
                self.github = GitHub()
                self.assertNotEqual(0, self.apply([create_event(), {"type": kind, "data": {}}])[0])
                self.assertEqual([], self.github.writes)

    def test_paths_and_ecosystem_mismatch_rejected_before_any_write(self):
        files = [dependency_file(name="../go.mod"), dependency_file(name="/go.mod"),
                 dependency_file(name="go.mod", directory="/../"), dependency_file(name="go.mod", directory="//tmp"),
                 dependency_file(name="config", directory="/.git"), dependency_file(name="ci.yml", directory="/.github/workflows"),
                 dependency_file(name="README.md"), dependency_file(name="go.mod", symlink_target="README.md")]
        for file in files:
            with self.subTest(file=file):
                self.github = GitHub()
                event = create_event()
                event["data"]["updated-dependency-files"] = [file]
                self.assertNotEqual(0, self.apply([event])[0])
                self.assertEqual([], self.github.writes)

    def test_actions_and_docker_only_allow_supported_dependency_paths(self):
        for ecosystem, file, allowed in [
                ("github-actions", dependency_file(name="ci.yml", directory="/.github/workflows", content="name: updated\n"), True),
                ("docker", dependency_file(name="Dockerfile", content="FROM alpine:3.21\n"), True),
                ("docker", dependency_file(name="docker-compose.yml", content="services:\n  db:\n    image: mysql:9\n"), True),
                ("github-actions", dependency_file(name="action.yml", directory="/.github"), False),
                ("docker", dependency_file(name="ci.yml", directory="/.github/workflows"), False),
        ]:
            with self.subTest(ecosystem=ecosystem, file=file):
                self.github = GitHub()
                event = create_event()
                event["data"]["updated-dependency-files"] = [file]
                self.assertEqual(allowed, self.apply([event], ecosystem=ecosystem)[0] == 0)

    def test_base64_content_keeps_original_bytes_without_added_newline(self):
        event = create_event()
        file = event["data"]["updated-dependency-files"][0]
        file["content_encoding"] = "base64"
        file["content"] = "bW9kdWxlIGV4YW1wbGUudGVzdC9hcHA="
        code, _, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        self.assertEqual(b"module example.test/app", self.github.content(self.github.pulls[0]["head"]["ref"], "go.mod"))

    def test_invalid_base64_and_unknown_encoding_fail_without_writes(self):
        for encoding, content in [("base64", "***"), ("sha256", "a" * 64)]:
            with self.subTest(encoding=encoding):
                event = create_event()
                file = event["data"]["updated-dependency-files"][0]
                file.update(content_encoding=encoding, content=content)
                self.assertNotEqual(0, self.apply([event])[0])
                self.assertEqual([], self.github.writes)

    def test_update_event_refreshes_existing_managed_pr_only(self):
        event = create_event("module example.test/app\nrequire example.test/lib v1.4.0\n")
        event["type"] = "update_pull_request"
        event["data"]["dependency-names"] = ["example.test/lib"]
        del event["data"]["dependencies"]
        self.assertNotEqual(0, self.apply([event])[0])
        self.assertEqual([], self.github.writes)
        self.assertEqual(0, self.apply()[0])
        code, _, errors = self.apply([event])
        self.assertEqual(0, code, errors)
        self.assertEqual(1, len(self.github.pulls))

    def test_no_update_events_reports_zero_and_no_writes(self):
        events = [{"type": "update_dependency_list", "data": {"dependencies": [], "dependency_files": ["/go.mod"]}},
                  {"type": "increment_metric", "data": {"metric": "updater.started", "tags": {}}}]
        code, out, errors = self.apply(events)
        self.assertEqual(0, code, errors)
        self.assertIn("published=0", out)
        self.assertEqual([], self.github.writes)

    def test_truncated_jsonl_without_completion_is_rejected(self):
        result = self.root / "result.jsonl"
        result.write_text(json.dumps(create_event()) + "\n")
        self.assertNotEqual(0, self.invoke(["apply", "go", str(result), BASE])[0])
        self.assertEqual([], self.github.writes)

    def test_raw_commit_message_cannot_escape_conventional_format(self):
        event = create_event()
        event["data"]["commit-message"] = "feat(admin): dangerous\n\nrun $(whoami)"
        event["data"]["pr-title"] = "Bump example.test/lib from 1.1.0 to 1.2.0\n\nMalicious body"
        self.assertNotEqual(0, self.apply([event])[0])
        self.assertEqual([], self.github.writes)


if __name__ == "__main__":
    unittest.main()

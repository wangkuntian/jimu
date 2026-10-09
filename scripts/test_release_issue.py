#!/usr/bin/env python3
"""Exercise release Issue commands against a local GitHub REST fixture."""

import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("release_issue.py")
REPO = "owner/jimu"
VERSION = "v0.3.5"
START = "2026-10-07T12:00:00Z"
RELEASE_URL = "https://github.com/owner/jimu/releases/tag/v0.3.5"

FAKE_GH = r'''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
from urllib.parse import parse_qs, unquote, urlsplit
fixture = Path(os.environ["RELEASE_ISSUE_FIXTURE"])
data = json.loads(fixture.read_text())
args = sys.argv[1:]
assert args[0] == "api", args
endpoint = args[1]
assert "--method" in args, "HTTP method must be explicit"
method = args[args.index("--method") + 1]
payload = None
if "--input" in args:
    assert args[args.index("--input") + 1] == "-"
    payload = json.load(sys.stdin)
data["calls"].append({"method": method, "endpoint": endpoint, "payload": payload})
fixture.write_text(json.dumps(data))
failure = data.get("errors", {}).get(method + " " + endpoint)
if method == "PATCH" and data.get("fail_patch_field") in (payload or {}):
    failure = "Forbidden (HTTP 403)"
if failure:
    print("gh: " + failure, file=sys.stderr)
    sys.exit(1)
parsed = urlsplit(endpoint)
path = parsed.path.removeprefix("repos/owner/jimu/")
query = parse_qs(parsed.query)
page = int(query.get("page", ["1"])[0])
size = int(query.get("per_page", ["100"])[0])
if method == "GET" and path == "issues":
    result = [i for i in data["issues"] if query.get("state", ["all"])[0] == "all" or i["state"] == query["state"][0]]
    result = result[(page - 1) * size:page * size]
elif method == "GET" and path.endswith("/comments"):
    number = path.split("/")[1]
    result = data["comments"].get(number, [])[(page - 1) * size:page * size]
elif method == "GET" and path == "pulls":
    mutation = data.get("during_pull_read")
    if mutation and page == mutation["page"]:
        target = next(i for i in data["issues"] if i["number"] == mutation["number"])
        target.update(mutation["changes"])
        del data["during_pull_read"]
        fixture.write_text(json.dumps(data))
    result = data["pulls"][(page - 1) * size:page * size]
elif method == "GET" and path.startswith("issues/"):
    number = int(path.split("/")[1])
    result = next(i for i in data["issues"] if i["number"] == number)
elif method == "GET" and path.startswith("releases/tags/"):
    result = data.get("release")
    if result is None:
        print("gh: Not Found (HTTP 404)", file=sys.stderr)
        sys.exit(1)
elif method == "PATCH" and path.startswith("issues/"):
    number = int(path.split("/")[1])
    result = next(i for i in data["issues"] if i["number"] == number)
    result.update(payload)
    if "labels" in payload:
        result["labels"] = [{"name": label} for label in payload["labels"]]
    fixture.write_text(json.dumps(data))
elif method == "POST" and path.endswith("/labels"):
    number = int(path.split("/")[1])
    target = next(i for i in data["issues"] if i["number"] == number)
    current = {label["name"] for label in target["labels"]}
    target["labels"] += [{"name": label} for label in payload["labels"] if label not in current]
    result = target["labels"]
    fixture.write_text(json.dumps(data))
elif method == "DELETE" and "/labels/" in path:
    number = int(path.split("/")[1])
    label = unquote(path.split("/labels/", 1)[1])
    target = next(i for i in data["issues"] if i["number"] == number)
    target["labels"] = [item for item in target["labels"] if item["name"] != label]
    result = target["labels"]
    fixture.write_text(json.dumps(data))
else:
    print("unexpected request " + method + " " + endpoint, file=sys.stderr)
    sys.exit(1)
print(json.dumps(result))
'''


def issue(number=73, version=VERSION, state="open", labels=None, owner="OWNER"):
    return {
        "number": number,
        "title": "release: " + version,
        "state": state,
        "author_association": owner,
        "labels": [{"name": label} for label in (
            labels if labels is not None else ["release: collecting", "documentation"]
        )],
        "body": "## 范围\n\n保留用户的计划与 [链接](https://example.com).\n",
        "created_at": "2026-10-06T00:00:00Z",
        "html_url": f"https://github.com/{REPO}/issues/{number}",
    }


def bootstrap(version=VERSION, created_at=START):
    return {
        "body": f"<!-- jimu-release-automation:{version} -->\n- release base: `{'a' * 40}`\n",
        "created_at": created_at,
        "user": {"login": "release-app[bot]", "type": "Bot"},
    }


def pull(number, head, base, *, title=None, author="owner", body="", state="open", merged=False, created_at="2026-10-08T00:00:00Z"):
    return {
        "number": number,
        "title": title or f"PR {number}",
        "html_url": f"https://github.com/{REPO}/pull/{number}",
        "url": f"https://api.github.com/repos/{REPO}/pulls/{number}",
        "head": {"ref": head, "repo": {"full_name": REPO}},
        "base": {"ref": base, "repo": {"full_name": REPO}},
        "user": {"login": author, "type": "Bot" if author.endswith("[bot]") else "User"},
        "body": body,
        "state": state,
        "merged_at": "2026-10-08T02:00:00Z" if merged else None,
        "created_at": created_at,
    }


class ReleaseIssueTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.fixture = self.root / "github.json"
        gh = self.root / "gh"
        gh.write_text(FAKE_GH)
        gh.chmod(0o755)
        self.output = self.root / "output"
        self.env = dict(os.environ, GH_REPO=REPO, RELEASE_APP_SLUG="release-app",
                        RELEASE_ISSUE_FIXTURE=str(self.fixture), GITHUB_OUTPUT=str(self.output),
                        PATH=str(self.root) + os.pathsep + os.environ["PATH"])
        self.write(issues=[issue()], comments={"73": [bootstrap()]})

    def write(self, **values):
        data = {"issues": [], "comments": {}, "pulls": [], "calls": []}
        data.update(values)
        self.fixture.write_text(json.dumps(data))

    def read(self):
        return json.loads(self.fixture.read_text())

    def run_command(self, *args, success=True):
        result = subprocess.run([sys.executable, str(SCRIPT), *args], env=self.env,
                                capture_output=True, text=True)
        if success:
            self.assertEqual(result.returncode, 0, result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout)
        return result

    def patches(self):
        return [call for call in self.read()["calls"] if call["method"] == "PATCH"]

    def writes(self):
        return [call for call in self.read()["calls"] if call["method"] != "GET"]

    def test_active_outputs_only_the_open_managed_owner_cycle(self):
        outsider = issue(80, "v0.3.6", owner="CONTRIBUTOR")
        pr_as_issue = issue(81, "v0.3.7")
        pr_as_issue["pull_request"] = {"url": "https://api.github.com/pulls/81"}
        self.write(issues=[issue(), outsider, pr_as_issue], comments={"73": [bootstrap()]})
        result = self.run_command("active")
        self.assertIn("active=true\nissue_number=73\nversion=v0.3.5\n", result.stdout)
        self.assertEqual(self.output.read_text(), result.stdout)
        self.assertEqual(self.patches(), [])

    def test_closed_blocked_published_unmanaged_and_wrong_marker_are_not_active(self):
        for candidate, comments in [
            (issue(state="closed"), [bootstrap()]),
            (issue(labels=["release: collecting", "release: blocked"]), [bootstrap()]),
            (issue(labels=["release: candidate", "release: published"]), [bootstrap()]),
            (issue(), []),
            (issue(), [bootstrap("v0.3.4")]),
            (issue(labels=["documentation"]), [bootstrap()]),
            (issue(version="v00.3.5"), [bootstrap("v00.3.5")]),
        ]:
            with self.subTest(candidate=candidate, comments=comments):
                self.write(issues=[candidate], comments={"73": comments})
                result = self.run_command("active")
                self.assertEqual(result.stdout, "active=false\n")

    def test_multiple_managed_open_cycles_are_rejected_without_write(self):
        self.write(issues=[issue(), issue(74, "v0.3.6")],
                   comments={"73": [bootstrap()], "74": [bootstrap("v0.3.6")]})
        self.run_command("active", success=False)
        self.assertEqual(self.patches(), [])

    def test_candidate_cycle_remains_active_and_closed_cycle_can_be_synced(self):
        self.write(issues=[issue(labels=["release: candidate"])], comments={"73": [bootstrap()]})
        self.assertIn("active=true", self.run_command("active").stdout)
        self.write(issues=[issue(state="closed")], comments={"73": [bootstrap()]})
        self.run_command("sync", VERSION)
        self.assertEqual(self.read()["issues"][0]["state"], "closed")
        self.assertIn("Release automation", self.read()["issues"][0]["body"])

    def test_duplicate_or_incomplete_status_markers_do_not_replace_user_body(self):
        for body in [
            "notes\n<!-- jimu-release-status:start -->\nmissing end",
            "<!-- jimu-release-status:end -->\n<!-- jimu-release-status:start -->",
            "<!-- jimu-release-status:start --><!-- jimu-release-status:end -->\n<!-- jimu-release-status:start --><!-- jimu-release-status:end -->",
        ]:
            with self.subTest(body=body):
                target = issue()
                target["body"] = body
                self.write(issues=[target], comments={"73": [bootstrap()]})
                self.run_command("sync", VERSION, success=False)
                self.assertEqual(self.patches(), [])
                self.assertEqual(self.read()["issues"][0]["body"], body)

    def test_sync_preserves_user_body_and_is_idempotent(self):
        original = issue()
        original["body"] += "\n<!-- jimu-release-status:start -->\nold status\n<!-- jimu-release-status:end -->\n\n手写附录\n"
        self.write(issues=[original], comments={"73": [bootstrap()]})
        self.run_command("sync", VERSION)
        updated = self.read()["issues"][0]
        self.assertTrue(updated["body"].startswith("## 范围\n\n保留用户的计划与 [链接](https://example.com).\n\n"))
        self.assertTrue(updated["body"].endswith("\n\n手写附录\n"))
        self.assertEqual(updated["title"], "release: v0.3.5")
        self.assertNotIn("old status", updated["body"])
        self.assertIn("release/v0.3.5", updated["body"])
        self.assertIn("dependabot-updates", updated["body"])
        first_writes = len(self.patches())
        self.run_command("sync", VERSION)
        self.assertEqual(len(self.patches()), first_writes)

    def test_sync_collects_pr_categories_and_statuses_without_old_or_unmanaged_dependencies(self):
        pulls = [
            pull(90, "feature/widget", "release/v0.3.5", title="feat: widget", merged=True, state="closed"),
            pull(91, "fix/crash", "release/v0.3.5", title="fix: crash", state="closed"),
            pull(92, "dependabot-updates", "release/v0.3.5", body="<!-- jimu-release-automation:v0.3.5 aggregate -->"),
            pull(93, "release/v0.3.5", "master", author="release-app[bot]", body="<!-- jimu-release-automation:v0.3.5 final -->"),
            pull(94, "dependabot/go/foo", "dependabot-updates", author="dependabot[bot]", merged=True, state="closed"),
            pull(95, "dependabot-cli/v0.3.5/go/bar", "dependabot-updates", author="release-app[bot]", body="<!-- jimu-release-automation:v0.3.5 dependency:gomod:bar -->"),
            pull(96, "dependabot/go/security", "master", author="dependabot[bot]"),
            pull(97, "dependabot/go/old", "dependabot-updates", author="dependabot[bot]", created_at="2026-09-01T00:00:00Z"),
            pull(98, "dependabot-cli/v0.3.4/go/old", "dependabot-updates", author="release-app[bot]", body="<!-- jimu-release-automation:v0.3.4 dependency:gomod:old -->"),
            pull(99, "manual", "dependabot-updates"),
            pull(100, "feature/later", "release/v0.3.6"),
            pull(101, "dependabot-cli/v0.3.5/go/unmarked", "dependabot-updates", author="release-app[bot]"),
            pull(102, "dependabot-cli/v0.3.5/go/foreign", "dependabot-updates", author="other-app[bot]", body="<!-- jimu-release-automation:v0.3.5 dependency:gomod:foreign -->"),
            pull(103, "manual", "dependabot-updates", author="release-app[bot]", body="<!-- jimu-release-automation:v0.3.5 dependency:gomod:manual -->"),
            pull(104, "dependabot/go/oldsecurity", "master", author="dependabot[bot]", created_at="2026-09-01T00:00:00Z"),
        ]
        pulls.append(pulls[0].copy())
        self.write(issues=[issue()], comments={"73": [bootstrap()]}, pulls=pulls)
        self.run_command("sync")
        body = self.read()["issues"][0]["body"]
        for number in range(90, 97):
            self.assertEqual(body.count(f"https://github.com/{REPO}/pull/{number})"), 1, body)
        for number in range(97, 105):
            self.assertNotIn(f"/pull/{number})", body)
        self.assertNotIn("api.github.com", body)
        self.assertIn("feature", body)
        self.assertIn("fix", body)
        self.assertIn("aggregate", body)
        self.assertIn("final", body)
        self.assertIn("dependency", body)
        self.assertIn("security", body)
        self.assertIn("merged", body)
        self.assertIn("closed", body)
        self.assertIn("open", body)
        self.assertLess(body.index("/pull/90)"), body.index("/pull/96)"))

    def test_legacy_dependencies_created_after_the_cycle_closed_are_excluded(self):
        target = issue(state="closed")
        target["closed_at"] = "2026-10-08T02:00:00Z"
        self.write(issues=[target], comments={"73": [bootstrap()]},
                   pulls=[pull(80, "dependabot/go/later", "master", author="dependabot[bot]", created_at="2026-10-09T00:00:00Z")])
        self.run_command("sync", VERSION)
        self.assertNotIn("/pull/80)", self.read()["issues"][0]["body"])

    def test_sync_without_active_cycle_performs_no_write(self):
        self.write()
        result = self.run_command("sync")
        self.assertEqual(result.stdout, "active=false\n")
        self.assertEqual(self.patches(), [])

    def test_pagination_finds_the_cycle_and_the_last_pr(self):
        irrelevant = [issue(number, "v1.0.0", owner="CONTRIBUTOR") for number in range(1001, 1101)]
        pulls = [pull(number, "unused", "other") for number in range(1, 101)]
        pulls.append(pull(201, "fix/end", "release/v0.3.5"))
        self.write(issues=irrelevant + [issue()], comments={"73": [bootstrap()]}, pulls=pulls)
        self.run_command("sync")
        target = self.read()["issues"][-1]
        self.assertIn("/pull/201)", target["body"])

    def test_sync_preserves_a_human_edit_made_during_pr_pagination(self):
        changed_body = issue()["body"] + "\n人类在扫描 PR 期间新增的范围\n"
        self.write(issues=[issue()], comments={"73": [bootstrap()]},
                   pulls=[pull(number, "unused", "other") for number in range(1, 101)],
                   during_pull_read={"page": 2, "number": 73, "changes": {"body": changed_body}})
        self.run_command("sync", VERSION)
        self.assertTrue(self.read()["issues"][0]["body"].startswith(changed_body))
        self.assertIn("Release automation", self.read()["issues"][0]["body"])

    def test_publish_preserves_human_body_and_label_edits_during_pr_pagination(self):
        changed_body = issue()["body"] + "\n发版期间补充的验收说明\n"
        self.write(issues=[issue()], comments={"73": [bootstrap()]},
                   pulls=[pull(number, "unused", "other") for number in range(1, 101)],
                   release={"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL},
                   during_pull_read={"page": 2, "number": 73, "changes": {
                       "body": changed_body,
                       "labels": [{"name": "release: candidate"}, {"name": "human-added"}],
                   }})
        self.run_command("published", VERSION, RELEASE_URL)
        target = self.read()["issues"][0]
        self.assertTrue(target["body"].startswith(changed_body))
        self.assertEqual({label["name"] for label in target["labels"]}, {"human-added", "release: published"})
        self.assertEqual(target["state"], "closed")
        self.assertFalse(any("labels" in patch["payload"] for patch in self.patches()))

    def test_issue_title_changed_during_pr_scan_prevents_all_writes(self):
        self.write(issues=[issue()], comments={"73": [bootstrap()]},
                   during_pull_read={"page": 1, "number": 73, "changes": {"title": "release: v0.3.6"}})
        self.run_command("sync", VERSION, success=False)
        self.assertEqual(self.writes(), [])

    def test_published_updates_content_and_labels_before_closing_and_retries(self):
        target = issue(labels=["release: blocked", "release: candidate", "documentation"])
        self.write(issues=[target], comments={"73": [bootstrap()]},
                   release={"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL})
        self.run_command("published", VERSION, RELEASE_URL)
        writes = self.writes()
        self.assertEqual(len(writes), 5)
        self.assertIn(RELEASE_URL, writes[0]["payload"]["body"])
        self.assertIn("published", writes[0]["payload"]["body"])
        self.assertEqual(writes[1]["method"], "POST")
        self.assertEqual(writes[1]["payload"], {"labels": ["release: published"]})
        self.assertEqual({call["endpoint"] for call in writes if call["method"] == "DELETE"}, {
            f"repos/{REPO}/issues/73/labels/release%3A%20blocked",
            f"repos/{REPO}/issues/73/labels/release%3A%20candidate",
        })
        self.assertEqual(writes[-1]["payload"], {"state": "closed", "state_reason": "completed"})
        self.assertEqual({label["name"] for label in self.read()["issues"][0]["labels"]}, {"documentation", "release: published"})
        self.run_command("published", VERSION, RELEASE_URL)
        self.assertEqual(len(self.writes()), 5)
        self.run_command("sync", VERSION)
        self.assertEqual(len(self.writes()), 5)

    def test_unpublished_release_and_unmanaged_issue_never_close(self):
        cases = [
            ({"tag_name": VERSION, "draft": True, "published_at": None, "html_url": RELEASE_URL}, [bootstrap()]),
            ({"tag_name": VERSION, "draft": False, "published_at": None, "html_url": RELEASE_URL}, [bootstrap()]),
            ({"tag_name": "v0.3.4", "draft": False, "published_at": START, "html_url": RELEASE_URL}, [bootstrap()]),
            ({"tag_name": VERSION, "draft": False, "published_at": START, "html_url": "https://wrong.example"}, [bootstrap()]),
            ({"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL}, []),
        ]
        for release, comments in cases:
            with self.subTest(release=release, comments=comments):
                self.write(issues=[issue()], comments={"73": comments}, release=release)
                self.run_command("published", VERSION, RELEASE_URL, success=False)
                self.assertEqual(self.patches(), [])

    def test_api_failure_never_marks_the_issue_published_or_closed(self):
        self.write(issues=[issue()], comments={"73": [bootstrap()]},
                   release={"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL},
                   errors={f"PATCH repos/{REPO}/issues/73": "Forbidden (HTTP 403)"})
        self.run_command("published", VERSION, RELEASE_URL, success=False)
        self.assertEqual(len(self.patches()), 1)
        self.assertNotIn("state", self.patches()[0]["payload"])
        self.assertEqual(self.read()["issues"][0]["state"], "open")

    def test_label_failure_keeps_issue_open_and_retry_finishes_publication(self):
        self.write(issues=[issue()], comments={"73": [bootstrap()]},
                   release={"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL},
                   errors={f"POST repos/{REPO}/issues/73/labels": "Forbidden (HTTP 403)"})
        self.run_command("published", VERSION, RELEASE_URL, success=False)
        self.assertEqual(self.read()["issues"][0]["state"], "open")
        self.assertFalse(any("state" in patch["payload"] for patch in self.patches()))
        data = self.read()
        del data["errors"]
        self.fixture.write_text(json.dumps(data))
        self.run_command("published", VERSION, RELEASE_URL)
        self.assertEqual(self.read()["issues"][0]["state"], "closed")
        self.assertEqual(sum("body" in patch["payload"] for patch in self.patches()), 1)

    def test_missing_release_unknown_version_or_nonowner_issue_never_close(self):
        self.run_command("published", VERSION, RELEASE_URL, success=False)
        self.assertEqual(self.patches(), [])
        self.run_command("published", "v0.3.6", RELEASE_URL, success=False)
        self.assertEqual(self.patches(), [])
        self.write(issues=[issue(owner="CONTRIBUTOR")], comments={"73": [bootstrap()]},
                   release={"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL})
        self.run_command("published", VERSION, RELEASE_URL, success=False)
        self.assertEqual(self.patches(), [])

    def test_transition_label_removal_failure_keeps_issue_open(self):
        self.write(issues=[issue()], comments={"73": [bootstrap()]},
                   release={"tag_name": VERSION, "draft": False, "published_at": START, "html_url": RELEASE_URL},
                   errors={f"DELETE repos/{REPO}/issues/73/labels/release%3A%20collecting": "Forbidden (HTTP 403)"})
        self.run_command("published", VERSION, RELEASE_URL, success=False)
        self.assertEqual(self.read()["issues"][0]["state"], "open")
        self.assertFalse(any("state" in patch["payload"] for patch in self.patches()))

    def test_read_failure_does_not_disappear_as_no_active_cycle(self):
        self.write(errors={f"GET repos/{REPO}/issues?state=open&per_page=100&page=1": "Forbidden (HTTP 403)"})
        result = self.run_command("active", success=False)
        self.assertNotIn("active=false", result.stdout)


if __name__ == "__main__":
    unittest.main()

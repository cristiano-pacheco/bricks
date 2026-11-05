#!/usr/bin/env python3
"""Validate maintainer records, not LLM reference selection."""

import argparse
import json
import math
import re
from pathlib import Path
from urllib.parse import unquote, urlsplit


STATES = {"planned", "verified", "blocked"}
TARGETS = {"router": 1000, "reference": 2000, "combined": 4000}


def local_path(root, value):
    if not isinstance(value, str) or not value or Path(value).is_absolute():
        raise ValueError(f"invalid relative path: {value!r}")
    path = (root / value).resolve()
    if not path.is_relative_to(root.resolve()):
        raise ValueError(f"path escapes root: {value}")
    return path


def token_estimate(text, router=False):
    if router and text.startswith("---\n"):
        text = re.sub(r"\A---\n.*?\n---(?:\n|$)", "", text, count=1, flags=re.S)
    return math.ceil(len(text.encode("utf-8")) / 4)


def passed(checks):
    return bool(checks) and all(
        isinstance(check, dict)
        and check.get("command")
        and check.get("scope")
        and check.get("result") == "passed"
        for check in checks
    )


def markdown_links(root):
    errors = []
    for source in sorted(root.rglob("*.md")):
        text = source.read_text()
        text = re.sub(r"^(`{3,}|~{3,}).*?^\1[^\n]*$", "", text, flags=re.M | re.S)
        targets = re.findall(r"\[[^\]\n]*\]\(([^\s)]+)(?:\s+[^)]*)?\)", text)
        definitions = dict(re.findall(r"^\s{0,3}\[([^\]]+)\]:\s*<?([^\s>]+)>?", text, re.M))
        targets.extend(definitions.values())
        for label in re.findall(r"\[[^\]\n]+\]\[([^\]\n]+)\]", text):
            if label not in definitions:
                errors.append(f"{source}: undefined link reference {label}")
        for target in targets:
            parts = urlsplit(target.strip("<>"))
            if parts.scheme or parts.netloc:
                continue
            destination = (source.parent / unquote(parts.path)).resolve() if parts.path else source
            if not destination.is_relative_to(root.resolve()):
                errors.append(f"{source}: consumer link escapes skill: {target}")
            elif not destination.is_file():
                errors.append(f"{source}: missing link target: {target}")
            elif parts.fragment and destination.suffix == ".md":
                headings = re.findall(r"^#{1,6}\s+(.+?)(?:\s+#+)?$", destination.read_text(), re.M)
                anchors = set()
                counts = {}
                for heading in headings:
                    anchor = re.sub(r"[^\w\- ]", "", heading.lower()).replace(" ", "-")
                    count = counts.get(anchor, 0)
                    counts[anchor] = count + 1
                    anchors.add(f"{anchor}-{count}" if count else anchor)
                anchors.update(re.findall(r'(?:id|name)=[\"\']([^\"\']+)', destination.read_text()))
                if unquote(parts.fragment) not in anchors:
                    errors.append(f"{source}: missing anchor: {target}")
    return errors


def validate(root, repo, coverage, routing, final=False):
    errors = []
    packages = coverage.get("packages", [])
    actual = {
        str(path.parent.relative_to(repo))
        for path in (repo / "pkg").rglob("*.go")
        if not path.name.endswith("_test.go")
    }
    recorded = [package["path"] for package in packages]
    if set(recorded) != actual or len(recorded) != len(set(recorded)):
        errors.append(f"package coverage mismatch: missing={sorted(actual - set(recorded))}, extra={sorted(set(recorded) - actual)}")
    if not coverage.get("revision"):
        errors.append("coverage needs source revision")
    ids = set()
    destinations = set()
    for package in packages:
        name = package["path"]
        package_path = local_path(repo, name)
        readme = package.get("readme")
        expected_readme = package_path / "README.md"
        if expected_readme.exists() and readme != str(expected_readme.relative_to(repo)):
            errors.append(f"{name}: README not inventoried")
        if readme and not local_path(repo, readme).is_file():
            errors.append(f"{name}: missing inventoried README")
        exports = package.get("exports", [])
        if not exports or len(exports) != len(set(exports)):
            errors.append(f"{name}: missing or duplicate exports")
        # Top-level declarations provide an independent omission check. Fields and
        # grouped declarations remain part of the source-backed inventory.
        for source in package_path.glob("*.go"):
            if source.name.endswith("_test.go"):
                continue
            declarations = re.findall(r"^(?:func|type|var|const)\s+([A-Z]\w*)", source.read_text(), re.M)
            for declaration in declarations:
                if declaration not in exports:
                    errors.append(f"{name}: missing export {declaration}")
        branches = package.get("branches", [])
        if not branches:
            errors.append(f"{name}: no public branches")
        for branch in branches:
            identity = branch.get("id")
            if not identity or identity in ids:
                errors.append(f"{name}: missing or duplicate branch id {identity}")
            ids.add(identity)
            if not branch.get("capability"):
                errors.append(f"{identity}: missing capability")
            reference = branch.get("reference")
            destination = local_path(root, reference)
            destinations.add(reference)
            state = branch.get("status")
            if state not in STATES:
                errors.append(f"{identity}: invalid coverage state")
            if state == "blocked" and not branch.get("blocker"):
                errors.append(f"{identity}: missing blocker")
            if state == "verified" and (not destination.is_file() or not passed(branch.get("verification", []))):
                errors.append(f"{identity}: verified coverage needs a real reference and passing evidence")
            if final and state != "verified":
                errors.append(f"{identity}: final coverage is {state}, not verified")
            examples = branch.get("examples", [])
            if not examples:
                errors.append(f"{identity}: missing example classification")
            for example in examples:
                kind, status = example.get("kind"), example.get("status")
                if kind not in {"fragment", "runnable"} or status not in STATES:
                    errors.append(f"{identity}: invalid example classification or state")
                path = local_path(root, example["path"]) if example.get("path") else None
                if path and not path.exists():
                    errors.append(f"{identity}: missing example path")
                if status == "verified":
                    if not path or not passed(example.get("checks", [])):
                        errors.append(f"{identity}: verified example needs a path and passing checks")
                    if kind == "runnable":
                        checks = {check.get("kind") for check in example.get("checks", []) if check.get("result") == "passed"}
                        required = {"compile", "runtime"}
                        if example.get("fx"):
                            required.add("graph")
                        if not required <= checks:
                            errors.append(f"{identity}: runnable example lacks {sorted(required - checks)} proof")
                if final and status != "verified":
                    errors.append(f"{identity}: final example is not verified")
    for reference in coverage.get("supporting_references", []):
        identity = reference.get("id")
        path = reference.get("reference")
        destination = local_path(root, path)
        destinations.add(path)
        if not identity or identity in ids:
            errors.append(f"missing or duplicate supporting reference id {identity}")
        ids.add(identity)
        state = reference.get("status")
        if state not in STATES or not reference.get("capability"):
            errors.append(f"{identity}: invalid supporting reference")
        if state == "blocked" and not reference.get("blocker"):
            errors.append(f"{identity}: missing blocker")
        if state == "verified" and (not destination.is_file() or not passed(reference.get("verification", []))):
            errors.append(f"{identity}: verified reference needs a real destination and passing evidence")
        examples = reference.get("examples", [])
        if not examples:
            errors.append(f"{identity}: missing example classification")
        for example in examples:
            if example.get("kind") != "fragment" or example.get("status") not in STATES:
                errors.append(f"{identity}: supporting reference examples must be classified fragments")
            if example.get("status") == "verified" and (not example.get("path")
                    or not local_path(root, example["path"]).is_file()
                    or not passed(example.get("checks", []))):
                errors.append(f"{identity}: verified fragment needs a real path and passing checks")
            if final and example.get("status") != "verified":
                errors.append(f"{identity}: final example is not verified")
        if final and state != "verified":
            errors.append(f"{identity}: final reference is not verified")
    cases = routing.get("cases", [])
    case_ids = [case.get("id") for case in cases]
    if not cases or None in case_ids or len(case_ids) != len(set(case_ids)):
        errors.append("missing or duplicate routing cases")
    reports = {}
    router = root / "SKILL.md"
    measured = {"SKILL.md": token_estimate(router.read_text(), router=True)} if router.is_file() else {}
    if final and not router.is_file():
        errors.append("missing consumer router SKILL.md")
    for reference in destinations:
        path = local_path(root, reference)
        if path.is_file():
            measured[reference] = token_estimate(path.read_text())
    for case in cases:
        required, excluded = case.get("required", []), case.get("excluded", [])
        if not case.get("task") or not isinstance(case.get("deduplicate"), bool) or "clarification" not in case:
            errors.append(f"{case.get('id')}: incomplete routing contract")
        if set(required) & set(excluded):
            errors.append(f"{case['id']}: required and excluded references overlap")
        conditional = case.get("conditional", [])
        for followup in conditional:
            if not followup.get("when") or not followup.get("references"):
                errors.append(f"{case['id']}: incomplete conditional pointer")
        references = set(required + excluded + [ref for item in conditional for ref in item.get("references", [])])
        for reference in references:
            path = local_path(root, reference)
            if final and not path.is_file():
                errors.append(f"{case['id']}: missing routing target {reference}")
            if path.is_file():
                measured[reference] = token_estimate(path.read_text())
        selected = {"SKILL.md", *required}
        missing = sorted(selected - measured.keys())
        reports[case["id"]] = {
            "tokens": None if missing else sum(measured[path] for path in selected),
            "unmeasured": missing,
            "conditional": [{"when": item["when"], "tokens": sum(measured[ref] for ref in set(item["references"])) if all(ref in measured for ref in item["references"]) else None} for item in conditional],
        }
        if final:
            evaluations = [item for item in routing.get("evaluations", []) if item.get("case") == case["id"]]
            if not evaluations:
                errors.append(f"{case['id']}: missing observed routing evaluation")
            for evaluation in evaluations:
                reads = evaluation.get("reads", [])
                if (not evaluation.get("model") or not evaluation.get("evidence")
                        or evaluation.get("result") != "passed"
                        or not evaluation.get("consumer_only")
                        or not set(required) <= set(reads)
                        or set(excluded) & set(reads)
                        or (case["deduplicate"] and len(reads) != len(set(reads)))
                        or (case["clarification"] and not evaluation.get("clarification"))):
                    errors.append(f"{case['id']}: invalid routing evaluation")
    errors.extend(markdown_links(root))
    overruns = []
    for path, tokens in measured.items():
        target = TARGETS["router" if path == "SKILL.md" else "reference"]
        if tokens > target:
            overruns.append({"path": path, "tokens": tokens, "target": target})
    for identity, report in reports.items():
        if report["tokens"] is not None and report["tokens"] > TARGETS["combined"]:
            overruns.append({"task": identity, "tokens": report["tokens"], "target": TARGETS["combined"]})
    return errors, {"method": "ceil(UTF-8 bytes / 4); router frontmatter excluded", "soft_targets": TARGETS, "overruns": overruns, "documents": dict(sorted(measured.items())), "tasks": reports}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1] / "skills/bricks")
    parser.add_argument("--final", action="store_true")
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[1]
    try:
        coverage = json.loads((args.root / "acceptance/coverage.json").read_text())
        routing = json.loads((args.root / "acceptance/routing.json").read_text())
        errors, report = validate(args.root, repo, coverage, routing, args.final)
    except (OSError, ValueError, KeyError, TypeError) as error:
        errors, report = [str(error)], {}
    print(json.dumps({"result": "failed" if errors else "passed", "errors": errors, "measurements": report}, indent=2))
    return bool(errors)


if __name__ == "__main__":
    raise SystemExit(main())

import copy
import importlib.util
import tempfile
import unittest
from pathlib import Path


SPEC = importlib.util.spec_from_file_location("check_bricks_skill", Path(__file__).with_name("check-bricks-skill.py"))
CHECK = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CHECK)


class AcceptanceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.root = self.repo / "skills/bricks"
        self.root.mkdir(parents=True)
        package = self.repo / "pkg/example"
        package.mkdir(parents=True)
        (package / "example.go").write_text("package example\nfunc New() {}\n")
        (self.root / "usage.md").write_text("# Usage\n\nA short fragment.\n")
        (self.root / "SKILL.md").write_text("---\nname: bricks\n---\n# Bricks\n[Usage](usage.md#usage)\n")
        self.evidence = {"command": "python3 fixture.py", "scope": "fragment review", "result": "passed"}
        self.coverage = {"revision": "fixture", "packages": [{
            "path": "pkg/example", "readme": None, "exports": ["New"],
            "branches": [{"id": "example", "capability": "Construct example", "reference": "usage.md",
                          "status": "verified", "verification": [self.evidence],
                          "examples": [{"kind": "fragment", "path": "usage.md", "status": "verified", "checks": [self.evidence]}]}],
        }]}
        self.routing = {"cases": [{"id": "example", "task": "Construct example", "required": ["usage.md"],
                                  "excluded": [], "conditional": [], "clarification": None, "deduplicate": True}],
                        "evaluations": [{"case": "example", "model": "fixture-model", "evidence": "fixture transcript",
                                         "result": "passed", "consumer_only": True, "reads": ["usage.md"]}]}

    def validate(self, final=False):
        return CHECK.validate(self.root, self.repo, self.coverage, self.routing, final)

    def branch(self):
        return self.coverage["packages"][0]["branches"][0]

    def test_verified_fixture_passes_final(self):
        errors, report = self.validate(final=True)
        self.assertEqual(errors, [])
        self.assertGreater(report["tasks"]["example"]["tokens"], 0)

    def test_missing_package_and_export_fail(self):
        original = copy.deepcopy(self.coverage)
        self.coverage["packages"] = []
        self.assertIn("package coverage mismatch", "\n".join(self.validate()[0]))
        self.coverage = original
        self.coverage["packages"][0]["exports"] = ["Different"]
        self.assertIn("missing export New", "\n".join(self.validate()[0]))

    def test_planned_is_not_final_or_measured_as_zero(self):
        branch = self.branch()
        branch.update(status="planned", reference="future.md", verification=[])
        branch["examples"][0].update(status="planned", path=None, checks=[])
        self.routing["cases"][0]["required"] = ["future.md"]
        errors, report = self.validate()
        self.assertEqual(errors, [])
        self.assertIsNone(report["tasks"]["example"]["tokens"])
        self.assertIn("future.md", report["tasks"]["example"]["unmeasured"])
        self.assertIn("not verified", "\n".join(self.validate(final=True)[0]))

    def test_verified_requires_evidence_and_real_target(self):
        self.branch()["verification"] = []
        self.assertIn("passing evidence", "\n".join(self.validate()[0]))
        self.branch()["verification"] = [self.evidence]
        (self.root / "usage.md").unlink()
        self.assertIn("real reference", "\n".join(self.validate()[0]))

    def test_blocked_needs_reason(self):
        self.branch()["status"] = "blocked"
        self.assertIn("missing blocker", "\n".join(self.validate()[0]))
        self.branch()["blocker"] = "Service unavailable"
        self.assertEqual(self.validate()[0], [])
        self.assertTrue(self.validate(final=True)[0])

    def test_links_check_files_anchors_and_references(self):
        for link in ("[Broken](missing.md)", "[Broken](usage.md#absent)", "[Broken][absent]", "[Escape](../outside.md)"):
            with self.subTest(link=link):
                (self.root / "broken.md").write_text(link)
                self.assertTrue(self.validate()[0])
        (self.root / "broken.md").write_text("[Valid][usage]\n\n[usage]: usage.md#usage\n\n```md\n[Fragment](not-real.md)\n```\n")
        self.assertEqual(self.validate()[0], [])

    def test_example_classification_and_runtime_graph_proof(self):
        example = self.branch()["examples"][0]
        example["kind"] = "unknown"
        self.assertIn("invalid example classification", "\n".join(self.validate()[0]))
        example.update(kind="runnable", fx=True)
        self.assertIn("proof", "\n".join(self.validate()[0]))
        example["checks"] = [dict(self.evidence, kind=kind) for kind in ("compile", "runtime", "graph")]
        self.assertEqual(self.validate(final=True)[0], [])
        example["checks"][1]["result"] = "not run"
        self.assertTrue(self.validate()[0])

    def test_routing_observation_is_required(self):
        self.routing["evaluations"] = []
        self.assertEqual(self.validate()[0], [])
        self.assertIn("missing observed", "\n".join(self.validate(final=True)[0]))

    def test_routing_exclusions_deduplication_and_clarification(self):
        evaluation = self.routing["evaluations"][0]
        evaluation["reads"] = ["usage.md", "usage.md"]
        self.assertIn("invalid routing evaluation", "\n".join(self.validate(final=True)[0]))
        evaluation["reads"] = ["usage.md", "unrelated.md"]
        self.routing["cases"][0]["excluded"] = ["unrelated.md"]
        self.assertTrue(self.validate(final=True)[0])
        evaluation["reads"] = ["usage.md"]
        self.routing["cases"][0].update(excluded=[], clarification="Ask which topology")
        self.assertTrue(self.validate(final=True)[0])
        evaluation["clarification"] = "Which topology?"
        self.assertEqual(self.validate(final=True)[0], [])

    def test_token_method_and_deduplicated_combined_load(self):
        self.assertEqual(CHECK.token_estimate("é"), 1)
        self.assertEqual(CHECK.token_estimate("abcde"), 2)
        self.assertEqual(CHECK.token_estimate("---\nname: ignored\n---\nabcde", router=True), 2)
        self.routing["cases"][0]["required"] = ["usage.md", "usage.md"]
        errors, report = self.validate()
        self.assertEqual(errors, [])
        expected = sum(report["documents"].values())
        self.assertEqual(report["tasks"]["example"]["tokens"], expected)
        self.assertEqual(report["soft_targets"], {"router": 1000, "reference": 2000, "combined": 4000})

    def test_soft_overruns_are_reported_not_failed(self):
        (self.root / "usage.md").write_text("# Usage\n" + "a" * 20000)
        errors, report = self.validate()
        self.assertEqual(errors, [])
        self.assertEqual(len(report["overruns"]), 2)
        self.assertEqual(report["documents"]["usage.md"], 5002)

    def test_final_requires_router(self):
        (self.root / "SKILL.md").unlink()
        self.assertIn("missing consumer router", "\n".join(self.validate(final=True)[0]))

    def test_paths_are_bounded(self):
        for value in ("../outside.md", "/tmp/outside.md", ""):
            with self.subTest(value=value), self.assertRaises(ValueError):
                CHECK.local_path(self.root, value)


if __name__ == "__main__":
    unittest.main()

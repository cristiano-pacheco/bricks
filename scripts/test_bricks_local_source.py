"""Exercise the local-source reference with offline, temporary Go consumers.

Fixtures model dependency selection, not Bricks runtime behavior. No production
router or discovery helper is introduced: tests execute the documented Go calls.
"""

import hashlib
import http.server
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import threading
import unittest
import zipfile


MODULE = "github.com/cristiano-pacheco/bricks"
PACKAGE = MODULE + "/pkg/http/request"
REFERENCE = Path(__file__).resolve().parents[1] / "skills/bricks/references/local-source.md"


class RetrievalTrap(http.server.BaseHTTPRequestHandler):
    requests = []

    def do_GET(self):
        self.requests.append(self.path)
        self.send_error(503)

    def do_CONNECT(self):
        self.requests.append(self.path)
        self.send_error(503)

    def log_message(self, *_):
        pass


class LocalSourceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not shutil.which("go") or not shutil.which("rg"):
            raise RuntimeError("local-source acceptance requires Go and rg")
        cls.server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), RetrievalTrap)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.app = self.root / "app"
        self.cache = self.root / "configured-cache"
        self.app.mkdir()
        self.cache.mkdir()
        self.put(self.app / "go.mod", f"module consumer.test/app\n\ngo 1.22.0\n\nrequire {MODULE} v1.0.0\n")
        self.put(self.app / "main.go", f'package main\nimport _ "{PACKAGE}"\nfunc main() {{}}\n')
        self.env = dict(os.environ, GOTOOLCHAIN="local", GOPROXY="off", GOSUMDB="off",
                        GOVCS="*:off", GOMODCACHE=str(self.cache), GOWORK="off", GOFLAGS="",
                        GOCACHE=str(self.root / "build-cache"), GOENV="off",
                        HTTP_PROXY=f"http://127.0.0.1:{self.server.server_port}",
                        HTTPS_PROXY=f"http://127.0.0.1:{self.server.server_port}",
                        NO_PROXY="", GONOPROXY="", GOPRIVATE="", GONOSUMDB="")
        self.request_count = len(RetrievalTrap.requests)

    def put(self, path, text):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)

    def source(self, path, marker="selected"):
        self.put(path / "go.mod", f"module {MODULE}\n\ngo 1.22.0\n")
        self.put(path / "pkg/http/request/request.go",
                 f'package request\n// {marker}\nfunc ReadJSON() {{}}\n')
        self.put(path / "pkg/http/request/request_test.go",
                 'package request\n// ReadJSON contract evidence\n')
        return path

    def cached(self, source=True):
        metadata = self.cache / "cache/download" / MODULE / "@v/v1.0.0.mod"
        self.put(metadata, f"module {MODULE}\n\ngo 1.22.0\n")
        self.put(metadata.with_suffix(".info"), '{"Version":"v1.0.0","Time":"2024-01-01T00:00:00Z"}')
        # Populate a real Go cache from a fixture-only file proxy before taking
        # discovery snapshots. This also creates valid sums and zip hashes.
        proxy = self.root / "fixture-proxy"
        version = proxy / MODULE / "@v/v1.0.0.mod"
        self.put(version.with_suffix(".mod"), metadata.read_text())
        self.put(version.with_suffix(".info"), metadata.with_suffix(".info").read_text())
        with zipfile.ZipFile(version.with_suffix(".zip"), "w") as archive:
            prefix = MODULE + "@v1.0.0/"
            archive.writestr(prefix + "go.mod", metadata.read_text())
            archive.writestr(prefix + "pkg/http/request/request.go", "package request\nfunc ReadJSON() {}\n")
            archive.writestr(prefix + "pkg/http/request/request_test.go", "package request\n// ReadJSON evidence\n")
        setup = subprocess.run(["go", "mod", "download", MODULE + "@v1.0.0"],
                               cwd=self.app, env=dict(self.env, GOPROXY=proxy.as_uri()),
                               text=True, capture_output=True, timeout=30)
        self.assertEqual(setup.returncode, 0, setup.stderr)
        directory = self.cache / (MODULE + "@v1.0.0")
        if not source:
            # Go marks extracted cache directories read-only.
            for path in directory.rglob("*"):
                if path.is_dir():
                    path.chmod(0o755)
            directory.chmod(0o755)
            shutil.rmtree(directory)
            metadata.with_suffix(".zip").unlink()
        return directory

    def snapshot(self):
        # Build cache is Go-owned scratch space, not dependency/source evidence.
        return {str(p.relative_to(self.root)): hashlib.sha256(p.read_bytes()).hexdigest()
                for p in self.root.rglob("*") if p.is_file()
                and "build-cache" not in p.relative_to(self.root).parts}

    def go(self, *args, success=True):
        result = subprocess.run(["go", *args], cwd=self.app, env=self.env,
                                text=True, capture_output=True, timeout=30)
        if success:
            self.assertEqual(result.returncode, 0, result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout)
        return result

    def inspect(self, expected, vendor=False):
        before = self.snapshot()
        settings = json.loads(self.go("env", "-json", "GOMOD", "GOWORK", "GOFLAGS", "GOMODCACHE", "GOVERSION").stdout)
        self.assertEqual(settings["GOMODCACHE"], str(self.cache))
        if not vendor:
            module = json.loads(self.go("list", "-mod=readonly", "-m", "-json", MODULE).stdout)
            self.assertEqual(Path(module["Dir"]).resolve(), expected.resolve())
        mode = "vendor" if vendor else "readonly"
        package = json.loads(self.go("list", f"-mod={mode}", "-find", "-json", PACKAGE).stdout)
        directory = Path(package["Dir"]).resolve()
        self.assertEqual(directory, (expected / "pkg/http/request").resolve())
        for pattern, glob in [("func ReadJSON|func ReadJSONWithMaxSize", "*.go"),
                              ("ReadJSON|Content-Type", "*_test.go")]:
            result = subprocess.run(["rg", "-n", "-g", glob, pattern, str(directory)],
                                    text=True, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)
            for match in result.stdout.splitlines():
                self.assertTrue(Path(match.split(":", 1)[0]).is_relative_to(directory))
        self.assertEqual(self.snapshot(), before, "discovery changed dependency or source files")
        self.assertEqual(len(RetrievalTrap.requests), self.request_count, "attempted network retrieval")

    def test_configured_cache_and_decoy_version(self):
        expected = self.cached()
        self.source(self.cache / (MODULE + "@v9.0.0"), "wrong version")
        self.inspect(expected)

    def test_local_replacement(self):
        expected = self.source(self.root / "replacement")
        with (self.app / "go.mod").open("a") as stream:
            stream.write(f"\nreplace {MODULE} => ../replacement\n")
        self.inspect(expected)

    def test_workspace_override(self):
        old = self.source(self.root / "module-replacement", "wrong replacement")
        expected = self.source(self.root / "workspace-bricks")
        with (self.app / "go.mod").open("a") as stream:
            stream.write(f"\nreplace {MODULE} => {old}\n")
        work = self.root / "go.work"
        self.put(work, f"go 1.22.0\nuse ./app\nreplace {MODULE} => ./workspace-bricks\n")
        self.env["GOWORK"] = str(work)
        self.inspect(expected)

    def test_workspace_use_module(self):
        expected = self.source(self.root / "workspace-bricks")
        work = self.root / "go.work"
        self.put(work, "go 1.22.0\nuse (\n./app\n./workspace-bricks\n)\n")
        self.env["GOWORK"] = str(work)
        self.inspect(expected)

    def test_alternate_modfile(self):
        expected = self.source(self.root / "alternate")
        self.put(self.app / "inspect.mod", f"module consumer.test/app\ngo 1.22.0\nrequire {MODULE} v1.0.0\nreplace {MODULE} => ../alternate\n")
        self.env["GOFLAGS"] = "-modfile=inspect.mod -mod=mod"
        self.inspect(expected)

    def vendor(self, root, workspace=False):
        expected = self.source(root / "vendor" / MODULE)
        (expected / "go.mod").unlink()
        prefix = "## workspace\n" if workspace else ""
        self.put(root / "vendor/modules.txt", prefix + f"# {MODULE} v1.0.0\n## explicit; go 1.22.0\n{PACKAGE}\n")
        return expected

    def test_automatic_vendor(self):
        self.cached()  # A different cache location must not win over vendoring.
        expected = self.vendor(self.app)
        self.inspect(expected, vendor=True)
        before = self.snapshot()
        automatic = json.loads(self.go("list", "-find", "-json", PACKAGE).stdout)
        self.assertEqual(Path(automatic["Dir"]), expected / "pkg/http/request")
        self.assertEqual(before, self.snapshot())

    def test_explicit_vendor_with_older_module_directive(self):
        self.put(self.app / "go.mod", f"module consumer.test/app\ngo 1.13\nrequire {MODULE} v1.0.0\n")
        expected = self.vendor(self.app)
        self.env["GOFLAGS"] = "-mod=vendor"
        self.inspect(expected, vendor=True)

    def test_explicit_readonly_overrides_automatic_vendor(self):
        expected = self.cached()
        self.vendor(self.app)
        self.env["GOFLAGS"] = "-mod=readonly"
        self.inspect(expected)

    def test_workspace_vendor(self):
        self.put(self.root / "go.work", "go 1.22.0\nuse ./app\n")
        self.env["GOWORK"] = str(self.root / "go.work")
        expected = self.vendor(self.root, workspace=True)
        self.inspect(expected, vendor=True)
        automatic = json.loads(self.go("list", "-find", "-json", PACKAGE).stdout)
        self.assertEqual(Path(automatic["Dir"]), expected / "pkg/http/request")

    def test_metadata_without_source_does_not_fetch(self):
        self.cached(source=False)
        self.source(self.cache / (MODULE + "@v9.0.0"), "unselected source")
        before = self.snapshot()
        module = json.loads(self.go("list", "-mod=readonly", "-m", "-json", MODULE).stdout)
        self.assertIn("GoMod", module)
        self.assertFalse(module.get("Dir"))
        result = self.go("list", "-mod=readonly", "-find", "-json", PACKAGE, success=False)
        self.assertTrue("disabled" in result.stderr or "go.sum" in result.stderr, result.stderr)
        self.assertEqual(before, self.snapshot())
        self.assertEqual(len(RetrievalTrap.requests), self.request_count)

    def test_missing_metadata_does_not_fetch(self):
        before = self.snapshot()
        result = self.go("list", "-mod=readonly", "-m", "-json", MODULE, success=False)
        self.assertIn("GOPROXY=off", result.stderr)
        self.go("list", "-mod=readonly", "-find", "-json", PACKAGE, success=False)
        self.assertEqual(before, self.snapshot())
        self.assertEqual(len(RetrievalTrap.requests), self.request_count)

    def test_inconsistent_vendor_stops_without_cache_fallback(self):
        self.cached()
        self.vendor(self.app)
        self.put(self.app / "vendor/modules.txt", "")
        before = self.snapshot()
        result = self.go("list", "-mod=vendor", "-find", "-json", PACKAGE, success=False)
        self.assertIn("inconsistent vendoring", result.stderr)
        self.assertEqual(before, self.snapshot())
        self.assertEqual(len(RetrievalTrap.requests), self.request_count)

    def test_incompatible_toolchain_does_not_fetch(self):
        self.put(self.app / "go.mod", "module consumer.test/app\ngo 1.999.0\n")
        before = self.snapshot()
        result = self.go("list", "-mod=readonly", "-m", "-json", MODULE, success=False)
        self.assertIn("GOTOOLCHAIN=local", result.stderr)
        self.assertEqual(before, self.snapshot())
        self.assertEqual(len(RetrievalTrap.requests), self.request_count)

    def test_reference_uses_current_module_and_guarded_commands(self):
        text = REFERENCE.read_text()
        self.assertIn(f"go list -mod=readonly -m -json {MODULE}", text)
        for mode in ("readonly", "vendor"):
            self.assertIn(f"go list -mod={mode} -find -json {PACKAGE}", text)
        for guard in ("GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOVCS='*:off'"):
            self.assertIn(guard, text)


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Tests for backlog-guard.py.

Run: python3 .claude/hooks/backlog-guard_test.py

Two things about this file are deliberate and should not be "tidied":

1. Paths are DERIVED, never hard-coded. This repository's public-release scan rejects the
   local home-directory prefix case-sensitively, in the working tree and in all reachable
   Git history, and history cannot be rewritten here. A hard-coded absolute path would
   permanently red-light the gate.
2. The denied flags are built by concatenation. The guard blocks any Bash command that
   mentions backlog and contains the bare flag, so spelling it literally in a command
   string would make the hook block its own test run.
"""

import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, os.pardir, os.pardir))
HOOK = os.path.join(HERE, "backlog-guard.py")

env = dict(os.environ, CLAUDE_PROJECT_DIR=ROOT)
N = "--" + "notes"
P = "--" + "plan"

cases = [
    # unsafe: the section-replacing flags, in every spelling
    ("bare notes flag",      {"tool_name": "Bash", "tool_input": {"command": f"backlog task edit GCV-0001 {N} hi"}}, 2),
    ("bare plan flag",       {"tool_name": "Bash", "tool_input": {"command": f"backlog task edit GCV-0001 {P} hi"}}, 2),
    ("equals form",          {"tool_name": "Bash", "tool_input": {"command": f"backlog task edit GCV-0001 {N}=hi"}}, 2),
    ("flag at end of line",  {"tool_name": "Bash", "tool_input": {"command": f"backlog task edit GCV-0001 {N}"}}, 2),

    # safe: these MUST exit 0 — a guard that blocks the append forms is useless
    ("append-notes allowed", {"tool_name": "Bash", "tool_input": {"command": "backlog task edit GCV-0001 --append-notes hi"}}, 0),
    ("append-plan allowed",  {"tool_name": "Bash", "tool_input": {"command": "backlog task edit GCV-0001 --append-plan hi"}}, 0),
    ("task list allowed",    {"tool_name": "Bash", "tool_input": {"command": "backlog task list --plain"}}, 0),
    ("finalize allowed",     {"tool_name": "Bash", "tool_input": {"command": "backlog task edit GCV-0001 --check-ac 1 -s Done"}}, 0),
    ("doc update allowed",   {"tool_name": "Bash", "tool_input": {"command": "backlog doc update doc-0002 --content x"}}, 0),
    ("non-backlog cmd",      {"tool_name": "Bash", "tool_input": {"command": f"mytool {N} foo"}}, 0),
    ("gate command allowed", {"tool_name": "Bash", "tool_input": {"command": "./scripts/validate.sh"}}, 0),

    # unsafe: hand-editing CLI-owned markdown
    ("edit task md",         {"tool_name": "Edit",  "tool_input": {"file_path": os.path.join(ROOT, "backlog/tasks/GCV-0001 - x.md")}}, 2),
    ("write doc md",         {"tool_name": "Write", "tool_input": {"file_path": os.path.join(ROOT, "backlog/docs/doc-0002 - y.md")}}, 2),
    ("edit completed md",    {"tool_name": "Edit",  "tool_input": {"file_path": os.path.join(ROOT, "backlog/completed/GCV-0009 - z.md")}}, 2),

    # safe: everything else, including the documented config.yml exception
    ("config.yml allowed",   {"tool_name": "Edit",  "tool_input": {"file_path": os.path.join(ROOT, "backlog/config.yml")}}, 0),
    ("source file allowed",  {"tool_name": "Edit",  "tool_input": {"file_path": os.path.join(ROOT, "platform/function/fn.go")}}, 0),
    ("AGENTS.md allowed",    {"tool_name": "Write", "tool_input": {"file_path": os.path.join(ROOT, "AGENTS.md")}}, 0),
    ("archive json allowed", {"tool_name": "Write", "tool_input": {"file_path": os.path.join(ROOT, "archive/issues-dump.json")}}, 0),
]

fails = 0
for name, payload, want in cases:
    r = subprocess.run([sys.executable, HOOK], input=json.dumps(payload),
                       capture_output=True, text=True, env=env)
    ok = r.returncode == want
    fails += not ok
    print(f"{'PASS' if ok else 'FAIL'}  exit={r.returncode} want={want}  {name}")

# garbage stdin must never block
r = subprocess.run([sys.executable, HOOK], input="not json", capture_output=True, text=True, env=env)
ok = r.returncode == 0
fails += not ok
print(f"{'PASS' if ok else 'FAIL'}  exit={r.returncode} want=0  garbage stdin never blocks")

total = len(cases) + 1
print(f"\n{total - fails}/{total} passed")
sys.exit(1 if fails else 0)

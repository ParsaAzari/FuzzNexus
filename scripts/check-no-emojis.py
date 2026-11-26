#!/usr/bin/env python3
"""
CI Script: check-no-emojis.py
Verifies that no raw emojis or unicode symbol characters exist in frontend source files.
Exit code 0 on clean pass; Exit code 1 if forbidden symbols are found.
"""

import os
import re
import sys

# Disallowed symbol ranges:
# - Emojis & Pictographs (\U00010000-\U0010ffff)
# - Miscellaneous Symbols (\u2600-\u26ff)
# - Dingbats (\u2700-\u27bf)
# - Geometric Shapes (\u25a0-\u25ff)
# - Arrows (\u2190-\u21ff)
DISALLOWED_PATTERN = re.compile(
    r"[\u2600-\u26ff\u2700-\u27bf\u25a0-\u25ff\u2190-\u21ff\U00010000-\U0010ffff]"
)

SRC_DIR = os.path.join(os.path.dirname(__file__), "..", "fuzznexus-desktop", "frontend", "src")

violations = []

for root, _, files in os.walk(SRC_DIR):
    for file in files:
        if file.endswith((".svelte", ".ts", ".js", ".html")):
            file_path = os.path.join(root, file)
            rel_path = os.path.relpath(file_path, os.path.join(SRC_DIR, ".."))
            with open(file_path, "r", encoding="utf-8", errors="ignore") as f:
                for line_idx, line in enumerate(f, 1):
                    matches = DISALLOWED_PATTERN.findall(line)
                    if matches:
                        violations.append((rel_path, line_idx, line.strip(), matches))

if violations:
    print(f"FAILED: Found {len(violations)} raw emoji/symbol violation(s) in frontend source:")
    for path, line_no, content, chars in violations:
        unique_chars = "".join(sorted(set(chars)))
        print(f"  {path}:{line_no} [{unique_chars}] -> {content}")
    print("\nPlease replace raw emojis/symbols with <Icon name=\"...\" /> from lib/components/Icon.svelte.")
    sys.exit(1)

print("PASSED: Zero emojis or raw unicode symbols found in frontend source.")
sys.exit(0)

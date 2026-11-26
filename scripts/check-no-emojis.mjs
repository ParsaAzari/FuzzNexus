#!/usr/bin/env node
/**
 * CI Script: check-no-emojis.mjs
 * Cross-platform check verifying no raw emojis or unicode symbol characters
 * exist in frontend source files.
 */

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const DISALLOWED_PATTERN = /[\u2600-\u26ff\u2700-\u27bf\u25a0-\u25ff\u2190-\u21ff\u{10000}-\u{10ffff}]/gu;
const FRONTEND_DIR = path.resolve(__dirname, '..', 'fuzznexus-desktop', 'frontend');
const SRC_DIR = path.join(FRONTEND_DIR, 'src');
const EXTENSIONS = new Set(['.svelte', '.ts', '.js', '.html']);

const violations = [];

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(fullPath);
    } else if (entry.isFile() && EXTENSIONS.has(path.extname(entry.name))) {
      const relPath = path.relative(FRONTEND_DIR, fullPath);
      const lines = fs.readFileSync(fullPath, 'utf8').split(/\r?\n/);
      lines.forEach((line, idx) => {
        const matches = line.match(DISALLOWED_PATTERN);
        if (matches) {
          violations.push({ relPath, lineNo: idx + 1, content: line.trim(), matches });
        }
      });
    }
  }
}

walk(SRC_DIR);

if (violations.length > 0) {
  console.error(`FAILED: Found ${violations.length} raw emoji/symbol violation(s) in frontend source:`);
  for (const v of violations) {
    const unique = [...new Set(v.matches)].sort().join('');
    console.error(`  ${v.relPath}:${v.lineNo} [${unique}] -> ${v.content}`);
  }
  console.error('\nPlease replace raw emojis/symbols with <Icon name="..." /> from lib/components/Icon.svelte.');
  process.exit(1);
}

console.log('PASSED: Zero emojis or raw unicode symbols found in frontend source.');

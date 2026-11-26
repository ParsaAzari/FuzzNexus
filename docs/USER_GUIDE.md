# FuzzNexus User Guide

This guide walks through installing FuzzNexus, running your first scan in the desktop app, keeping noise out of your results, and working with the command-line engine.

---

## 1. Installation & First Launch

Prebuilt desktop releases are available on the [Releases page](https://github.com/ParsaAzari/FuzzNexus/releases). Each release includes a `SHA256SUMS.txt` file so you can verify downloads before running them.

### macOS (`FuzzNexus-1.0.0-macOS-Universal.dmg`)

1. Open the `.dmg` image and drag **FuzzNexus** into **Applications**.
2. Because open-source builds are not notarized with an Apple Developer ID certificate, macOS Gatekeeper may warn on first launch. Open **System Settings > Privacy & Security** and click **Open Anyway**, or remove the quarantine attribute from Terminal:

```bash
xattr -cr /Applications/FuzzNexus.app
```

### Windows (`FuzzNexus-1.0.0-Windows-x64.zip`)

1. Extract the archive to a folder you control (for example, `C:\Tools\FuzzNexus\`).
2. Run `FuzzNexus.exe`. Windows 10 and 11 include Microsoft Edge WebView2 by default; if WebView2 is missing on an older image, install the Evergreen WebView2 Runtime from Microsoft.
3. If Windows SmartScreen prompts on first launch, choose **More info > Run anyway**.

### Linux (`FuzzNexus-1.0.0-Linux-x86_64.tar.gz`)

1. Extract the archive and ensure the binary is executable:

```bash
tar -xzf FuzzNexus-1.0.0-Linux-x86_64.tar.gz
chmod +x FuzzNexus
./FuzzNexus
```

2. The desktop UI relies on GTK 3 and WebKitGTK. On Debian or Ubuntu, install the runtime libraries if they are not already present:

```bash
sudo apt-get install -y libgtk-3-0 libwebkit2gtk-4.1-0
```

*(On older distributions that ship `4.0` instead of `4.1`, install `libwebkit2gtk-4.0-37`.)*

### Verifying Checksums

Before launching a downloaded archive, compare its SHA-256 hash against `FuzzNexus-1.0.0-SHA256SUMS.txt`:

```bash
# macOS / Linux
shasum -a 256 -c FuzzNexus-1.0.0-SHA256SUMS.txt --ignore-missing
```

```powershell
# Windows (PowerShell)
Get-FileHash .\FuzzNexus-1.0.0-Windows-x64.zip -Algorithm SHA256
```

---

## 2. Running Your First Desktop Scan

1. **Choose a target URL and place `FUZZ`:**
   Enter a URL such as `https://example.org/FUZZ` or `https://example.org/search?q=FUZZ`. If you leave `FUZZ` out of a bare host or path, FuzzNexus appends `/FUZZ` automatically. You can also place `FUZZ` inside custom request headers or a POST body.
2. **Select a wordlist:**
   Pick one of the built-in curated packs or import your own `.txt` wordlist in the **Wordlists** view.
3. **Set concurrency conservatively:**
   Start with modest **Threads** (for example, `10`–`25`) and set a **Rate** limit if you are testing a target with strict WAF or rate-limiting rules.
4. **Start and monitor:**
   Click **Start Scan** to watch matches stream into the live results table, inspect HTTP status codes, sizes, word/line counts, and response times, or pause/stop at any time.

---

## 3. Filtering Noise & Auto-Calibration

Real targets frequently return `200 OK` soft-404 pages or identical redirect bodies for unknown paths. FuzzNexus gives you two complementary ways to separate real findings from baseline noise:

- **Auto-Calibration (`-ac`):** Probes random non-existent paths before scanning to fingerprint baseline soft-404 lengths, word counts, and line counts, then filters matching responses automatically.
- **Matchers and Filters:** Matchers decide *what enters* your results (for example, status `200,204,301,302,403`), while filters *exclude* known noise by status code, byte size, words, lines, regex, or response time. You can also apply live filters during a running scan directly from the filter bar without restarting.

---

## 4. Managing Wordlists Safely

The **Wordlists** manager stores metadata in SQLite and provides built-in packs covering common administrative panels, API routes, cloud/DevOps paths, authentication endpoints, and parameters.

- **Sanitization & Risk Checks:** Before scanning, FuzzNexus inspects imported wordlists for traversal patterns (`../`, `.//`) or payloads likely to trigger instant WAF IP bans and lets you generate a cleaned sandbox copy without modifying your original file.
- **Active-Scan Locking:** Wordlists currently used by an active scan are locked against deletion or destructive edits until the scan finishes or stops.

---

## 5. Safe Stop, Circuit Breaker & Proxying

- **Circuit Breaker & Safe Stop:** If a target starts returning sustained `403` blocks or transport failures, the circuit breaker trips and runs canary checks against benign origin paths to distinguish a genuine IP ban from a noisy wordlist path. When Safe Stop pauses a scan, you can inspect the diagnostic summary, switch VPN/exit IP, test connectivity, and resume from the exact saved index.
- **TLS & Browser Profiles:** Choose a browser TLS fingerprint and matching headers when testing endpoints behind strict TLS-inspecting proxies or CDNs.
- **Proxy & Replay Proxy:** Route all scan traffic through an upstream intercepting proxy (such as Burp Suite or mitmproxy), or configure a **Replay Proxy** so only matched findings are forwarded for manual follow-up.

---

## 6. Inspecting Requests & Exporting Reports

Select any row in the results table to open the **Inspector**, where you can review the raw HTTP request, headers, response body, and extracted scraper matches without re-issuing traffic.

When a scan completes, export findings from the session controls in **JSON**, **CSV**, **Extended JSON/CSV**, **HTML**, or **Markdown** formats, or enable **Audit Logs** to record full request/response transcripts on disk.

---

## 7. CLI Quick Reference

The CLI binary shares the exact same scanning engine as the desktop app. Build it from the repository root:

```bash
go build -o fuzznexus .
```

Common patterns:

```bash
# Directory discovery with auto-calibration and colorized output
./fuzznexus -u https://example.org/FUZZ -w wordlist.txt -ac -c

# Match specific status codes and exclude baseline response size
./fuzznexus -u https://example.org/FUZZ -w wordlist.txt -mc 200,301,302,403 -fs 1248

# Fuzz JSON POST body and filter out error strings
./fuzznexus -u https://example.org/api/login \
  -X POST -H "Content-Type: application/json" \
  -d '{"username":"FUZZ","password":"test"}' \
  -w users.txt -fr "invalid user"

# Multi-wordlist pitchfork mode
./fuzznexus -u https://example.org/login \
  -X POST -d "u=USER&p=PASS" \
  -w users.txt:USER -w passwords.txt:PASS -mode pitchfork

# Export results as HTML and save a full audit log
./fuzznexus -u https://example.org/FUZZ -w wordlist.txt -o report.html -of html -audit-log audit.log
```

For reusable presets, copy [fuzznexusrc.example](../fuzznexusrc.example) to your config path and load it with `-config`. Run `./fuzznexus -h` for the complete flag reference.

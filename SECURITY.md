# Security Policy

## Reporting a vulnerability

FuzzNexus is a security testing tool, and we take vulnerabilities in FuzzNexus
itself seriously. Please report them privately so a fix can be prepared before
the issue is widely known.

Use GitHub's private vulnerability reporting after this repository is published,
or contact the project maintainers through the private channel listed on the
repository's Security page. Do not include sensitive target data in a public
issue.

Please do not open a public issue, pull request, or discussion for a security
vulnerability.

Include as much of the following as you can:

- The version of FuzzNexus affected (`fuzznexus -V`), or the commit if you built
  from source.
- A description of the vulnerability and its impact.
- Steps to reproduce, ideally a minimal command line and input.
- A suggested fix, if you have one.

## Scope

This policy is for vulnerabilities in FuzzNexus's own code. For example: a
crafted target response, wordlist, or config that makes FuzzNexus write outside
its output directory, run an unintended command, leak data, or crash in a way
that could be exploited.

FuzzNexus sends the requests you tell it to. A finding about a *target* you
scanned with FuzzNexus is not a vulnerability in FuzzNexus, and belongs to that
target's owner.

## Supported versions

Fixes are made against the latest release. Before reporting, please confirm the
issue reproduces on the most recent release or on current `main`.

## Disclosure

We will acknowledge your report, work with you on a fix, and coordinate the
disclosure. Please give a reasonable window for a fix to ship before disclosing
publicly.

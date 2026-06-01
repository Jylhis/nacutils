# Engineering Principles

These are the foundational beliefs that guide every technical decision at Jylhis.

## Ship working software

A working feature today beats a perfect feature in three months. We iterate toward quality; we do not block shipping on it.

## Start simple, stay simple

The right abstraction usually isn't visible until the third time you need it. Three similar lines of code is better than a premature abstraction. Add complexity only when its absence is the bottleneck.

## Everything in code

Infrastructure, policy, configuration, and process live in version-controlled files. If it's not in a repo it doesn't exist.

## Automate the boring parts

Repetitive human work is a bug. Lint, format, test, release, and deploy automatically. Humans review intent, not syntax.

## Security by default

Secrets never touch source code. Dependencies are pinned and audited. Branch protection and secret scanning run on every repo, from day one.

## Observability from day one

If you can't measure it, you can't improve it. Structured logs, clear exit codes, and usage that surfaces errors early are not optional extras.

## Own the whole system

Engineers own the full lifecycle of what they build: design, tests, deploy, monitor, on-call. There is no "throw it over the fence."

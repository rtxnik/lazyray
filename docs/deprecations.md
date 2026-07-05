# Deprecation policy

When a covered surface (see [stability.md](stability.md)) has to change, it is
deprecated first, then removed. The rule:

> A deprecated feature keeps working for **at least 2 minor releases AND at
> least 6 months** after the release in which its runtime deprecation warning
> first appeared — whichever ends later.

## What counts as a deprecation

- Removing or renaming a flag, command, or environment variable
- Removing a `--json` field
- Dropping support for a config schema version
- Changing a default in a behavior-visible way

## Mechanics

1. The deprecation lands with all three of: a runtime warning printed when the
   deprecated surface is used, a CHANGELOG entry under `### Deprecated`, and a
   row in the table below.
2. The warning names the replacement and the earliest release that may remove
   the feature.
3. Removal happens only after the window above, in a **major** release.
   Exception: config-format changes that ship with an automatic migrator (the
   old format still loads and is converted in place) count as **minor** and
   need no removal window.

## Current deprecations

| Surface | Deprecated in | Replacement | Earliest removal |
|---|---|---|---|
| _none_ | | | |

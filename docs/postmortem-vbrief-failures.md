# Post-Mortem: Two Incorrect vBRIEF Files & Broken Renderer

**Date**: 2026-03-11
**Context**: Specification generation for DEFT Evolution via `PRD.md` interview process

---

## What Happened

During the specification interview for DEFT Evolution, I produced two incorrect `vbrief/specification.vbrief.json` files before getting the format right on the third attempt. The built-in `task spec:render` also failed to render the correct file. The entire sequence wasted significant time and required the user to intervene twice.

---

## Failure 1: Flat Decisions Dump (Not a vBRIEF at All)

### What I wrote

```json
{
  "name": "deft-evolution",
  "version": "1.0.0",
  "status": "draft",
  "decisions": { ... },
  "stack": { ... }
}
```

### Why it was wrong

- No `vBRIEFInfo` root key, no `plan` root key — missing both REQUIRED top-level fields per vBRIEF v0.5
- No `items` array, no `edges`, no `narratives`
- Used a completely invented schema (flat key-value decisions object)
- Was essentially a raw JSON config file, not a vBRIEF document

### Root cause

I never read the vBRIEF specification before writing the file. I saw the deft doc at `deft/vbrief/vbrief.md` which described the *lifecycle and conventions* of vBRIEF files within deft, but not the *format itself*. That doc assumes the reader already knows what a vBRIEF document looks like — it describes when and where to use them, not how to structure them.

I fabricated a schema based on:
- The field names mentioned in `deft/Taskfile.yml`'s `spec:render` task (which references `status`, `title`, `tasks`, `do`, `narrative`, `acceptance`)
- General assumptions about what a "specification JSON" should contain
- Zero actual knowledge of the vBRIEF v0.5 spec

---

## Failure 2: Deft-Renderer-Shaped JSON (Still Not vBRIEF)

### What I wrote

```json
{
  "title": "DEFT Evolution",
  "status": "approved",
  "overview": "...",
  "decisions": { ... },
  "stack": { ... },
  "tasks": [
    { "id": "1.0", "do": "Phase 1: Foundation", "status": "todo", ... },
    { "id": "1.1.1", "do": "Initialize monorepo", "status": "todo", ... }
  ]
}
```

### Why it was wrong

- Still no `vBRIEFInfo` or `plan` wrapper — invalid vBRIEF v0.5
- Used `do` instead of `title` on items (mimicking the `spec:render` Python script's field access)
- Used flat `tasks` array instead of hierarchical `items` with `subItems`
- Used `depends: [...]` arrays instead of `edges` with `from`/`to`/`type`
- Numeric IDs (`"1.1.1"`) instead of semantic dot-notation IDs (`"phase1.repo"`)

### Root cause

Same fundamental problem: I still hadn't read the vBRIEF spec. Instead, I reverse-engineered a format from the broken `spec:render` Python script in `deft/Taskfile.yml`. I shaped my JSON to match what the renderer expected — but the renderer itself was written against an older/incorrect schema, not vBRIEF v0.5. So I was conforming to a broken reference.

---

## The spec:render Task Is Broken

The `spec:render` task in `deft/Taskfile.yml` (lines 97–136) has multiple issues:

### Bug 1: Shell variables inside Python heredoc

```yaml
python3 - <<'EOF'
import json, sys
with open("$SPEC_FILE") as f:    # <-- $SPEC_FILE is a shell variable
```

The heredoc uses `<<'EOF'` (single-quoted delimiter), which **prevents shell variable expansion**. So `$SPEC_FILE` and `$OUT_FILE` are passed as literal strings `"$SPEC_FILE"` to Python, which will raise `FileNotFoundError`. The script can never actually open the file.

**Fix**: Either use `<<EOF` (unquoted) to allow shell expansion, or pass the path as a Python argument (`python3 - "$SPEC_FILE" "$OUT_FILE" <<'EOF'` and use `sys.argv`).

### Bug 2: Status check looks at wrong path

```bash
STATUS=$(python3 -c "import json; d=json.load(open('$SPEC_FILE')); print(d.get('status',''))")
```

This looks for `status` at the **root** of the JSON. In vBRIEF v0.5, status lives at `plan.status`. So even a valid vBRIEF file returns empty string, and the renderer refuses to run ("status is '' (expected 'approved')").

**Fix**: `d.get('plan', {}).get('status', '')` or `d['plan']['status']`.

### Bug 3: Renderer reads wrong schema

The Python script reads:
- `spec.get("title")` — should be `spec["plan"]["title"]`
- `spec.get("overview")` — should be `spec["plan"]["narratives"]["Overview"]`
- `spec.get("tasks", [])` — should be `spec["plan"]["items"]`
- `task.get("do")` — should be `item["title"]` (vBRIEF items use `title`, not `do`)

The renderer was written for a pre-v0.5 format (or an internal deft convention) that doesn't match the actual vBRIEF specification. It can't render any valid vBRIEF v0.5 document.

### Bug 4: No support for hierarchical items or edges

The renderer iterates `tasks` flat. vBRIEF v0.5 uses nested `subItems` and `edges` for DAG dependencies — the renderer ignores both entirely. Even if the field access bugs were fixed, it would produce a flat task list with no phase structure, no dependency information, and no narratives.

---

## Why I Failed Twice

### 1. No link to the actual vBRIEF spec from deft docs

`deft/vbrief/vbrief.md` describes lifecycle, conventions, and anti-patterns for vBRIEF files within deft. It never links to the vBRIEF specification itself (https://vbrief.org or the `vbrief-spec-0.5.md` document). It never shows a complete example of a valid vBRIEF document. It never defines `vBRIEFInfo`, `plan`, `items`, `edges`, or `narratives`.

An agent (or human) reading only the deft docs has no way to learn the actual format without external knowledge.

### 2. The renderer served as a de facto (wrong) schema

When I couldn't find a format definition, I reverse-engineered the expected format from `spec:render`'s Python code. The renderer became my "specification by example" — but it was written against an incompatible schema. This created a feedback loop: I shaped my JSON to match the broken renderer, producing invalid vBRIEF.

### 3. No validation against vBRIEF JSON Schema

The `spec:validate` task only checks that the file is valid JSON. It does not validate against the vBRIEF v0.5 JSON Schema (available at `schemas/` in the vBRIEF repo). Both of my incorrect files passed validation because they were syntactically valid JSON — just structurally wrong.

### 4. No example specification.vbrief.json in deft

The `deft/templates/` directory has `make-spec.md` (the interview workflow) and `make-spec-example.md` (an example prompt), but no example `specification.vbrief.json`. There's nothing showing "here's what the output should look like."

---

## Recommendations for Deft

### R1: Add a link to vBRIEF spec in `deft/vbrief/vbrief.md`

At the top of the file, add:

```markdown
**Format reference**: [vBRIEF Specification v0.5](https://vbrief.org) — all vBRIEF files
MUST conform to this specification.
```

This is the single highest-impact fix. An agent that reads the spec will produce correct files.

### R2: Add an example `specification.vbrief.json` to `deft/templates/`

Create `deft/templates/specification-example.vbrief.json` — a minimal but complete example showing the correct structure with `vBRIEFInfo`, `plan`, `items` with `subItems`, `edges`, and `narratives`. This serves as a "golden file" that agents and humans can copy.

### R3: Fix `spec:render` to handle vBRIEF v0.5

The renderer needs to:
- Fix the shell variable heredoc bug (use `<<EOF` or `sys.argv`)
- Read `plan.status` instead of root `status`
- Read `plan.title`, `plan.narratives.Overview`, `plan.items` instead of root-level fields
- Support nested `subItems` (recursive rendering)
- Support `edges` (render as dependency annotations or a dependency map section)
- Support `narratives` (render as sections: Overview, Background, Constraint, Risk, etc.)

Alternatively, replace the inline Python script with a proper renderer from the `libvbrief` Python package (`pip install libvbrief`), which presumably handles v0.5 correctly.

### R4: Add schema validation to `spec:validate`

Enhance the validation task to check against the vBRIEF v0.5 JSON Schema:

```bash
python3 -c "
from libvbrief import validate
validate('$SPEC_FILE')
"
```

Or download `schemas/vbrief-v0.5.schema.json` from the vBRIEF repo and validate with `jsonschema`. This catches structural errors (missing `vBRIEFInfo`, wrong `items` shape) before the renderer runs.

### R5: Add a MUST rule about reading vBRIEF spec before writing

In `deft/vbrief/vbrief.md`, add:

```
- ! Before writing ANY vBRIEF file, the agent MUST read the vBRIEF v0.5 specification
  or a validated example. ⊗ Infer the format from renderer code or field names alone.
```

This prevents the reverse-engineering-from-broken-renderer failure mode.

### R6: Add a smoke test for spec:render

Add a test fixture (`deft/tests/fixtures/example-spec.vbrief.json`) and a task that verifies `spec:render` can produce valid markdown from it. This catches renderer regressions.

---

## Summary

| Problem | Root Cause | Fix |
|---------|-----------|-----|
| Agent wrote wrong format twice | No link to vBRIEF spec, no example file | R1, R2, R5 |
| Agent shaped JSON to match broken renderer | Renderer was de facto schema reference | R3 |
| Invalid files passed validation | Validation only checks JSON syntax | R4 |
| spec:render can't render valid vBRIEF v0.5 | Renderer written for incompatible schema + shell bug | R3, R6 |

The core issue is a documentation gap: deft describes *when* and *where* to use vBRIEF files, but never links to *what* the format actually is. The broken renderer compounds this by serving as a misleading example. Fixing R1 (link to spec) and R3 (fix renderer) would prevent this entire class of failure.

# Agent Instructions

This project uses the **deft framework** for AI-assisted development.
You MUST read and follow these files before writing any code:

## Required Reading (in order)

1. `deft/main.md` — AI behavior rules and framework overview
2. `PROJECT.md` — Tech stack, strategy, and project-specific overrides
3. `deft/coding/coding.md` — Software development guidelines
4. `deft/tools/taskfile.md` — Task runner (use `task` for ALL build/test/lint operations)
5. `deft/scm/git.md` + `deft/scm/github.md` — Commit and PR conventions

## Language-Specific (read only when relevant)

- Go code: `deft/languages/go.md`
- TypeScript/Next.js: `deft/languages/typescript.md`

## Key Rules

- Use `task` for everything — never run `go test`, `npm test`, etc. directly
- Run `task check` and ensure it fully passes before creating any PR
- TDD: write tests first, watch fail, implement, refactor
- ≥85% test coverage required on all code
- Conventional commits format (https://www.conventionalcommits.org)
- File names use hyphens, not underscores
- Secrets go in `secrets/` with `.example` templates

## Project Specification

See `SPECIFICATION.md` for the full implementation plan.
The current phase to implement is specified in the GitHub Actions workflow prompt.

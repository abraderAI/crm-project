package seed

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/abraderAI/crm-project/api/internal/models"
	"github.com/abraderAI/crm-project/api/pkg/slug"
)

// forumThreadDef defines a seed forum thread.
type forumThreadDef struct {
	Title    string
	Body     string
	IsPinned bool
}

// forumSeedThreads lists the seed threads for the DEFT General Discussion forum.
// Threads cover the deftai/directive framework and the deftai/vBRIEF specification.
var forumSeedThreads = []forumThreadDef{
	{
		Title:    "Welcome to DEFT General Discussion!",
		IsPinned: true,
		Body: `Welcome to the official DEFT community forum! This is the place to ask questions, share tips, and discuss everything related to the Deft framework.

Whether you're just getting started or you've been using Deft across multiple projects, we'd love to hear from you. A few ground rules:

- Be kind and constructive
- Search before posting — someone may have already answered your question
- Share what you've learned — your experience helps everyone

Happy coding!`,
	},
	{
		Title: "Getting Started: AGENTS.md vs SKILL.md — which should I use?",
		Body: `I just cloned the directive repo and I'm a bit confused about the entry points. There's both an AGENTS.md approach and a SKILL.md approach mentioned in the README.

From what I understand:
- **AGENTS.md** is the simpler path — just add a line pointing to deft/main.md and you're set
- **SKILL.md** is the native Deft entry point for platforms that support it

For anyone getting started: if your AI platform supports SKILL.md, use that. Otherwise, AGENTS.md works great as a bridge. The README covers this under "Platform compatibility."

Has anyone found one approach works better than the other in practice?`,
	},
	{
		Title: "How the layer system works: USER.md → PROJECT.md → language files",
		Body: `One of the things that clicked for me after using Deft for a week is the layer system. Here's how precedence works (highest to lowest):

1. **USER.md** (~/.config/deft/USER.md) — your personal preferences, always wins
2. **PROJECT.md** — project-specific overrides
3. **Language files** (python.md, go.md, typescript.md) — language standards
4. **Taskfile guidelines** (taskfile.md) — tool conventions
5. **main.md** — general AI behavior

The key insight: you don't need to repeat yourself. Set your personal style once in USER.md, set project conventions in PROJECT.md, and the language files handle the rest.

The RFC 2119 notation (!, ~, ⊗, ≉) makes it scannable too — ! means MUST, ~ means SHOULD, ⊗ means MUST NOT.`,
	},
	{
		Title: "My favorite strategies: interview, yolo, and map",
		Body: `Deft's strategy system is incredibly powerful. Here are the three I use most:

**Interview** (/deft:run:interview) — Best for new features. It walks through a structured interview with a sizing gate that determines whether you get the Light or Full treatment. Great for making sure requirements are solid before writing code.

**Yolo** (/deft:run:yolo) — Same as interview but the AI picks all the options. Perfect when you trust the framework and want to move fast. I use this for smaller features where I know the pattern.

**Map** (/deft:run:map) — Essential for brownfield projects. It builds a mental model of the codebase before making changes. I always run this first when joining a new repo.

What strategies do you all use most? Has anyone built custom strategies?`,
	},
	{
		Title: "Taskfile workflows: why I stopped writing shell scripts",
		Body: `Before Deft, I had a collection of shell scripts for every project. Now everything goes through Task (taskfile.dev).

The key commands I use daily:
- **task check** — runs fmt + lint + test + coverage in one shot. This is the pre-commit gate.
- **task test:coverage** — ensures you stay above the coverage threshold (85% by default)
- **task dev** — starts the dev server

Pro tips:
- Use sources/generates for caching — Task skips steps when inputs haven't changed
- Split large Taskfiles: Taskfile.dev.yml, Taskfile.ci.yml, Taskfile.tools.yml
- Always add desc: to tasks so task --list is self-documenting

The Deft taskfile.md guide covers all of this. Highly recommend reading it if you haven't.`,
	},
	{
		Title: "vBRIEF plans and session continuity — how to not lose context",
		Body: `One problem I kept hitting with AI coding was losing context between sessions. Deft solves this with vBRIEF files in the ./vbrief/ directory.

The key files:
- **plan.vbrief.json** — your single active work plan (replaces scattered todo files)
- **continue.vbrief.json** — interruption recovery checkpoint
- **specification.vbrief.json** — project spec source of truth

The workflow: when you start a task, the plan tracks progress. If you get interrupted, a continue checkpoint saves your state. When you come back, /deft:continue picks up right where you left off.

Status values: draft → proposed → approved → pending → running → completed → blocked → cancelled

The biggest win for me: I stopped losing track of what I was doing mid-feature. The plan file is always there.`,
	},
	{
		Title: "Context management and lazy loading — keeping token budgets lean",
		Body: `Deft's lazy loading system (documented in REFERENCES.md) is something I didn't appreciate until I started hitting context limits.

Instead of loading every Deft file into the AI context, you only load what's relevant:
- Working on Go? Load go.md, not python.md or typescript.md
- Writing tests? Load testing.md
- Doing a deployment? Load the relevant deployment guide from deployments/

The RFC 2119 notation helps too — the compact symbols (!, ~, ⊗) pack more meaning into fewer tokens compared to writing out "MUST", "SHOULD", "MUST NOT" everywhere.

For larger projects, the context/ directory has advanced strategies: deterministic splits, fractal summaries, working memory patterns. Worth reading if you're managing complex codebases.

What are your strategies for keeping context efficient?`,
	},
	{
		Title: "What is vBRIEF and why should I care?",
		Body: `If you haven't looked at vBRIEF yet (github.com/deftai/vBRIEF), it's worth understanding even outside of Deft.

vBRIEF stands for Basic Relational Intent Exchange Format. It's a universal JSON format for structured thinking — todo lists, project plans, playbooks, specs, and even AI agent memory. One schema, graduated complexity.

The key insight: every AI agent invents its own memory format. Every planning tool has its own schema. vBRIEF is the common language. A minimal document is just 4 fields:

  vBRIEFInfo: { version: "0.5" }
  plan: { title, status, items: [{ title, status }] }

That's it. Everything else is optional — narratives, edges (DAG), tags, metadata. You add complexity only when you need it.

The "graduated complexity" design means you're never fighting boilerplate for simple tasks, but the format scales up to full DAG workflows when you need them.`,
	},
	{
		Title: "TRON encoding: 35-40% fewer tokens for LLM plans",
		Body: `One of the hidden gems in the vBRIEF spec is TRON encoding (documented in docs/tron-encoding.md). It's an alternative serialization that cuts LLM token usage by 35-40% compared to standard JSON.

The idea: LLMs don't need pretty-printed JSON with redundant keys. TRON strips the format down to the minimum needed for accurate round-tripping while staying human-readable.

This matters a lot when you're working with large plans inside context windows. A 200-item plan in JSON might eat 3000 tokens. In TRON, it's closer to 1800. That's context budget you can spend on actual code.

Anyone benchmarked TRON vs JSON for their workflows? I'm seeing consistent 37% savings on my project plans.`,
	},
	{
		Title: "Graduated complexity in practice: from todo list to DAG workflow",
		Body: `The vBRIEF spec defines four complexity levels. Here's how I actually use them:

**Minimal** — Quick task tracking. I use this for daily standup notes and quick bug lists. Just title + status + items. Takes 30 seconds to write.

**Structured** — When I need to explain WHY something is happening. Narratives on items add context like "Problem: API timeout under load" or "Outcome: Reduced p95 latency from 2s to 200ms". Great for handoffs.

**Retrospective** — After a sprint or feature delivery. Captures outcomes, strengths, weaknesses, and lessons learned. I use the retrospective-plan.vbrief.json example as a template.

**Graph/DAG** — For complex features with dependencies. The edges array lets you model "task B depends on task A" relationships. I used this for a database migration that had 12 interdependent steps.

The beauty is you can start minimal and graduate up. My plans often start as flat lists and gain narratives as I learn things worth documenting.`,
	},
	{
		Title: "Using the Python library (libvbrief) for plan automation",
		Body: `Just discovered that vBRIEF ships a Python library: pip install libvbrief

This is useful for automating plan management in CI/CD or custom tooling. A few things I've built with it:

1. A pre-commit hook that validates all .vbrief.json files against the JSON schema
2. A script that extracts "blocked" items from plan.vbrief.json and posts them to Slack
3. A GitHub Action that checks if the plan status matches the PR state

The validation is especially handy — the JSON Schema in schemas/ catches issues like invalid status values or missing required fields before they cause problems downstream.

The library handles both JSON and TRON serialization, so you can read TRON files and write JSON (or vice versa) without manual parsing.`,
	},
	{
		Title: "How I use plan.vbrief.json as my single source of truth",
		Body: `The vBRIEF spec says there should be exactly ONE plan.vbrief.json per project. At first I thought this was limiting, but it's actually the point.

Before vBRIEF, I had TODO.md, PLAN.md, a Notion board, and random notes scattered across files. Now everything is in plan.vbrief.json:

- Current sprint items with status tracking
- Blocked items with narrative explanations
- Completed items preserved for context (don't delete — mark cancelled or completed)

The status lifecycle is clean: draft → proposed → approved → pending → running → completed/blocked/cancelled

When I start a new coding session, the AI reads plan.vbrief.json and immediately knows what's in progress, what's blocked, and what's next. No more "where was I?" moments.

Tip: use the specification.vbrief.json for project requirements and keep plan.vbrief.json for execution tracking. They serve different purposes.`,
	},
	{
		Title: "DAG plans: modeling task dependencies with edges",
		Body: `The DAG (Directed Acyclic Graph) feature in vBRIEF v0.5 is powerful for complex workflows. Instead of a flat list, you define edges between items:

  edges: [
    { from: "design-api", to: "implement-api" },
    { from: "implement-api", to: "write-tests" },
    { from: "write-tests", to: "deploy" }
  ]

This tells the AI (and any tooling) that implement-api can't start until design-api is complete. The examples/dag-plan.vbrief.json file has a full example.

I've been using this for:
- Database migrations with ordering constraints
- Multi-service deployments where services depend on each other
- Feature rollouts with feature-flag gating

The validator catches cycles too — if you accidentally create A → B → C → A, it fails validation. This has saved me from some gnarly circular dependency bugs in deployment plans.`,
	},
	{
		Title: "Migrating from v0.4 to v0.5: what changed and why",
		Body: `If you're coming from vBRIEF v0.4, the migration guide (MIGRATION.md) covers everything, but here are the highlights:

Key changes in v0.5:
- The vBRIEFInfo envelope is now required (was optional in v0.4)
- Status enum is formalized: exactly 8 values (draft, proposed, approved, pending, running, completed, blocked, cancelled)
- TRON encoding is now part of the spec (was experimental)
- DAG edges use a simpler format: { from, to } instead of the old dependency arrays
- Narratives use free-form key-value maps instead of fixed fields

The biggest philosophical shift: v0.5 treats the plan as a living document that evolves during execution. The status lifecycle reflects this — items move through states rather than being static checkboxes.

Migration is straightforward: update the version field, ensure status values match the new enum, and restructure any dependency arrays into edges. The validator catches anything you miss.`,
	},
}

// seedForumThreads creates seed threads in the global-forum space.
// It is idempotent: threads with matching slugs are skipped.
func seedForumThreads(db *gorm.DB, forumBoardID string) error {
	systemAuthorID := "system-seed"
	baseTime := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)

	for i, def := range forumSeedThreads {
		threadSlug := slug.Generate(def.Title)

		// Skip if already seeded.
		var count int64
		if err := db.Model(&models.Thread{}).
			Where("board_id = ? AND slug = ?", forumBoardID, threadSlug).
			Count(&count).Error; err != nil {
			return fmt.Errorf("checking forum seed thread %q: %w", def.Title, err)
		}
		if count > 0 {
			continue
		}

		thread := models.Thread{
			BoardID:    forumBoardID,
			Title:      def.Title,
			Body:       def.Body,
			Slug:       threadSlug,
			Metadata:   "{}",
			AuthorID:   systemAuthorID,
			IsPinned:   def.IsPinned,
			ThreadType: models.ThreadTypeForum,
			Visibility: models.ThreadVisibilityPublic,
		}
		thread.CreatedAt = baseTime.Add(time.Duration(i) * time.Hour)
		thread.UpdatedAt = thread.CreatedAt

		if err := db.Create(&thread).Error; err != nil {
			return fmt.Errorf("seeding forum thread %q: %w", def.Title, err)
		}
	}
	return nil
}

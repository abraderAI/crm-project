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
// All threads are themed around setting up and using the deftai/directive repo.
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

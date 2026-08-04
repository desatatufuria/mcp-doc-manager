# DocManager OpenCode Guidance

Version: v1

Use DocManager only for an explicit documentation-impact review. Select exactly one scope: `worktree` (including unstaged changes), `staged` (index changes), or a two-dot `base..head` range. Make one `document_change` call for that selected scope.

Review the `document_change` result with a human review before any receipt action. After review, make at most one `verify_receipt` call only when the selected scope and considered documentation bytes are unchanged. If either changed, select the scope again and run a new analysis.

Do not call DocManager for routine coding, unrelated questions, inferred scope or ambiguous scope, automatic editing, or unavailable evidence. DocManager is read-only and MUST NOT mutate documentation, user instructions, or `AGENTS.md`.

This guidance MUST NOT manage MCP lifecycle, workspace or hook behavior, auto-enable hooks, change other agents, perform generic agent or release composition, or use historical installer-plan evidence. It describes a review workflow only.

# Proposal: Living Documentation Lifecycle

## Intent

Evolve DocManager from a checker-only end-of-change workflow into the lifecycle orchestrator, catalog, and evidence authority for living repository documentation. Preserve visible, reviewable docs and human approval while making plans, change impact, freshness, and outcomes traceable.

## Scope

### In Scope
- Read-only repository radiography covering purpose, stack, modules, architecture, flows, data, execution, tests, deployment, risks, and existing docs; surface uncertainty and prioritize depth by value, risk, complexity, and change frequency.
- User-approved information architecture, visible storage location, audiences/language, ownership candidates from CODEOWNERS/Git/structure, and policy (`approval-required` or `automatic-after-approved-plan`), including audited policy changes.
- `.docmanager/` policy/catalog/provenance state; import existing Markdown docs in place with required catalog fields.
- Change-impact and on-demand freshness audits; batch approvals by feature/change; evidence-marked orphans require approval before archival/removal.
- Verification of caller-authored approved edits and lifecycle outcome recording.

### Out of Scope
- Direct DocManager writes to visible documentation, source, or user instructions.
- Default relocation/duplication of existing docs, autonomous structural/destructive operations, truthfulness scoring, and cross-repository catalogs.
- Incorporating or changing `opencode-daily-docmanager-guidance`.

## Capabilities

### New Capabilities
- `repository-radiography`: Discover repository/documentation evidence, exclusions, audiences, ownership candidates, and uncertainty.
- `documentation-lifecycle-planning`: Propose and approve information architecture, visible storage, policy, and coherent change batches.
- `documentation-catalog-lifecycle`: Persist internal catalog/policy/provenance; evaluate change impact and freshness; verify approved caller-authored outcomes.

### Modified Capabilities
- None; no existing main specs are present.

## Approach

Add staged MCP lifecycle contracts over reusable Git validation, evidence, SQLite, receipt, MCP-stdio, and safe OpenCode configuration foundations. Keep human-facing files in an approved discovered location; keep lifecycle state under `.docmanager/`. The coding agent authors approved edits through normal repository changes; DocManager validates evidence and records results. Automatic policy authorizes content create/update only within an approved plan; structural or destructive actions always need approval. Current checker-only guidance is not the target product.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/adapters/mcp/mcp.go` | Modified | Staged lifecycle MCP surface. |
| `internal/app/` | Modified | Radiography, planning, catalog, audit, and verification use cases. |
| `internal/adapters/sqlite/` | Modified | Versioned `.docmanager/` catalog and policy state. |
| `internal/adapters/git/`, `internal/domain/` | Modified | Discovery and evidence inputs. |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Discovery misclassifies docs | Medium | Evidence, exclusions, and explicit uncertainty. |
| Automatic mode overwrites assets | Medium | Approved-plan boundary; approval for structural/destructive actions. |
| MVP exceeds review budget | Medium | Ask-on-risk delivery decision before apply; split by lifecycle capability. |

## Rollback Plan

Remove new MCP lifecycle operations and catalog migrations; retain existing receipt/evidence primitives. Restore prior visible docs from normal Git history; never auto-delete imported docs.

## Dependencies

- User approval of radiography/plan before lifecycle persistence or authoring.

## Success Criteria

- [ ] A repository can produce and approve a complete radiography and visible-doc plan before authoring.
- [ ] Catalog entries store path, purpose, audience, owner, related areas, lifecycle state, evidence, last verification, and pending actions.
- [ ] Audits report evidenced impacts/orphans; destructive outcomes require approval.
- [ ] Approved caller edits are verified and recorded without visible provenance markers.

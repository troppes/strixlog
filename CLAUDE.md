# Claude Instructions

## Code Conventions

All commits use semantic versioning as well as the ticketnumber. As an example: fix: [TASK-X] COMMIT-TEXT.

Furthermoore all tasks are worked inside of short-lived branched, e.g. trunk-based development. You can never change anything on the main-branch, a branch is always needed. The branch names have also prefixes like: chore,feat or fix

Always modify the backlog before commiting so no filechanges remain after you are finished.

The goal is to use only libraries when its necessary, if only a small part of a lib is needed, recreate the function.

## Simplicity First

Do not introduce infrastructure, abstractions, or configuration for requirements that do not exist yet. Add complexity only when it is concretely needed. Examples:
- Do not add Docker networks, volumes, or `depends_on` unless a service actually depends on another at runtime
- Do not add shared libraries between services until duplication becomes a real maintenance problem
- Do not add error handling, retries, or fallbacks for scenarios that cannot happen yet

<!-- BACKLOG.MD MCP GUIDELINES START -->

<CRITICAL_INSTRUCTION>

## BACKLOG WORKFLOW INSTRUCTIONS

This project uses Backlog.md MCP for all task and project management activities.

### CRITICAL GUIDANCE

- If your client supports MCP resources, read `backlog://workflow/overview` to understand when and how to use Backlog for this project.
- If your client only supports tools or the above request fails, call `backlog.get_workflow_overview()` tool to load the tool-oriented overview (it lists the matching guide tools).

- **First time working here?** Read the overview resource IMMEDIATELY to learn the workflow
- **Already familiar?** You should have the overview cached ("## Backlog.md Overview (MCP)")
- **When to read it**: BEFORE creating tasks, or when you're unsure whether to track work

These guides cover:

- Decision framework for when to create tasks
- Search-first workflow to avoid duplicates
- Links to detailed guides for task creation, execution, and finalization
- MCP tools reference

You MUST read the overview resource to understand the complete workflow. The information is NOT summarized here.

</CRITICAL_INSTRUCTION>

<!-- BACKLOG.MD MCP GUIDELINES END -->

---
name: bulk-address-github-review-comments
description: Process multiple unresolved GitHub Pull Request (PR) review threads as a reviewed queue with upfront user confirmation, sequential replies, commit, and push. Use ONLY when the task is to address review comments on an existing GitHub pull request from the checked-out feature branch or from a PR URL provided in the prompt.
compatibility: Requires a local git checkout, network access, and GitHub access through GitHub MCP tools preferred or authenticated gh CLI fallback.
metadata:
  author: Benizzio with OpenCode
  maturity: beta
  version: 0.0.0
  scope: project-local
---

# Bulk Address GitHub Pull Request Review Comments

## Use This Skill For

- Addressing unresolved review threads on an existing GitHub pull request.
- Making code changes, running tests, committing, pushing, and replying to each unresolved thread in order.

## Do Not Use This Skill For

- General code review.
- Resolving threads without implementing and verifying code changes.

## Required Capabilities

- Prefer GitHub MCP tools for Pull Request lookup, unresolved review-thread reads, and replies.
  - Examples in environments that expose them: `github_search_pull_requests`, `github_pull_request_read`, `github_add_reply_to_pull_request_comment`.
- If GitHub MCP tools are unavailable, use authenticated `gh` commands.
- In `gh` fallback mode, use thread-aware API calls. `gh pr view --comments` alone is not enough because it does not reliably expose unresolved review-thread state.
- Stop with `🚫 [UNFULFILLABLE]` if neither GitHub MCP nor authenticated `gh` access is available.

## Determine The Pull Request

1. If the prompt includes a Pull Request URL, use it.
2. Otherwise derive the Pull Request from the checked-out local branch:
   - inspect the local git remote to identify the GitHub repository
   - inspect the checked-out branch name
   - map that branch to its open pull request in the same repository
3. If the current branch does not map to exactly one pull request, stop and ask the user for the Pull Request URL.
4. Do not guess between multiple candidate pull requests.
5. If the derived pull request and a supplied Pull Request URL disagree, stop and ask the user which pull request should be used.
6. If the pull request cannot be derived from the currently checked-out local branch, stop and ask the user for the Pull Request URL.

## Collect Unresolved Review Threads

1. Read only review conversations that are still unresolved.
2. Read the entire conversation for each unresolved review thread:
   - original review comment
   - all replies
   - file path and line context
   - author and bot context when present
   - outdated and resolved state
3. Prefer review-thread APIs over plain pull request comments because unresolved state belongs to the thread.
4. Build a working queue in stable order:
   - first choice: the unresolved thread order returned by GitHub
   - fallback: oldest unresolved thread first
5. Before editing, read enough local code to understand the request and detect overlap with other unresolved threads.

## Plan Atomic Work Units

After collecting the unresolved review threads and before editing code, convert the thread queue into atomic work units.

The GitHub-visible process stays thread-by-thread. Atomic work units only change how local implementation work is delegated to sub-agents with clean context.

1. Keep the original unresolved review-thread queue as the authoritative reply order.
2. Group one or more unresolved review threads into the smallest coherent units of implementation work.
3. Prefer one thread per unit unless multiple threads require the same code change or have direct dependency overlap.
4. Group threads together when separating them would create duplicate edits, conflicting edits, or misleading partial fixes.
5. Keep dependent work in earlier units and downstream cleanup or follow-up work in later units.
6. Avoid units that span unrelated files, unrelated behavior, or unrelated test scopes.
7. Record for each unit:
   - unit identifier
   - included review thread identifiers
   - original reply order positions for those threads
   - files and behavior expected to be touched
   - known dependencies on earlier or later units
   - expected verification scope
   - whether the unit is expected to fully satisfy one or more later review threads
8. If the threads cannot be grouped without unresolved conflicts or ambiguity, stop and ask the user for instructions.

## Sub-Agent Work Dynamics

These rules are **MANDATORY** for the main agent session. Repeat and preserve them in the main session state before each sub-agent handoff and after any context compaction.

1. The main agent is the orchestrator and remains responsible for correctness, verification, commits, pushes, and GitHub replies.
2. Sub-agents receive clean-context implementation handoffs for exactly one atomic work unit at a time.
3. Sub-agents must not post GitHub replies, resolve threads, create commits, push branches, or change the work-unit plan unless the main agent explicitly instructs otherwise.
4. Sub-agents may edit files and run local verification for their assigned unit when the handoff authorizes it.
5. The main agent must inspect the resulting diff after each sub-agent returns.
6. The main agent must verify that the returned work addresses the assigned review thread requirements and does not break the original thread queue, dependency plan, or existing code style.
7. The main agent must run or review credible verification evidence before any commit, push, or GitHub reply.
8. The next sub-agent must not start until the main agent has accepted or corrected the previous unit's work.
9. If sub-agent output is incomplete, conflicting, unverifiable, or broader than the handoff allowed, the main agent must fix it locally or stop and ask the user.

Maintain a visible work ledger in the main session while using this skill:

1. List all atomic work units and their thread coverage.
2. Mark exactly one unit as in progress.
3. After each sub-agent returns, record:
   - files changed
   - review threads satisfied or partially satisfied
   - verification run or still needed
   - whether the main agent accepted, corrected, or rejected the result
4. Before continuing after compaction or a long interruption, restate the ledger and the sub-agent dynamics above.

## Sub-Agent Handoff Requirements

Each sub-agent handoff must be clear enough for a clean-context agent to work without reading the main conversation.

Include all of the following in the handoff:

1. Pull Request repository, number, branch, and base branch when known.
2. The atomic work-unit identifier.
3. The included unresolved review thread identifiers and their original queue positions.
4. The full text of the relevant review comments and replies, with author context when useful.
5. Referenced file paths, line context, and any nearby code context already read by the main agent.
6. The intended behavior change and non-goals.
7. Dependencies on earlier work units and constraints needed to avoid conflicts with later units.
8. The exact files or package areas the sub-agent is expected to inspect or modify.
9. The required verification scope, including preferred narrow tests and when wider tests are required.
10. Explicit prohibitions against GitHub replies, thread resolution, commits, pushes, broad refactors, and unrelated cleanup.
11. The expected final report format:
    - summary of changes
    - files changed
    - tests or checks run with results
    - review threads believed to be fully addressed
    - review threads still needing main-agent attention

## Review And Confirm Before Proceeding

Do not edit code, commit, push, or reply to any review thread until this confirmation gate is complete.

1. Present the unresolved thread queue and proposed atomic work-unit plan to the user.
2. Inform the user that they must review the thread comments before the process continues.
3. Ask the user to confirm that they have reviewed the thread comments and have a concrete conclusion for what needs to be done.
4. Do not require agent-authored conclusions as part of this gate.
5. Proceed only after explicit user confirmation.
6. If the user does not confirm, stop without editing, committing, pushing, or replying.

## Sequential Execution Contract

Process exactly one atomic work unit at a time while preserving the original unresolved review-thread reply order.

The implementation work for one atomic unit may address several review threads. GitHub replies still happen one thread at a time, only when each thread reaches its original turn in the queue.

Before each unit, restate the active work ledger and the rule that implementation is delegated to exactly one clean-context sub-agent, then verified by the main agent before any commit, push, or GitHub reply.

For the current work unit:

1. Re-read all full threads included in the unit before editing or delegating.
2. Re-read the referenced code and any nearby code needed to understand the request.
3. Check other unresolved threads for overlapping files or behavior. Keep their requirements in mind, but do not reply to them yet.
4. Prepare a full sub-agent handoff that satisfies the Sub-Agent Handoff Requirements.
5. Delegate the current unit to one sub-agent with clean context.
6. After the sub-agent returns, inspect the local diff and its report.
7. Apply any main-agent corrections needed to keep the change small, consistent, and complete.
8. Verify the accepted change with the proper local test scope.
   - Use repository-provided test or coverage entrypoints when they exist.
   - Start with the narrowest sufficient test scope.
   - Widen the scope when the change affects shared behavior or when the narrow scope is not credible evidence.
9. If the change cannot be verified, do not reply in GitHub. Stop and report the blocker.

For each review thread in the original queue whose requirements are now satisfied by accepted and verified work:

1. Re-read the full thread before replying.
2. Confirm the current code state and verification evidence still address that thread.
3. If the thread requires code changes and the accepted work has not yet been committed:
   - stage only the related modified files
   - commit with `git commit -m "Addressing review comment"`
   - push the current branch
4. If an earlier sequential change already fully addressed this thread:
   - do not create an empty commit
   - still confirm the current code state and verification evidence before replying
5. After the push for the current thread, reply only to that thread.
   - summarize the solution
   - mention the verification that was run
   - reference other related review threads when that context matters
6. Do not resolve the thread.
7. Move to the next unresolved thread reply only after the current thread has been verified, committed when needed, pushed, and replied to.

If a work unit satisfies a later review thread before its reply turn, record that in the work ledger. Do not reply to that later thread until its turn in the original queue.

When an atomic unit contains exactly one review thread, the same process reduces to the original one-thread flow with sub-agent delegation added only for the local implementation step:

For the current thread:

1. Re-read the full thread before delegation.
2. Read the referenced code and any nearby code needed to understand the request.
3. Check other unresolved threads for overlapping files or behavior. Keep their requirements in mind, but do not reply to them yet.
4. Delegate the smallest correct patch for the current thread to one clean-context sub-agent.
5. Inspect the returned diff and apply any main-agent corrections needed to keep the code consistent with related unresolved feedback.
6. Verify the change with the proper local test scope.
   - Use repository-provided test or coverage entrypoints when they exist.
   - Start with the narrowest sufficient test scope.
   - Widen the scope when the change affects shared behavior or when the narrow scope is not credible evidence.
7. If the change cannot be verified, do not reply in GitHub. Stop and report the blocker.
8. If the current thread requires code changes:
   - stage only the related modified files
   - commit with `git commit -m "Addressing review comment"`
   - push the current branch
9. If an earlier sequential change already fully addressed this thread:
   - do not create an empty commit
   - still confirm the current code state and verification evidence before replying
10. After the push for the current thread, reply only to that thread.
   - summarize the solution
   - mention the verification that was run
   - reference other related review threads when that context matters
11. Do not resolve the thread.
12. Move to the next unresolved thread only after the current thread has been fixed, verified, committed when needed, pushed, and replied to.

## Strict Sequencing Rules

- Never run multiple sub-agent work units in parallel.
- Never reply to multiple review threads in one batch.
- Never post replies for later threads before the current thread has been fixed, verified, committed when needed, pushed, and replied to.
- If one coherent atomic work unit satisfies several unresolved threads, make that change in the earliest affected unit, but still reply to each thread only when its turn arrives.
- Do not collapse several review threads into one shared reply.
- Do not auto-resolve any thread.
- Keep this sequence to avoid review-bot rate-limit spikes and to preserve a clear audit trail.

## Stop Conditions

Stop and ask the user for instructions when:

- the Pull Request URL is not in the prompt and the current branch cannot be mapped to exactly one pull request
- unresolved review threads conflict with each other
- a requested change is unsafe, out of scope, or not feasible from the checked-out branch
- required GitHub access, push access, or local test execution is unavailable
- the next required step would need an empty commit or a misleading reply

## Completion

After the last unresolved review thread has been processed:

- report that the code changes are done
- report that replies were posted in sequence
- mention that the review threads were intentionally left unresolved

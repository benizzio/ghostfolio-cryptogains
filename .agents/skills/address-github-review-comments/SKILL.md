---
name: address-github-review-comments
description: GitHub PR review comments, unresolved conversations, sequential replies, commit, and push. Use ONLY when the task is to address review comments on an existing GitHub pull request from the checked-out feature branch or from a PR URL provided in the prompt.
compatibility: Requires a local git checkout, network access, and GitHub access through GitHub MCP tools preferred or authenticated gh CLI fallback.
metadata:
  author: Benizzio with OpenCode
  maturity: beta
  version: 0.0.0
  scope: project-local
---

# Address GitHub PR Review Comments

## Use This Skill For

- Addressing unresolved review threads on an existing GitHub pull request.
- Making code changes, running tests, committing, pushing, and replying to each unresolved thread in order.

## Do Not Use This Skill For

- General code review.
- Resolving threads without implementing and verifying code changes.
- Guessing which pull request to use when branch-to-PR mapping is ambiguous.

## Required Capabilities

- Prefer GitHub MCP tools for PR lookup, unresolved review-thread reads, and replies.
  - Examples in environments that expose them: `github_search_pull_requests`, `github_pull_request_read`, `github_add_reply_to_pull_request_comment`.
- If GitHub MCP tools are unavailable, use authenticated `gh` commands.
- In `gh` fallback mode, use thread-aware API calls. `gh pr view --comments` alone is not enough because it does not reliably expose unresolved review-thread state.
- Stop with `🚫 [UNFULFILLABLE]` if neither GitHub MCP nor authenticated `gh` access is available.

## Determine The Pull Request

1. If the prompt includes a PR URL, use it.
2. Otherwise derive the PR from the checked-out local branch:
   - inspect the local git remote to identify the GitHub repository
   - inspect the checked-out branch name
   - map that branch to its open pull request in the same repository
3. If the current branch does not map to exactly one pull request, stop and ask the user for the PR URL.
4. Do not guess between multiple candidate pull requests.
5. If the derived pull request and a supplied PR URL disagree, stop and ask the user which pull request should be used.
6. If the pull request cannot be derived from the currently checked-out local branch, stop and ask the user for the PR URL.

## Collect The Unresolved Review Work

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

## Sequential Execution Contract

Process exactly one unresolved review thread at a time.

For the current thread:

1. Re-read the full thread before editing.
2. Read the referenced code and any nearby code needed to understand the request.
3. Check other unresolved threads for overlapping files or behavior. Keep their requirements in mind, but do not reply to them yet.
4. Apply the smallest correct patch that addresses the current thread while keeping the code consistent with related unresolved feedback.
5. Verify the change with the proper local test scope.
   - Use repository-provided test or coverage entrypoints when they exist.
   - Start with the narrowest sufficient test scope.
   - Widen the scope when the change affects shared behavior or when the narrow scope is not credible evidence.
6. If the change cannot be verified, do not reply in GitHub. Stop and report the blocker.
7. If the current thread requires code changes:
   - stage only the related modified files
   - commit with `git commit -m "Adressing review comment"`
   - push the current branch
8. If an earlier sequential change already fully addressed this thread:
   - do not create an empty commit
   - still confirm the current code state and verification evidence before replying
9. After the push for the current thread, reply only to that thread.
   - summarize the solution
   - mention the verification that was run
   - reference other related review threads when that context matters
10. Do not resolve the thread.
11. Move to the next unresolved thread only after the current thread has been fixed, verified, committed when needed, pushed, and replied to.

## Strict Sequencing Rules

- Never reply to multiple review threads in one batch.
- Never post replies for later threads before the current thread has been fixed, verified, committed when needed, pushed, and replied to.
- If one coherent code change satisfies several unresolved threads, make that change when the first affected thread is being processed, but still reply to each thread only when its turn arrives.
- Do not collapse several review threads into one shared reply.
- Do not auto-resolve any thread.
- Keep this sequence to avoid review-bot rate-limit spikes and to preserve a clear audit trail.

## Stop Conditions

Stop and ask the user for instructions when:

- the PR URL is not in the prompt and the current branch cannot be mapped to exactly one pull request
- unresolved review threads conflict with each other
- a requested change is unsafe, out of scope, or not feasible from the checked-out branch
- required GitHub access, push access, or local test execution is unavailable
- the next required step would need an empty commit or a misleading reply

## Completion

After the last unresolved review thread has been processed:

- report that the code changes are done
- report that replies were posted in sequence
- mention that the review threads were intentionally left unresolved

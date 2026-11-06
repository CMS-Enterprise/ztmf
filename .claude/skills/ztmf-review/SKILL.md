---
name: ztmf-review
description: Review a pull request in CMS-Enterprise/ztmf. Reads the PR, its ticket, and the full diff, runs the tests when the change touches behavior, and drafts one GitHub review that opens with a plain-English TL;DR and leaves out nits. Shows the draft to the person running it and posts nothing until they say go. Use when asked to review a ztmf PR ("review PR 634", "take a look at ztmf!634", "/ztmf-review 634").
---

# Reviewing a ztmf PR

People read these reviews, usually between other work. A review that takes five minutes to parse, or buries one real problem under six style notes, does not get read. Every rule below serves two goals: say only what matters, and say it so a person gets it on the first read.

Nothing is posted to GitHub until the person running this skill has seen the exact text and said go.

## Step 1: read before judging

1. `gh pr view <n> --repo CMS-Enterprise/ztmf --json title,body,author,headRefName,baseRefName,headRefOid,files,reviews,comments` for the PR body and any existing reviews. Inline review comments are not in that output, so also read `gh api repos/CMS-Enterprise/ztmf/pulls/<n>/comments`. Do not repeat a point another reviewer already made.
2. Run `gh pr checks <n> --repo CMS-Enterprise/ztmf` for CI. A red check is often the first finding, and it decides whether the review can approve (see Step 4).
3. Read the linked ticket and its comments. Tickets usually live in `CMS-Enterprise/ztmf-misc`. The comment thread often holds constraints the PR body leaves out.
4. Read the whole diff with `gh pr diff <n> --repo CMS-Enterprise/ztmf`, then open the surrounding code for every changed function. A hunk that looks wrong alone is often right in context.
5. If the person running this skill already reviewed this PR, this is a re-review. Read the commits pushed since their last review, and check each of their earlier findings: fixed, answered, or still open. Raise only what is still open or new. Do not post a finding again that the author already answered, unless the answer is wrong.
6. When the PR changes a handler, a query, a migration, scoring, an export, or a Terraform root, run the tests. Check the branch out in its own worktree (`git worktree add <dir> <branch>`) so the person's own checkout and any work in it stay untouched, then run from the repo root of that worktree: `make test-unit` and `make test-integration`, plus `make test-e2e` when an API response changes. Do not run anything that tears down or rebuilds a shared dev stack. Remove the worktree when done.
7. Before saying the PR causes a problem, check that the problem is not already on `main`. A wrong claim costs the author more time than a missed one.

## Step 2: apply the bar

A finding goes in the review only if it is one of these:

- A bug: wrong result, crash, data loss, broken migration, or a security or authorization gap.
- A behavior change the PR body does not mention.
- An API response or request change without a matching update to `backend/emberfall_tests.yml` and a regenerated `backend/openapi.yaml`.
- A migration in `backend/cmd/api/internal/migrations` whose number sorts at or before the newest one on `main`. CI does not catch this.
- A missing test for the exact bug the PR says it fixes.
- A pattern others will copy: a hardcoded value that belongs in config, a temporary hack with no ticket, an optional field modeled as non-nullable, an issue or PR number written into code or config.
- A question the reviewer needs answered before they can approve.

For anything else, ask: if this merges as it is, will something break, will someone be misled, or will it get copied into new code? If the answer is no, leave it out.

These always stay out:

- Naming, formatting, import order, whitespace, comment wording.
- "I would have written it differently" when both versions work.
- Anything a linter or formatter catches or could catch.
- Hypothetical future needs the ticket does not ask for.
- Praise for the parts that are fine.

There is no "nit:" comment. If a point is not worth fixing before merge, it is not worth posting. An approval with no comments is a normal, good outcome.

Fewer comments get more attention. If two findings share a cause, write one comment. If you have more than five, look again: some are probably the same problem, and some probably do not meet the bar.

## Step 3: write it

The review body always opens with a TL;DR:

```
**TL;DR:** Approve. Nothing blocking.
```

```
**TL;DR:** Request changes. The new migration is numbered 0068, which sorts before 0070 on main, so databases that already ran 0070 will skip it. Everything else looks right.
```

The TL;DR gives the verdict (approve, request changes, or comment), then one to three short sentences on what matters, in plain English. Someone who reads only the TL;DR should know whether the PR can merge and what has to change. Name things by what they do, not by function or file names, unless the name is the point.

After the TL;DR, the body has nothing else unless a finding cannot be tied to a line. Findings go in inline comments on the line they are about.

GitHub accepts an inline comment only on a line inside the diff: an added line, or an unchanged line shown in a hunk. One comment on any other line makes GitHub reject the whole review. A finding about code the PR did not touch, such as a caller it should have updated, goes in the body after the TL;DR, with the file and line written out.

Each inline comment:

- Says what is wrong, why it matters, and what would fix it, in that order, then stops.
- Is one idea. Two problems are two comments, unless they share a cause.
- Shows code when code is faster than prose. Use a `suggestion` block when the fix is a direct line replacement.
- Uses plain paragraphs. No headers, bullet lists, tables, or bold inside a comment.

Plain-English rules for everything posted:

- Short sentences. Common words. One idea per sentence.
- No warm-up ("Great work here!"), no restating what the diff does, no closing summary.
- No hedging filler: "just", "maybe worth considering", "might be nice to". If you are unsure, say so once: "I'm not sure about this, can you check?"
- Do not use these words and phrases, which make a review read as machine-written: robust, leverage, delve, seamless, crucial, crucially, essentially, nuanced, comprehensive, straightforward, elegant, gracefully, load-bearing, footgun, surface area, guardrail, orthogonal, "it's worth noting", "that said", "to be clear", "the key insight", "at the end of the day".

Before showing the draft, read it twice. First, check for banned words and any list or header formatting in a comment. Second, delete every phrase whose removal loses no information.

## Step 4: show the draft, then post

Show the person running this skill, in one message:

1. The full review: the TL;DR body, then each inline comment with its file and line, exactly as it will post.
2. A short "Left out" list: each finding dropped under the bar, one line each, so they can pull one back. This list is never posted.

Then wait. If they ask for edits, show the revised text and wait again.

On approval, post everything as a single review so the author gets one notification. Write the payload to a file:

```json
{
  "commit_id": "<headRefOid from Step 1>",
  "event": "APPROVE",
  "body": "**TL;DR:** ...",
  "comments": [
    { "path": "backend/internal/...", "line": 42, "side": "RIGHT", "body": "..." }
  ]
}
```

`commit_id` is the head commit you reviewed, so the comments stay on the lines you read even if the author pushes while you write. If the head has moved since Step 1, read the new commits before posting. A comment that spans lines adds `"start_line"` and `"start_side"`.

`event` is `APPROVE`, `REQUEST_CHANGES`, or `COMMENT`. GitHub does not let anyone approve or request changes on their own PR, so when the person running this skill is the author, use `COMMENT` and keep the verdict in the TL;DR. Then:

```bash
gh api -X POST repos/CMS-Enterprise/ztmf/pulls/<n>/reviews --input review.json
```

Do not approve a PR whose CI checks are red or still running, or whose local test run failed. Report what failed or is still pending, with the output, and let the person decide what to post.

After posting, give the review URL.

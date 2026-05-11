# Project Copilot Instructions

These rules apply to all coding tasks in this repository.

## Change policy
- Prefer minimal, targeted edits.
- Do not refactor unrelated code unless explicitly requested.
- Preserve existing behavior unless the task requires a behavior change.

## Safety policy
- Do not run destructive git commands unless explicitly requested.
- Avoid changing build, deployment, or secrets files unless required by the task.

## Go coding policy
- Keep code compile-safe after each change.
- Return early on errors and keep error context clear.
- Keep public APIs stable unless requested.

## Verification policy
- For code changes, run lightweight checks relevant to changed files first.
- If full test/build is not possible, report what was validated and what was not.

## Response policy
- Summarize what changed, why, and any residual risk.
- Include exact file paths touched.

## Git policy
- 使用 `git add` 时，必须显式指定文件路径，不直接对目录执行 add。
- 默认仅纳入已跟踪文件的修改与删除，绝不自动纳入未跟踪文件；如需新增文件，必须先明确获得用户许可并逐个说明原因。
- 如需纳入新增文件（未跟踪文件），必须逐个文件显式 add，并在提交说明中注明原因。
- 提交信息中不描述 `go.mod`、`go.sum`、`ggo.mod`、`Makefile` 的变化。

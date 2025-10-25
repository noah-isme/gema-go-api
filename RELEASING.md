# Releasing

## Releasing via GitHub Actions

The repository provides a `Release` workflow (`.github/workflows/release.yml`) that safely creates release branches and tags.

1. Open the **Actions** tab and launch the **Release** workflow via **Run workflow**.
2. Supply the desired `version` (for example `v1.0.0`).
3. Decide whether to keep `create_branch` and/or `create_tag` enabled.
   - Branches are created as `refs/heads/release/<version>`.
   - Tags are created as `refs/tags/<version>`.
4. Optionally override the target commit SHA via the `sha` input. When omitted, the workflow uses the commit that triggered the run (`context.sha`).
5. Trigger the workflow. Existing branch refs are skipped, while existing tags are force-updated to point at the requested commit so reruns remain idempotent.

The workflow uses the GitHub REST API namespace (`github.rest.git`) and includes a `gh api` fallback to ensure refs are created even if the primary step fails.

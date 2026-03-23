# Mark - Confluence Markdown Sync Tool

Mark is a CLI tool that syncs markdown files to Atlassian Confluence pages via the REST API. It compiles markdown to Confluence-compatible HTML with support for diagrams (Mermaid, D2), admonitions, macros, mentions, and file attachments.

## Repository Setup

- **Origin**: `gitlab.agodadev.io:Security/Architecture/mark.git` (our fork)
- **Upstream**: `github.com:kovetskiy/mark.git` (original project)
- Pull upstream changes with `git fetch upstream && git merge upstream/master`, but we maintain this fork independently.

## Build & Test

```bash
make build    # Build binary to ./mark (from ./cmd/mark, injects version/commit via ldflags)
make test     # Run tests with race detector and coverage (outputs profile.cov)
make image    # Build Docker image
```

- Go module: `github.com/kovetskiy/mark/v16` (kept for upstream compatibility)
- Go version: 1.25.0 (go.mod), CI uses 1.26.1
- CGO disabled for builds
- Entry point: `cmd/mark/main.go`

## Project Structure

| Package | Purpose |
|---------|---------|
| `cmd/mark/` | CLI entry point (urfave/cli/v3) |
| `mark.go` | Core library - `Config`, `Run()`, `ProcessFile()` |
| `confluence/` | Confluence REST API client |
| `markdown/` | Markdown-to-HTML compilation (goldmark) |
| `renderer/` | HTML rendering (blockquotes, code, headings, images, links, mentions) |
| `metadata/` | Page metadata extraction from frontmatter |
| `attachment/` | File attachment handling |
| `page/` | Page ancestry and link resolution |
| `macro/` | Confluence macro extraction/application |
| `includes/` | Template include processing |
| `d2/` | D2 diagram rendering |
| `mermaid/` | Mermaid diagram rendering (via chromedp) |
| `parser/` | Confluence-specific parsing (IDs, tags, mentions) |
| `stdlib/` | Standard library templates |
| `types/` | Shared type definitions |
| `util/` | CLI flags, auth, error handling |
| `vfs/` | Virtual filesystem abstraction |

## CI/CD

GitLab CI pipeline (`.gitlab-ci.yml`) with stages: lint, test, build, docker.
Docker images are pushed to GitLab Container Registry on master commits and tags.

## Key Dependencies

- `goldmark` - Markdown parser
- `chromedp` - Headless Chrome for Mermaid rendering
- `d2` - D2 diagram rendering
- `urfave/cli/v3` - CLI framework
- `gopencils` - REST API client wrapper

## Testing

Tests exist in: `markdown/`, `metadata/`, `renderer/`, `attachment/`, `page/`, `d2/`, `mermaid/`, `util/`.
Test fixtures are in `testdata/`.

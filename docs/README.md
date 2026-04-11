# molt docs

This directory contains the source for the molt documentation website, built with [VitePress](https://vitepress.dev).

## Docgen

`cmd/docgen` is a Go program that generates the reference pages from source:

| Output | Source |
| --- | --- |
| `reference/standard-library.md` | `internal/builtins/*.go` via `ModuleDocs()` |
| `reference/examples.md` | Every `.molt` file under `examples/` |
| `public/logo.png` | `assets/molt-logo.png` |

**To add or update standard-library documentation**, edit the module definitions under `internal/builtins/` and re-run docgen. Do not edit `reference/standard-library.md` directly.

**To add an example**, drop a `.molt` file into the appropriate subdirectory under `examples/` and re-run docgen. A leading `# comment` in the file becomes its description on the examples page.

Run docgen from the repository root:

```sh
go run ./cmd/docgen
```

Or via the dev helpers:

```sh
./dev.sh docs:gen
./dev.ps1 docs:gen
```

## Local development

Install dependencies:

```sh
cd docs
npm install
```

Generate reference pages, then start the dev server:

```sh
# from repo root
go run ./cmd/docgen

# from docs/
npm run docs:dev
```

Or use the combined helper:

```sh
./dev.sh docs        # gen + dev server
./dev.ps1 docs       # gen + dev server
```

## Production build

```sh
./dev.sh docs:build
./dev.ps1 docs:build
```

Output is written to `docs/.vitepress/dist/`.

## Deployment

The docs are deployed automatically to **Cloudflare Pages** on every push to `main` via `.github/workflows/docs.yml`. The workflow installs Go, runs docgen, installs Node, builds VitePress, and deploys the `docs/.vitepress/dist` output.

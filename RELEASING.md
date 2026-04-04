# Releasing miniblue

## Version scheme

miniblue follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (v2.0.0) - breaking API changes
- **MINOR** (v0.2.0) - new services, features, backwards-compatible
- **PATCH** (v0.1.1) - bug fixes, response format fixes

## How to release

### 1. Ensure main is clean

```bash
git checkout main
git pull origin main
make build
go test ./...
```

### 2. Tag the release

```bash
# For a new release:
git tag v0.1.0
git push origin v0.1.0

# For a patch:
git tag v0.1.1
git push origin v0.1.1
```

### 3. What happens automatically

When you push a tag matching `v*`:

| Step | Pipeline | Result |
|------|----------|--------|
| Build multi-arch image | `release.yml` | linux/amd64 + linux/arm64 |
| Push to Docker Hub | `release.yml` | `moabukar/miniblue:0.1.0` + `:latest` |
| Push to GHCR | `release.yml` | `ghcr.io/moabukar/miniblue:0.1.0` + `:latest` |
| Run tests | `ci.yml` | Build + test + vet + lint |
| Deploy docs | `docs.yml` | moabukar.github.io/miniblue (if website/ changed) |

### 4. Create GitHub release (optional)

```bash
gh release create v0.1.0 \
  --repo moabukar/miniblue \
  --title "v0.1.0" \
  --notes "Initial public release. 14 Azure services, Terraform support, azlocal CLI."
```

### 5. Verify

```bash
# Check images exist
docker pull moabukar/miniblue:0.1.0
docker pull ghcr.io/moabukar/miniblue:0.1.0

# Smoke test
docker run -p 4566:4566 -p 4567:4567 moabukar/miniblue:0.1.0 &
sleep 2
curl http://localhost:4566/health
```

## Image tags

| Push to | Tag |
|---------|-----|
| `main` branch | `:latest`, `:sha-abc1234` |
| `v1.2.3` tag | `:1.2.3`, `:latest`, `:sha-abc1234` |

## Required secrets

Set these in **Settings > Secrets > Actions** on the GitHub repo:

| Secret | Value |
|--------|-------|
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token ([create here](https://hub.docker.com/settings/security)) |

GHCR uses `GITHUB_TOKEN` automatically - no extra setup needed.

## Rollback

If a release has issues:

```bash
# Delete the tag
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0

# Fix the issue, then re-tag
git tag v0.1.1
git push origin v0.1.1
```

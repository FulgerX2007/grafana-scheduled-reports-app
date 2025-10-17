# Security Notes

## Secrets in Git History

If you cloned this repository before the secrets were removed, you may encounter GitHub push protection errors. The secrets have been removed from the codebase, but they may still exist in your local git history.

### Solution 1: Fresh Clone (Recommended)

The easiest solution is to:

1. Back up any local changes you've made
2. Delete your local repository
3. Clone fresh from GitHub
4. Reapply your changes

### Solution 2: Rewrite Git History (Advanced)

If you need to keep your local commits, you can rewrite the git history to remove the secrets:

```bash
# Install git-filter-repo
pip install git-filter-repo

# Create a file with patterns to replace
cat > /tmp/secrets-to-remove.txt << 'PATTERNS'
glsa_.*==><REDACTED_TOKEN>
regex:GF_SMTP_PASSWORD: .*==>${GF_SMTP_PASSWORD:-}
PATTERNS

# Run git-filter-repo
git filter-repo --replace-text /tmp/secrets-to-remove.txt --force
```

### Solution 3: Use .env File

The repository now uses environment variables for sensitive data:

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and add your actual SMTP credentials:
   ```
   GF_SMTP_PASSWORD=your_actual_password_here
   ```

3. The `.env` file is in `.gitignore` and will never be committed.

**Note**: Since Grafana 10.3+, the plugin uses managed service accounts for authentication. No manual service account token configuration is required.

## Never Commit Secrets

- Always use environment variables for sensitive data
- Never commit `.env` files
- Use `.env.example` with placeholder values for documentation
- Review commits before pushing to remote repositories

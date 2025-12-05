# Real-World Workflows

How people actually use atlas-cli. No contrived examples, just stuff that works.

## Bulk Export Before Migration

Your team is moving from Confluence to something else (or just backing up docs).

```bash
# 1. Export all page IDs from a space
atl page list MYSPACE --format json > pages.json

# 2. Extract IDs and export each page
cat pages.json | jq -r '.[].id' | while read id; do
  atl page view $id -o "export/$id.md" --format markdown --with-images
  echo "Exported page $id"
done

# 3. Now you have markdown files with images
ls export/
# 12345678.md  12345679.md  12345680.md  ...
```

**With images:** Add `--with-images` flag. Slow but complete.

**Without images:** Skip the flag. Faster, but broken image links.

**Alternative (storage format):**
```bash
# Export as Confluence XHTML (preserves formatting perfectly)
cat pages.json | jq -r '.[].id' | while read id; do
  atl page view $id -o "export/$id.html" --format storage
done
```

Use storage format if you're migrating to another Confluence instance.

---

## Create Pages from Template

You have a design doc template and need to create 10 new docs.

```bash
# 1. Export template from existing page
atl page view 11111111 -o design-template.html --format storage

# 2. Edit template (replace placeholder text)
vim design-template.html

# 3. Create new pages with variations
for project in api-redesign db-migration frontend-rewrite; do
  # Copy template
  cp design-template.html /tmp/${project}.html

  # Replace placeholders (use sed, or edit manually)
  sed -i "s/PROJECT_NAME/${project}/g" /tmp/${project}.html

  # Create page
  atl page create -s MYSPACE -t "Design: ${project}" \
    -f /tmp/${project}.html -p "Design Docs"

  echo "Created design doc for ${project}"
done
```

**Better approach:** Use a script with placeholders, generate HTML, then create.

---

## Daily Standup: Check Your Work

See what you worked on yesterday and what PRs are waiting.

```bash
# Your open PRs
echo "=== Open PRs ==="
atl pr list PROJ/repo --state OPEN --format json | \
  jq -r '.[] | select(.author.user.name == "your.username") | "[\(.id)] \(.title)"'

# JIRAs for each PR
echo -e "\n=== JIRA Status ==="
atl issue list --assignee me --status "In Progress"

# PRs linked to your JIRA tickets
atl issue list --assignee me --format json | \
  jq -r '.[].key' | \
  xargs -I {} sh -c 'echo "{}:"; atl issue prs {}'
```

**Make it a script:** Save to `~/bin/standup` and run it every morning.

---

## Search and Export Filtered Results

Find all meeting notes from December and export them.

```bash
# 1. Search
atl page search "meeting" -s MYSPACE --modified month --format json > meetings.json

# 2. Export
cat meetings.json | jq -r '.[].id' | while read id; do
  title=$(cat meetings.json | jq -r ".[] | select(.id == \"$id\") | .title")
  # Sanitize filename
  filename=$(echo "$title" | tr ' /:' '_')
  atl page view $id -o "meetings/${filename}.md" --format markdown
done

ls meetings/
# 2024-12-01_Team_Meeting.md
# 2024-12-04_Product_Sync.md
# ...
```

---

## Bulk Create Child Pages

Create a bunch of pages under a parent.

```bash
parent_id=12345678

for topic in Introduction Setup Configuration Troubleshooting FAQ; do
  echo "<h1>${topic}</h1><p>TODO: Fill this in</p>" > /tmp/${topic}.html

  atl page create -s MYSPACE -t "${topic}" \
    -f /tmp/${topic}.html -p $parent_id

  echo "Created ${topic} page"
done
```

---

## Export and Reimport (Content Backup)

Back up a page, make risky edits in UI, restore if needed.

```bash
# Backup
atl page view 12345678 -o backup.html --format storage

# [... make changes in Confluence UI ...]

# Restore if you messed up
atl page edit 12345678 -f backup.html
```

**Warning:** This overwrites the current version. Confluence keeps history, but don't be stupid.

---

## PR Review Workflow

Quick check of PRs waiting for review.

```bash
# Open PRs not created by you
atl pr list PROJ/repo --state OPEN --format json | \
  jq -r '.[] | select(.author.user.name != "your.username") |
    "[\(.id)] \(.title)\n  Author: \(.author.user.displayName)\n  \(.links.self[0].href)\n"'

# View specific PR
atl pr view PROJ/repo 123

# Check diff
atl pr diff PROJ/repo 123 | less

# Merge after review
atl pr merge PROJ/repo 123
```

---

## Create Documentation Structure

Set up a new doc space with standard structure.

```bash
space="NEWTEAM"
sections=(
  "Getting Started"
  "Architecture"
  "API Reference"
  "Deployment"
  "Troubleshooting"
)

# Create root page
root_html="<h1>Team Documentation</h1><p>Main index page</p>"
atl page create -s $space -t "Documentation Home" -c "$root_html"

# Get root page ID
root_id=$(atl page search --title "Documentation Home" -s $space --format json | jq -r '.[0].id')

# Create child sections
for section in "${sections[@]}"; do
  content="<h1>${section}</h1><p>TODO</p>"
  atl page create -s $space -t "$section" -c "$content" -p $root_id
done
```

---

## Monitor JIRA Ticket Progress

Check if JIRA ticket has PRs merged yet.

```bash
ticket="PROJ-123"

# View ticket
atl issue view $ticket

# Check linked PRs
echo "=== Linked PRs ==="
atl issue prs $ticket

# Check if all PRs are merged
atl issue prs $ticket --format json | jq -r '.[].state' | grep -v MERGED
# If empty output, all PRs are merged
```

**Use in CI:** Check if JIRA has merged PRs before deploying.

---

## Export Space Table of Contents

Generate a markdown TOC for a space.

```bash
space="MYSPACE"

echo "# ${space} Table of Contents" > toc.md
echo "" >> toc.md

atl page list $space --format json | \
  jq -r '.[] | "- [\(.title)](\(.links.webui))"' >> toc.md

cat toc.md
```

Output:
```markdown
# MYSPACE Table of Contents

- [Getting Started](https://confluence.../display/MYSPACE/Getting+Started)
- [Architecture](https://confluence.../display/MYSPACE/Architecture)
...
```

---

## Sync Local Markdown to Confluence

You write docs in markdown locally, sync to Confluence.

**Requirements:**
- Use a markdown-to-html converter (pandoc, etc.)
- Store page IDs in frontmatter or separate mapping file

```bash
# 1. Convert markdown to HTML
pandoc design.md -o design.html

# 2. Update existing page
page_id=12345678
atl page edit $page_id -f design.html

# Or create new page
atl page create -s MYSPACE -t "Design Doc" -f design.html
```

**Better:** Build a script that tracks page ID <-> markdown file mapping.

---

## Clone Page to Another Space

Copy a page to different space (API doesn't support this directly).

```bash
source_id=12345678
target_space="NEWSPACE"

# 1. Export content
atl page view $source_id -o /tmp/clone.html --format storage

# 2. Get original title
title=$(atl page view $source_id --info --format json | jq -r '.title')

# 3. Create in new space
atl page create -s $target_space -t "$title" -f /tmp/clone.html

echo "Cloned to $target_space"
```

**Images won't copy.** You'd need to download images separately and upload them to new space.

---

## Batch Delete Old Pages

Delete pages older than X that match a pattern.

```bash
# Find pages to delete
atl page search "DRAFT" -s MYSPACE --format json > drafts.json

# Review list
cat drafts.json | jq -r '.[] | "\(.id): \(.title)"'

# Delete (with confirmation for each)
cat drafts.json | jq -r '.[].id' | while read id; do
  atl page delete $id
  # Prompts for each page - type ID to confirm
done

# Or skip confirmation (dangerous)
cat drafts.json | jq -r '.[].id' | xargs -I {} atl page delete {} --yes
```

**Warning:** Test on a few pages first. Deletes are permanent on Server.

---

## Generate Release Notes from JIRA

Get all tickets in a release and format as release notes.

```bash
version="1.2.0"

# Get tickets (assumes JQL filter or label)
atl issue list --project PROJ --status Done --format json > issues.json

# Format as markdown
echo "# Release ${version}" > release-notes.md
echo "" >> release-notes.md

cat issues.json | jq -r '.[] | "- \(.key): \(.fields.summary)"' >> release-notes.md

# Upload to Confluence
pandoc release-notes.md -o release-notes.html
atl page create -s MYSPACE -t "Release ${version}" -f release-notes.html -p "Releases"
```

---

## Weekly PR Digest

See what PRs were merged this week.

```bash
# Get merged PRs
atl pr list PROJ/repo --state MERGED --format json > merged.json

# Filter by date (this week)
# Note: No date filter in CLI yet, filter with jq
cat merged.json | jq -r '.[] | select(.updatedDate > now - 7*86400*1000) |
  "[\(.id)] \(.title) - merged by \(.author.user.displayName)"'
```

---

## Pre-Commit Hook: Check JIRA Reference

Ensure commits reference a JIRA ticket.

```bash
#!/bin/bash
# .git/hooks/commit-msg

commit_msg=$(cat "$1")
jira_pattern="^(PROJ|WEBAPP)-[0-9]+"

if ! echo "$commit_msg" | grep -qE "$jira_pattern"; then
  echo "Error: Commit message must start with JIRA ticket (e.g., PROJ-123)"
  exit 1
fi

# Optional: Verify ticket exists
ticket=$(echo "$commit_msg" | grep -oE "$jira_pattern" | head -1)
atl issue view "$ticket" > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "Warning: JIRA ticket $ticket not found or you don't have access"
  # Don't fail, just warn
fi
```

---

## Archive Old Docs

Export old docs to archive space before deleting.

```bash
archive_space="ARCHIVE"
old_space="OLDTEAM"

# 1. List pages in old space
atl page list $old_space --format json > old-pages.json

# 2. Export and recreate in archive space
cat old-pages.json | jq -r '.[].id' | while read id; do
  # Get content and metadata
  atl page view $id -o /tmp/page.html --format storage
  title=$(atl page view $id --info --format json | jq -r '.title')

  # Create in archive space with prefix
  atl page create -s $archive_space -t "[${old_space}] ${title}" -f /tmp/page.html

  echo "Archived: $title"
done

# 3. Review in archive space, then delete from old space
# (manual verification step)
```

---

## Confluence -> GitHub Wiki Sync

Export Confluence pages to GitHub wiki format.

```bash
space="DOCS"
wiki_dir="/path/to/repo.wiki"

atl page list $space --format json | jq -r '.[].id' | while read id; do
  title=$(atl page view $id --info --format json | jq -r '.title')
  filename=$(echo "$title" | tr ' /:' '-').md

  atl page view $id -o "${wiki_dir}/${filename}" --format markdown
  echo "Exported: $filename"
done

# Commit to wiki repo
cd $wiki_dir
git add .
git commit -m "Sync from Confluence"
git push
```

---

## Tips for Scripting

**Use `--format json`** - easier to parse with jq
```bash
atl page list SPACE --format json | jq -r '.[].id'
```

**Error handling:**
```bash
if ! atl page view 12345678 > /dev/null 2>&1; then
  echo "Page doesn't exist or auth failed"
  exit 1
fi
```

**Batch operations with `xargs`:**
```bash
cat ids.txt | xargs -I {} atl page delete {} --yes
```

**Rate limiting:** Add delays for large batches
```bash
cat ids.txt | while read id; do
  atl page view $id -o "export/$id.md" --format markdown
  sleep 1  # Don't hammer the API
done
```

---

## See Also

- [Usage](USAGE.md) - Full command reference
- [Configuration](CONFIGURATION.md) - API tokens and config
- [Troubleshooting](TROUBLESHOOTING.md) - When things break

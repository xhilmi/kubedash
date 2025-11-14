# Commit Guide

This document provides suggested commit messages and commands for committing the improvements.

## 📋 Recommended Commit Strategy

You can commit all changes at once or split into logical commits. Here are both approaches:

---

## Option 1: Single Comprehensive Commit (Recommended for Quick Deploy)

### Commit Message

```
feat: major enhancements - deployment safety, visual improvements, and UX polish

New Features (November 2025):
- Add color-coded operation type badges in Resource History
  * Blue (Edit), Green (Resume), Amber (Rollback), Gray (Restart), Cyan (Scale), Orange (Suspend)
- Add YAML save confirmation dialog with detailed warnings
- Add Helm-based rollback with automatic FluxCD suspension
- Add manual FluxCD Suspend/Resume controls
- Add Helm History tab with image version tracking
- Add Flux Status tab with real-time monitoring
- Add comprehensive action history tracking (all deployment operations)

UX & Safety Improvements:
- Add confirmation dialogs for ALL risky actions (Scale, Restart, Rollback, Suspend, Resume, YAML Save)
- Convert all text to human-friendly conversational English
- Add cross-reference tips between tabs and action buttons
- Add toast notifications with emojis for all actions
- Implement consistent dialog pattern with warnings and resource info
- Add color coding for better visual distinction of operations

Technical Improvements:
- Implement real-time log filtering across all logs
- Fix scale validation to allow scaling to 0 replicas
- Fix type assertion errors in handlers
- Remove flux CLI dependency (~50MB smaller Docker image)
- Add multi-version FluxCD API support
- Add 4 new Badge variants (success, warning, info, orange)

Security Enhancements:
- Add mandatory confirmation before YAML configuration changes
- Prevent accidental deployments with safety dialogs
- Force users to read warnings before executing risky operations
- Maintain complete audit trail in Resource History

See CHANGES.md for detailed documentation.
```

### Commands

```bash
# Navigate to repo
cd <your-repo-path>

# Check status
git status

# Add all changes
git add .

# Commit with the message above
git commit -F- <<'EOF'
feat: major enhancements - deployment safety, visual improvements, and UX polish

New Features (November 2025):
- Add color-coded operation type badges in Resource History
  * Blue (Edit), Green (Resume), Amber (Rollback), Gray (Restart), Cyan (Scale), Orange (Suspend)
- Add YAML save confirmation dialog with detailed warnings
- Add Helm-based rollback with automatic FluxCD suspension
- Add manual FluxCD Suspend/Resume controls
- Add Helm History tab with image version tracking
- Add Flux Status tab with real-time monitoring
- Add comprehensive action history tracking (all deployment operations)

UX & Safety Improvements:
- Add confirmation dialogs for ALL risky actions (Scale, Restart, Rollback, Suspend, Resume, YAML Save)
- Convert all text to human-friendly conversational English
- Add cross-reference tips between tabs and action buttons
- Add toast notifications with emojis for all actions
- Implement consistent dialog pattern with warnings and resource info
- Add color coding for better visual distinction of operations

Technical Improvements:
- Implement real-time log filtering across all logs
- Fix scale validation to allow scaling to 0 replicas
- Fix type assertion errors in handlers
- Remove flux CLI dependency (~50MB smaller Docker image)
- Add multi-version FluxCD API support
- Add 4 new Badge variants (success, warning, info, orange)

Security Enhancements:
- Add mandatory confirmation before YAML configuration changes
- Prevent accidental deployments with safety dialogs
- Force users to read warnings before executing risky operations
- Maintain complete audit trail in Resource History

See CHANGES.md for detailed documentation.
EOF

# Push to remote
git push origin main
```

---

## Option 2: Multiple Logical Commits (Recommended for Better Git History)

### Commit 1: UI Safety & Visual Improvements

```bash
git add ui/src/components/ui/badge.tsx
git add ui/src/components/resource-history-table.tsx
git add ui/src/components/yaml-save-confirmation-dialog.tsx
git add ui/src/components/yaml-editor.tsx
git add ui/src/components/scale-confirmation-dialog.tsx
git add ui/src/components/restart-confirmation-dialog.tsx
git add ui/src/components/rollback-confirmation-dialog.tsx
git add ui/src/components/suspend-confirmation-dialog.tsx
git add ui/src/components/resume-confirmation-dialog.tsx
git add ui/src/pages/deployment-detail.tsx

git commit -m "feat(ui): add color-coded badges and YAML save confirmation

- Add 4 new Badge variants: success (green), warning (amber), info (cyan), orange
- Add color-coded operation types in Resource History table
  * Edit: Blue, Resume: Green, Rollback: Amber, Restart: Gray, Scale: Cyan, Suspend: Orange
- Add YAML save confirmation dialog with detailed warnings
- Add confirmation dialogs for all deployment actions (Scale, Restart, Rollback, Suspend, Resume)
- Implement consistent safety pattern across all risky operations
- Remove popover intermediate step - direct to confirmation dialogs
- Update deployment detail page with all confirmation dialogs
- Improve user safety by forcing review of warnings before actions"
```

### Commit 2: Documentation Updates

```bash
git add CHANGES.md
git add COMMIT.md
git add docs/guide/resource-history.md

git commit -m "docs: update documentation with latest features

- Update CHANGES.md with color-coded badges and YAML confirmation features
- Add detailed operation type reference table
- Update resource-history.md with color coding information
- Add best practices section for using Resource History
- Update commit guide with latest features
- Add security enhancements section
- Update version to 2.6.3 and date to November 6, 2025"
```

### Push All Commits

```bash
git push origin main
```

---

## 🔍 Pre-Commit Checklist

Before committing, make sure:

- [ ] All files are saved
- [ ] Docker image builds successfully
- [ ] No TypeScript errors in build
- [ ] Go code compiles without errors
- [ ] Test key features in browser:
  - [ ] Rollback works and suspends FluxCD
  - [ ] Helm tab shows history with image versions
  - [ ] Flux tab shows status correctly
  - [ ] Log filtering works in real-time
  - [ ] Scale to 0 works without error
  - [ ] All action history types show correctly

---

## 📤 After Pushing

1. **Tag the release** (optional):
   ```bash
   git tag -a v2.3.x -m "Custom fork with deployment management improvements"
   git push origin v2.3.x
   ```

2. **Build and push Docker image**:
   ```bash
   # Build the image
   docker build -t <your-registry>/<image-name>:2.6.3 .
   docker push <your-registry>/<image-name>:2.6.3
   
   # Also tag as latest (optional)
   docker tag <your-registry>/<image-name>:2.6.3 <your-registry>/<image-name>:latest
   docker push <your-registry>/<image-name>:latest
   
   # Example with actual values:
   docker build -t example-registry/kite:2.6.3 .
   docker push example-registry/kite:2.6.3
   ```

3. **Deploy to cluster**:
   ```bash
   # Example with Helm
   helm upgrade --install <release-name> <chart> -f <values-file> -n <namespace>
   ```

4. **Verify deployment**:
   ```bash
   kubectl get pods -n <namespace>
   kubectl logs -n <namespace> -l app=<app-label> --tail=50
   ```

---

## 💡 Tips

- **First time pushing?** Set upstream:
  ```bash
  git push -u origin main
  ```

- **Want to see what changed?**
  ```bash
  git diff HEAD
  git status
  ```

- **Need to amend commit?**
  ```bash
  git commit --amend
  git push --force-with-lease origin main
  ```

- **Check commit history:**
  ```bash
  git log --oneline --graph --decorate -10
  ```

---

**Ready to commit?** Choose your option above and run the commands! 🚀

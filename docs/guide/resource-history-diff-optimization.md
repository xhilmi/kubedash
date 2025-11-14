# Resource History Diff Optimization

## Overview

Untuk meningkatkan efisiensi penyimpanan dan keamanan, resource history sekarang menyimpan **hanya diff (patch)** alih-alih full YAML untuk setiap perubahan.

## Perubahan

### 1. Backend Database Schema

**File**: `pkg/model/resource_history.go`

```go
type ResourceHistory struct {
    // ...existing fields...
    
    // NEW: Store only unified diff to save disk space
    YAMLDiff string `json:"yamlDiff" gorm:"type:text"`
    
    // Only populated for CREATE operations (when there's no previous version)
    ResourceYAML string `json:"resourceYaml" gorm:"type:text"`
    
    // Deprecated: kept for backward compatibility
    PreviousYAML string `json:"previousYaml" gorm:"type:text"`
}
```

### 2. Diff Utility dengan Security

**File**: `pkg/utils/diff.go`

Fitur keamanan:
- ✅ **Size Limits**: Maksimal 2MB per YAML, 5MB per diff
- ✅ **UTF-8 Validation**: Validasi input untuk mencegah binary injection
- ✅ **Timeout Protection**: Maksimal 2 detik untuk generate diff
- ✅ **Label Sanitization**: Hapus control characters dari labels
- ✅ **Memory Safe**: Tidak load semua YAML sekaligus

Functions:
- `GenerateUnifiedDiff(oldYAML, newYAML string) string` - Generate unified diff patch
- `ApplyDiff(oldYAML, diffPatch string) string` - Apply diff to reconstruct YAML
- `GenerateHumanReadableDiff(old, new, oldLabel, newLabel string) string` - For display

### 3. Storage Strategy

**CREATE Operations**:
```go
history.ResourceYAML = currYAML  // Store full YAML
history.YAMLDiff = ""             // No diff needed
```

**UPDATE/EDIT Operations**:
```go
history.YAMLDiff = GenerateUnifiedDiff(prevYAML, currYAML)  // Store only diff
history.ResourceYAML = ""                                     // Don't store full YAML
```

### 4. API Endpoints

#### List History (Optimized)
```
GET /:resourceType/:namespace/:name/history?page=1&pageSize=10
```

Response tidak include YAML fields (hemat bandwidth):
```json
{
  "data": [
    {
      "id": 123,
      "sequenceId": 45,
      "operationType": "update",
      "success": true,
      // NO resourceYaml/previousYaml fields
    }
  ]
}
```

#### Get History Detail (On-Demand)
```
GET /:resourceType/:namespace/:name/history/:historyId
```

Response dengan full YAML (reconstructed from diffs):
```json
{
  "id": 123,
  "resourceYaml": "apiVersion: apps/v1...",  // Reconstructed
  "previousYaml": "apiVersion: apps/v1...",  // Reconstructed
  "operationType": "update"
}
```

### 5. Frontend Changes

**File**: `ui/src/components/resource-history-table.tsx`

- ✅ List tidak load YAML (hemat memory)
- ✅ Fetch detail only saat view diff
- ✅ Loading state saat fetch detail

**File**: `ui/src/components/yaml-diff-viewer.tsx`

Labels yang jelas:
- **Previous vs Modified**: "Previous (Old)" ← → "Modified (New)"
- **Current vs Modified**: "Current (Live)" ← → "Modified"
- **Create operation**: "Empty" ← → "Created (New)"

### 6. Environment Variables

**File**: `pkg/common/common.go`

```bash
# Maximum helm revision history to fetch (default: 20)
HELM_MAX_REVISIONS=20
```

## Benefits

### 🔒 Security
1. **Size limits** mencegah DoS via large diffs
2. **UTF-8 validation** mencegah binary injection
3. **Timeout protection** mencegah hanging
4. **Label sanitization** mencegah XSS

### 💾 Storage Efficiency
1. **~70-90% reduction** dalam disk space untuk updates
2. CREATE operation: Store full YAML (~5-10KB)
3. UPDATE operation: Store diff only (~500B-2KB)

Example:
```
Before: 100 history records × 8KB = 800KB
After:  1 CREATE (8KB) + 99 UPDATEs (99 × 1KB) = 107KB
Savings: ~87%
```

### ⚡ Performance
1. **List history** tidak load YAML → faster API response
2. **Frontend** tidak parse YAML sampai user view diff
3. **Bandwidth** hemat 70-90% untuk list operations

### 🔄 Backward Compatibility
- Old records dengan `previousYaml`/`resourceYaml` tetap bisa dibaca
- Migration tidak diperlukan (soft migration)
- Reconstruction fallback ke deprecated fields jika tidak ada diff

## Migration Guide

### Automatic (Recommended)
Tidak perlu migration manual. System akan:
1. Baca old records menggunakan deprecated fields
2. New records otomatis menggunakan diff strategy
3. Gradual transition saat resources di-update

### Manual (Optional)
Jika ingin convert old records ke diff format:

```sql
-- Backup first!
CREATE TABLE resource_histories_backup AS SELECT * FROM resource_histories;

-- Analyze disk space usage
SELECT 
  pg_size_pretty(pg_total_relation_size('resource_histories')) as total_size,
  count(*) as total_records,
  count(*) FILTER (WHERE yaml_diff != '') as diff_records,
  count(*) FILTER (WHERE resource_yaml != '') as full_yaml_records
FROM resource_histories;
```

## Testing

### Test Cases
1. ✅ CREATE operation stores full YAML
2. ✅ UPDATE operation stores diff only
3. ✅ List history returns minimal data
4. ✅ Get history detail reconstructs full YAML
5. ✅ Diff too large (>5MB) handled gracefully
6. ✅ Invalid UTF-8 rejected
7. ✅ Labels sanitized properly
8. ✅ Backward compatibility with old records

### Security Tests
```bash
# Test size limits
curl -X POST /api/v1/deployments/test/large-app \
  -d @large-deployment.yaml  # Should fail if >2MB

# Test UTF-8 validation
# Binary data should be rejected during YAML parse

# Test timeout
# Very complex diffs should timeout at 2s
```

## Monitoring

### Metrics to Watch
```sql
-- Average diff size
SELECT AVG(LENGTH(yaml_diff)) as avg_diff_size 
FROM resource_histories 
WHERE yaml_diff != '';

-- Storage savings
SELECT 
  SUM(LENGTH(resource_yaml) + LENGTH(previous_yaml)) as old_method_size,
  SUM(LENGTH(yaml_diff) + LENGTH(resource_yaml)) as new_method_size,
  (1 - SUM(LENGTH(yaml_diff) + LENGTH(resource_yaml))::float / 
       SUM(LENGTH(resource_yaml) + LENGTH(previous_yaml))::float) * 100 as savings_percent
FROM resource_histories;

-- Reconstruction performance
-- Monitor GetHistoryDetail endpoint latency (should be <100ms for reasonable chains)
```

## Troubleshooting

### "Diff too large to store"
```
Solution: Resource YAML exceeds limits. Check:
1. Is YAML really >2MB? (very unusual)
2. Are there base64 embedded data? (consider external storage)
3. Large ConfigMaps/Secrets? (use volume mounts instead)
```

### "Failed to reconstruct YAML"
```
Solution: Diff chain broken. Check:
1. Database consistency
2. Missing intermediate history records
3. Fallback: use deprecated fields if available
```

### Slow history loading
```
Solution:
1. Check reconstruction chain length (optimize query)
2. Add database index on (cluster_name, resource_type, resource_name, created_at)
3. Consider caching reconstructed YAMLs
```

## Related Files

### Backend
- `pkg/model/resource_history.go` - Model definition
- `pkg/utils/diff.go` - Diff utility with security
- `pkg/handlers/resources/generic_resource_handler.go` - History handlers
- `pkg/handlers/resources/handler.go` - Route registration

### Frontend
- `ui/src/lib/api.ts` - API client
- `ui/src/components/resource-history-table.tsx` - History table
- `ui/src/components/yaml-diff-viewer.tsx` - Diff viewer with labels
- `ui/src/types/api.ts` - Type definitions

### Documentation
- `docs/config/env.md` - Environment variables
- `docs/guide/resource-history.md` - User guide

## References
- [go-diff library](https://github.com/sergi/go-diff) - Diff algorithm
- [Unified Diff Format](https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html)

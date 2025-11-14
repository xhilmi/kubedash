import { useVersionInfo } from '@/lib/api'

export function VersionInfo() {
  const { data: versionInfo } = useVersionInfo()

  if (!versionInfo) return null

  // Professional display for Kubedash Kubernetes Platform
  const version = versionInfo.version.replace(/^v/, '')
  const displayVersion = version === 'dev' 
    ? 'K8s Platform'  // Professional name for platform
    : `v${version}`   // e.g., v2.0.4 for releases
  
  // Clean commit display - show version or build info
  const commitShort = versionInfo.commitId === 'unknown' 
    ? 'Enterprise' 
    : versionInfo.commitId.slice(0, 7)

  return (
    <div className="text-[10px] text-muted-foreground/60 font-mono leading-none">
      {displayVersion} • {commitShort}
    </div>
  )
}

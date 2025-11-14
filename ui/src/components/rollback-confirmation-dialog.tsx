import { useState, useEffect } from 'react'
import { AlertTriangle } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface RollbackConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  releaseName: string
  revision: string
  namespace?: string
  suspendFlux: boolean
  onSuspendFluxChange: (checked: boolean) => void
  onRevisionChange: (revision: string) => void
  onReleaseNameChange: (releaseName: string) => void
  onConfirm: () => void
  isRollingBack?: boolean
}

export function RollbackConfirmationDialog({
  open,
  onOpenChange,
  releaseName,
  revision,
  namespace,
  suspendFlux,
  onSuspendFluxChange,
  onRevisionChange,
  onReleaseNameChange,
  onConfirm,
  isRollingBack,
}: RollbackConfirmationDialogProps) {
  const [localRevision, setLocalRevision] = useState(revision)
  const [localReleaseName, setLocalReleaseName] = useState(releaseName)

  useEffect(() => {
    setLocalRevision(revision)
  }, [revision])

  useEffect(() => {
    setLocalReleaseName(releaseName)
  }, [releaseName])

  const handleDialogChange = (open: boolean) => {
    onOpenChange(open)
  }

  const handleConfirm = () => {
    onConfirm()
  }

  const handleRevisionChange = (value: string) => {
    setLocalRevision(value)
    onRevisionChange(value)
  }

  const handleReleaseNameChange = (value: string) => {
    setLocalReleaseName(value)
    onReleaseNameChange(value)
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-500/10">
              <AlertTriangle className="h-5 w-5 text-amber-500" />
            </div>
            <div className="flex-1">
              <DialogTitle className="text-left">
                Confirm Rollback to Revision {localRevision || 'Previous'}
              </DialogTitle>
              <DialogDescription className="text-left">
                This will revert your deployment to a previous state
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          {/* Important: Check Helm tab first */}
          <div className="rounded-lg bg-blue-500/5 p-4 border border-blue-500/20">
            <div className="text-sm">
              <p className="font-medium text-blue-600 dark:text-blue-500 mb-2">
                📋 Before you rollback:
              </p>
              <p className="text-muted-foreground">
                <strong>1. Check the "Helm" tab first</strong> to view all available revisions
              </p>
              <p className="text-muted-foreground mt-1">
                <strong>2. Review the revision history</strong> to see image versions, dates, and changes
              </p>
              <p className="text-muted-foreground mt-1">
                <strong>3. Choose the revision number</strong> that has the stable image you want to revert to
              </p>
            </div>
          </div>

          <div className="rounded-lg bg-amber-500/5 p-4 border border-amber-500/20">
            <div className="text-sm">
              <p className="font-medium text-amber-600 dark:text-amber-500 mb-2">
                🔙 You are about to rollback:
              </p>
              <div className="space-y-1 text-muted-foreground">
                {namespace && (
                  <p>
                    <span className="font-medium">Namespace:</span> {namespace}
                  </p>
                )}
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="releaseName">Helm Release Name</Label>
            <Input
              id="releaseName"
              placeholder="e.g., nginx-test"
              value={localReleaseName}
              onChange={(e) => handleReleaseNameChange(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              The name of the Helm release to rollback. Usually matches your deployment name.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="revision">Revision Number (from Helm Tab)</Label>
            <Input
              id="revision"
              placeholder="e.g., 5 (or leave empty for previous revision)"
              value={localRevision}
              onChange={(e) => handleRevisionChange(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              💡 <strong>Tip:</strong> Check the <strong>"Helm"</strong> tab to see revision numbers and their corresponding image versions. Enter a specific revision number, or leave empty to rollback to the previous revision.
            </p>
          </div>

          <div className="rounded-lg bg-blue-500/5 p-3 border border-blue-500/20">
            <p className="text-sm text-blue-600 dark:text-blue-400">
              <strong>💡 What happens:</strong>
            </p>
            <ul className="text-sm text-muted-foreground mt-2 space-y-1 ml-4 list-disc">
              <li>Your app will revert to the selected revision</li>
              <li>Current pods will be recreated with old settings</li>
              <li>This may cause brief downtime during the transition</li>
            </ul>
          </div>

          <div className="space-y-3">
            <div className="flex items-start space-x-2">
              <Checkbox
                id="suspendFlux"
                checked={suspendFlux}
                onCheckedChange={(checked) => onSuspendFluxChange(checked === true)}
                className="mt-0.5"
              />
              <div className="flex-1">
                <Label
                  htmlFor="suspendFlux"
                  className="text-sm font-medium cursor-pointer"
                >
                  Suspend FluxCD after rollback
                </Label>
                <p className="text-xs text-muted-foreground mt-1">
                  Highly recommended to prevent FluxCD from auto-upgrading back to the latest version
                </p>
              </div>
            </div>
          </div>

          <div className="rounded-lg bg-amber-500/5 p-3 border border-amber-500/20">
            <p className="text-xs text-amber-600 dark:text-amber-400">
              ⚠️ <strong>Warning:</strong> Rolling back to an incompatible version may cause issues. Make sure the selected revision is stable and compatible with your current environment.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleDialogChange(false)}
            disabled={isRollingBack}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleConfirm}
            disabled={isRollingBack || !releaseName}
            className="bg-amber-600 hover:bg-amber-700"
          >
            {isRollingBack ? 'Rolling back...' : 'Confirm Rollback'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

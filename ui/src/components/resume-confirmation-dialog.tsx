import { useState, useEffect } from 'react'
import { AlertTriangle } from 'lucide-react'

import { Button } from '@/components/ui/button'
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

interface ResumeConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  deploymentName: string
  namespace?: string
  onConfirm: (releaseName: string) => void
  isResuming?: boolean
}

export function ResumeConfirmationDialog({
  open,
  onOpenChange,
  deploymentName,
  namespace,
  onConfirm,
  isResuming,
}: ResumeConfirmationDialogProps) {
  const [releaseName, setReleaseName] = useState('')

  // Pre-populate release name with deployment name when dialog opens
  useEffect(() => {
    if (open && deploymentName) {
      setReleaseName(deploymentName)
    }
  }, [open, deploymentName])

  const handleDialogChange = (open: boolean) => {
    if (!open) {
      setReleaseName('')
    }
    onOpenChange(open)
  }

  const handleConfirm = () => {
    onConfirm(releaseName)
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-500/10">
              <AlertTriangle className="h-5 w-5 text-green-500" />
            </div>
            <div className="flex-1">
              <DialogTitle className="text-left">
                ▶️ Resume FluxCD Auto-Sync
              </DialogTitle>
              <DialogDescription className="text-left">
                Turn automatic updates back on
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg bg-green-500/5 p-4 border border-green-500/20">
            <div className="text-sm">
              <p className="font-medium text-green-600 dark:text-green-500 mb-2">
                ▶️ Deployment to Resume
              </p>
              <div className="space-y-1 text-muted-foreground">
                <p>
                  <span className="font-medium">Deployment:</span> {deploymentName}
                </p>
                {namespace && (
                  <p>
                    <span className="font-medium">Namespace:</span> {namespace}
                  </p>
                )}
              </div>
            </div>
          </div>

          <div className="rounded-lg bg-blue-500/5 p-3 border border-blue-500/20">
            <p className="text-sm text-blue-600 dark:text-blue-400">
              <strong>💡 What this does:</strong>
            </p>
            <ul className="text-sm text-muted-foreground mt-2 space-y-1 ml-4 list-disc">
              <li>Turns FluxCD back on to sync your app with Git</li>
              <li>Your app will automatically upgrade to the latest image from Git</li>
              <li>Returns to normal automated deployment workflow</li>
            </ul>
          </div>

          <div className="space-y-2">
            <Label htmlFor="releaseName">Helm Release Name</Label>
            <Input
              id="releaseName"
              placeholder="e.g., nginx-test"
              value={releaseName}
              onChange={(e) => setReleaseName(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              The name of the Helm release to resume. Check the <strong>Flux tab</strong> for the exact name.
            </p>
          </div>

          <div className="rounded-lg bg-amber-500/5 p-3 border border-amber-500/20">
            <p className="text-xs text-amber-600 dark:text-amber-400">
              ⚠️ <strong>Important:</strong> Make sure your Git repo is ready - your app will sync to the latest version immediately!
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleDialogChange(false)}
            disabled={isResuming}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleConfirm}
            disabled={isResuming || !releaseName}
            className="bg-green-600 hover:bg-green-700"
          >
            {isResuming ? 'Resuming...' : 'Confirm Resume'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

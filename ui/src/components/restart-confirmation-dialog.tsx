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

interface RestartConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  deploymentName: string
  namespace?: string
  onConfirm: () => void
  isRestarting?: boolean
}

export function RestartConfirmationDialog({
  open,
  onOpenChange,
  deploymentName,
  namespace,
  onConfirm,
  isRestarting,
}: RestartConfirmationDialogProps) {
  const handleDialogChange = (open: boolean) => {
    onOpenChange(open)
  }

  const handleConfirm = () => {
    onConfirm()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-500/10">
              <AlertTriangle className="h-5 w-5 text-blue-500" />
            </div>
            <div className="flex-1">
              <DialogTitle className="text-left">
                🔄 Restart Deployment
              </DialogTitle>
              <DialogDescription className="text-left">
                Recreate all pods with current settings
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg bg-blue-500/5 p-4 border border-blue-500/20">
            <div className="text-sm">
              <p className="font-medium text-blue-600 dark:text-blue-500 mb-2">
                🔄 Deployment to Restart
              </p>
              <div className="space-y-1 text-muted-foreground">
                <p>
                  <span className="font-medium">Name:</span> {deploymentName}
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
              <li>Recreates all pods with the current configuration</li>
              <li>Useful after changing configs or for troubleshooting</li>
              <li>Fresh start without changing any settings</li>
            </ul>
          </div>

          <div className="rounded-lg bg-amber-500/5 p-3 border border-amber-500/20">
            <p className="text-xs text-amber-600 dark:text-amber-400">
              ⚠️ <strong>Heads up:</strong> Your app will be briefly unavailable during the restart. Best done during low traffic periods.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleDialogChange(false)}
            disabled={isRestarting}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleConfirm}
            disabled={isRestarting}
            className="bg-blue-600 hover:bg-blue-700"
          >
            {isRestarting ? 'Restarting...' : 'Confirm Restart'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

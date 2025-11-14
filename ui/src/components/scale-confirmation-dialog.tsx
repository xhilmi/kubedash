import { useState } from 'react'
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

interface ScaleConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  deploymentName: string
  currentReplicas: number
  namespace?: string
  onConfirm: (replicas: number) => void
  isScaling?: boolean
}

export function ScaleConfirmationDialog({
  open,
  onOpenChange,
  deploymentName,
  currentReplicas,
  namespace,
  onConfirm,
  isScaling,
}: ScaleConfirmationDialogProps) {
  const [replicas, setReplicas] = useState(currentReplicas)

  const handleDialogChange = (open: boolean) => {
    if (open) {
      setReplicas(currentReplicas)
    }
    onOpenChange(open)
  }

  const handleConfirm = () => {
    onConfirm(replicas)
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
                📈 Scale Deployment
              </DialogTitle>
              <DialogDescription className="text-left">
                Adjust the number of pod replicas
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-lg bg-blue-500/5 p-4 border border-blue-500/20">
            <div className="text-sm">
              <p className="font-medium text-blue-600 dark:text-blue-500 mb-2">
                📊 Deployment Details
              </p>
              <div className="space-y-1 text-muted-foreground">
                <p>
                  <span className="font-medium">Name:</span> {deploymentName}
                </p>
                <p>
                  <span className="font-medium">Current Replicas:</span> {currentReplicas}
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
              <li>Adjust pod count for more capacity or to save resources</li>
              <li>Scale up for increased traffic, scale down to reduce costs</li>
              <li>Scale to 0 temporarily stops the app without deleting it</li>
            </ul>
          </div>

          <div className="space-y-2">
            <Label htmlFor="replicas">Number of Replicas</Label>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="h-9 w-9 p-0"
                onClick={() => setReplicas(Math.max(0, replicas - 1))}
                disabled={replicas <= 0}
              >
                -
              </Button>
              <Input
                id="replicas"
                type="number"
                min="0"
                value={replicas}
                onChange={(e) => setReplicas(parseInt(e.target.value) || 0)}
                className="text-center"
              />
              <Button
                variant="outline"
                size="sm"
                className="h-9 w-9 p-0"
                onClick={() => setReplicas(replicas + 1)}
              >
                +
              </Button>
            </div>
          </div>

          <div className="rounded-lg bg-amber-500/5 p-3 border border-amber-500/20">
            <p className="text-xs text-amber-600 dark:text-amber-400">
              ⚠️ <strong>Heads up:</strong> Scaling to 0 makes your app unavailable. New pods take time to start when scaling up.
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleDialogChange(false)}
            disabled={isScaling}
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={handleConfirm}
            disabled={isScaling}
            className="bg-blue-600 hover:bg-blue-700"
          >
            {isScaling ? 'Scaling...' : 'Confirm Scale'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

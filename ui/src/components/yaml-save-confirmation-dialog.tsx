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

interface YamlSaveConfirmationDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  resourceName: string
  resourceType: string
  namespace?: string
  onConfirm: () => void
  isLoading?: boolean
}

export function YamlSaveConfirmationDialog({
  open,
  onOpenChange,
  resourceName,
  resourceType,
  namespace,
  onConfirm,
  isLoading = false,
}: YamlSaveConfirmationDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/30">
              <AlertTriangle className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <DialogTitle>Confirm YAML Configuration Save</DialogTitle>
          </div>
          <DialogDescription asChild>
            <div className="space-y-4 pt-4">
              {/* Resource Info Card */}
              <div className="rounded-lg border bg-muted/50 p-4 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    Resource Type:
                  </span>
                  <span className="text-sm font-medium">{resourceType}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Name:</span>
                  <span className="text-sm font-medium">{resourceName}</span>
                </div>
                {namespace && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-muted-foreground">
                      Namespace:
                    </span>
                    <span className="text-sm font-medium">{namespace}</span>
                  </div>
                )}
              </div>

              {/* What will happen */}
              <div className="space-y-2">
                <p className="text-sm font-semibold text-foreground">
                  What will happen:
                </p>
                <ul className="space-y-1.5 text-sm text-muted-foreground">
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500 mt-0.5">•</span>
                    <span>
                      The YAML configuration will be applied to the cluster
                    </span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500 mt-0.5">•</span>
                    <span>
                      Your resource will be updated with the new configuration
                    </span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500 mt-0.5">•</span>
                    <span>
                      Pods may restart if the changes affect their specification
                    </span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="text-blue-500 mt-0.5">•</span>
                    <span>
                      This action will be recorded in the Resource History
                    </span>
                  </li>
                </ul>
              </div>

              {/* Warnings */}
              <div className="rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20 p-4 space-y-2">
                <p className="text-sm font-semibold text-amber-900 dark:text-amber-200">
                  ⚠️ Important:
                </p>
                <ul className="space-y-1.5 text-sm text-amber-800 dark:text-amber-300">
                  <li className="flex items-start gap-2">
                    <span className="mt-0.5">•</span>
                    <span>
                      Make sure your YAML syntax is correct to avoid errors
                    </span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="mt-0.5">•</span>
                    <span>
                      Invalid configurations may cause the resource to fail
                    </span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="mt-0.5">•</span>
                    <span>
                      Test critical changes in a non-production environment first
                    </span>
                  </li>
                  <li className="flex items-start gap-2">
                    <span className="mt-0.5">•</span>
                    <span>
                      You can rollback to previous versions from Resource History
                    </span>
                  </li>
                </ul>
              </div>

              {/* Additional note */}
              <p className="text-xs text-muted-foreground italic">
                💡 Tip: Review the changes carefully before confirming. Check
                the Resource History tab to see previous configurations if
                needed.
              </p>
            </div>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            onClick={onConfirm}
            disabled={isLoading}
            className="bg-blue-600 hover:bg-blue-700 focus:ring-blue-600"
          >
            {isLoading ? 'Saving...' : 'Confirm & Save'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

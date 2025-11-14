import { useCallback, useEffect, useState } from 'react'
import {
  IconHistory,
  IconInfoCircle,
  IconLoader,
  IconPlayerPause,
  IconPlayerPlay,
  IconRefresh,
  IconReload,
  IconScale,
  IconTrash,
} from '@tabler/icons-react'
import * as yaml from 'js-yaml'
import { Deployment } from 'kubernetes-types/apps/v1'
import { Container } from 'kubernetes-types/core/v1'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  editDeployment,
  getFluxStatus,
  getHelmHistory,
  getHelmValues,
  detectHelmRelease,
  restartDeployment,
  rollbackDeployment,
  resumeHelmRelease,
  scaleDeployment,
  suspendHelmRelease,
  useResource,
  useResourcesWatch,
} from '@/lib/api'
import { useCluster } from '@/hooks/use-cluster'
import { getDeploymentStatus, toSimpleContainer } from '@/lib/k8s'
import { formatDate, translateError } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { ResponsiveTabs } from '@/components/ui/responsive-tabs'
import { ContainerTable } from '@/components/container-table'
import { DeploymentStatusIcon } from '@/components/deployment-status-icon'
import { DescribeDialog } from '@/components/describe-dialog'
import { ErrorMessage } from '@/components/error-message'
import { EventTable } from '@/components/event-table'
import { LabelsAnno } from '@/components/lables-anno'
import { LogViewer } from '@/components/log-viewer'
import { PodMonitoring } from '@/components/pod-monitoring'
import { PodTable } from '@/components/pod-table'
import { RelatedResourcesTable } from '@/components/related-resource-table'
import { ResourceDeleteConfirmationDialog } from '@/components/resource-delete-confirmation-dialog'
import { ResourceHistoryTable } from '@/components/resource-history-table'
import { RestartConfirmationDialog } from '@/components/restart-confirmation-dialog'
import { ResumeConfirmationDialog } from '@/components/resume-confirmation-dialog'
import { RollbackConfirmationDialog } from '@/components/rollback-confirmation-dialog'
import { ScaleConfirmationDialog } from '@/components/scale-confirmation-dialog'
import { SuspendConfirmationDialog } from '@/components/suspend-confirmation-dialog'
import { Terminal } from '@/components/terminal'
import { VolumeTable } from '@/components/volume-table'
import { YamlEditor } from '@/components/yaml-editor'

export function DeploymentDetail(props: { namespace: string; name: string }) {
  const { namespace, name } = props
  const [scaleReplicas, setScaleReplicas] = useState<number>(1)
  const [yamlContent, setYamlContent] = useState('')
  const [isSavingYaml, setIsSavingYaml] = useState(false)
  const [isScaleConfirmOpen, setIsScaleConfirmOpen] = useState(false)
  const [isRestartConfirmOpen, setIsRestartConfirmOpen] = useState(false)
  const [isSuspendConfirmOpen, setIsSuspendConfirmOpen] = useState(false)
  const [isResumeConfirmOpen, setIsResumeConfirmOpen] = useState(false)
  const [isRollbackConfirmOpen, setIsRollbackConfirmOpen] = useState(false)
  const [rollbackReleaseName, setRollbackReleaseName] = useState<string>('')
  const [rollbackRevision, setRollbackRevision] = useState<string>('')
  const [rollbackSuspendFlux, setRollbackSuspendFlux] = useState<boolean>(true)
  const [suspendResumeName, setSuspendResumeName] = useState<string>('')
  const [refreshKey, setRefreshKey] = useState(0)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState<number>(0)
  const { t } = useTranslation()

  // Fetch deployment data
  const {
    data: deployment,
    isLoading: isLoadingDeployment,
    isError: isDeploymentError,
    error: deploymentError,
    refetch: refetchDeployment,
  } = useResource('deployments', name, namespace, {
    refreshInterval,
  })

  const labelSelector = deployment?.spec?.selector.matchLabels
    ? Object.entries(deployment.spec.selector.matchLabels)
        .map(([key, value]) => `${key}=${value}`)
        .join(',')
    : undefined
  const { data: relatedPods, isLoading: isLoadingPods } = useResourcesWatch(
    'pods',
    namespace,
    {
      labelSelector,
      enabled: !!deployment?.spec?.selector.matchLabels,
    }
  )

  useEffect(() => {
    if (deployment) {
      setYamlContent(yaml.dump(deployment, { indent: 2 }))
      setScaleReplicas(deployment.spec?.replicas || 1)
    }
  }, [deployment])

  // Auto-reset refresh interval when deployment reaches stable state
  useEffect(() => {
    if (deployment) {
      const status = getDeploymentStatus(deployment)
      const isStable =
        status === 'Available' ||
        status === 'Scaled Down' ||
        status === 'Paused'

      if (isStable) {
        const timer = setTimeout(() => {
          setRefreshInterval(0)
        }, 2000)
        return () => clearTimeout(timer)
      } else {
        setRefreshInterval(1000)
      }
    }
  }, [deployment, refreshInterval])

  const handleRefresh = () => {
    setRefreshKey((prev) => prev + 1)
    refetchDeployment()
    toast.success('Screen refreshed! 🔄')
  }

  const handleRestart = useCallback(async () => {
    if (!deployment) return

    try {
      await restartDeployment(name, namespace)
      toast.success('Deployment restarting - hang tight!')
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to restart deployment:', error)
      toast.error(translateError(error, t))
    }
  }, [t, deployment, name, namespace])

  const handleScale = useCallback(async () => {
    if (!deployment) return

    try {
      await scaleDeployment(namespace, name, scaleReplicas)
      toast.success(`Deployment scaled to ${scaleReplicas} replicas`)
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to scale deployment:', error)
      toast.error(translateError(error, t))
    }
  }, [t, deployment, name, namespace, scaleReplicas])

  const handleRollback = useCallback(async () => {
    if (!deployment || !rollbackReleaseName) return

    try {
      const revision = rollbackRevision ? parseInt(rollbackRevision) : undefined
      const result = await rollbackDeployment(
        name, 
        namespace, 
        rollbackReleaseName, 
        revision,
        rollbackSuspendFlux
      )
      toast.success(result.message || 'Rolled back successfully! 🎉')
      setRollbackReleaseName('')
      setRollbackRevision('')
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to rollback helm release:', error)
      toast.error(translateError(error, t))
    }
  }, [deployment, t, name, namespace, rollbackReleaseName, rollbackRevision, rollbackSuspendFlux])

  const handleSuspend = useCallback(async (releaseName?: string) => {
    if (!deployment) return
    
    const nameToUse = releaseName || suspendResumeName
    if (!nameToUse) return

    try {
      const result = await suspendHelmRelease(name, namespace, nameToUse)
      toast.success(result.message || 'FluxCD paused - you\'re in manual mode now!')
      setSuspendResumeName('')
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to suspend HelmRelease:', error)
      toast.error(translateError(error, t))
    }
  }, [deployment, t, name, namespace, suspendResumeName])

  const handleResume = useCallback(async (releaseName?: string) => {
    if (!deployment) return
    
    const nameToUse = releaseName || suspendResumeName
    if (!nameToUse) return

    try {
      const result = await resumeHelmRelease(name, namespace, nameToUse)
      toast.success(result.message || 'FluxCD is back online - auto-sync enabled! ✅')
      setSuspendResumeName('')
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to resume HelmRelease:', error)
      toast.error(translateError(error, t))
    }
  }, [deployment, t, name, namespace, suspendResumeName])

  const handleSaveYaml = async (content: Deployment) => {
    setIsSavingYaml(true)
    try {
      await editDeployment(name, namespace, content)
      toast.success('Changes saved! 💾')
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to save YAML:', error)
      toast.error(translateError(error, t))
    } finally {
      setIsSavingYaml(false)
    }
  }

  const handleYamlChange = (content: string) => {
    setYamlContent(content)
  }

  const handleContainerUpdate = async (
    updatedContainer: Container,
    init = false
  ) => {
    if (!deployment) return

    try {
      // Create a deep copy of the deployment
      const updatedDeployment = { ...deployment }

      if (init) {
        // Update the specific container in the deployment spec
        if (updatedDeployment.spec?.template?.spec?.initContainers) {
          const containerIndex =
            updatedDeployment.spec.template.spec.initContainers.findIndex(
              (c) => c.name === updatedContainer.name
            )

          if (containerIndex >= 0) {
            updatedDeployment.spec.template.spec.initContainers[
              containerIndex
            ] = updatedContainer
          }
        }
      } else {
        // Update the specific container in the deployment spec
        if (updatedDeployment.spec?.template?.spec?.containers) {
          const containerIndex =
            updatedDeployment.spec.template.spec.containers.findIndex(
              (c) => c.name === updatedContainer.name
            )

          if (containerIndex >= 0) {
            updatedDeployment.spec.template.spec.containers[containerIndex] =
              updatedContainer
          }
        }
      }

      // Call the edit API (requires 'edit' verb permission)
      await editDeployment(name, namespace, updatedDeployment)
      toast.success(`Container ${updatedContainer.name} updated! 🚀`)
      setRefreshInterval(1000)
    } catch (error) {
      console.error('Failed to update container:', error)
      toast.error(translateError(error, t))
    }
  }

  if (isLoadingDeployment) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-center gap-2">
              <IconLoader className="animate-spin" />
              <span>Loading deployment details...</span>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (isDeploymentError || !deployment) {
    return (
      <ErrorMessage
        resourceName={'Deployment'}
        error={deploymentError}
        refetch={handleRefresh}
      />
    )
  }

  const { status } = deployment
  const readyReplicas = status?.readyReplicas || 0
  const totalReplicas = status?.replicas || 0

  return (
    <div className="space-y-2">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-bold">{name}</h1>
          <p className="text-muted-foreground">
            Namespace: <span className="font-medium">{namespace}</span>
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleRefresh}>
            <IconRefresh className="w-4 h-4" />
            Refresh
          </Button>
          <DescribeDialog
            resourceType="deployments"
            namespace={namespace}
            name={name}
          />
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => setIsScaleConfirmOpen(true)}
          >
            <IconScale className="w-4 h-4" />
            Scale
          </Button>
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => setIsRestartConfirmOpen(true)}
          >
            <IconReload className="w-4 h-4" />
            Restart
          </Button>
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => {
              setRollbackReleaseName(name)
              setIsRollbackConfirmOpen(true)
            }}
          >
            <IconHistory className="w-4 h-4" />
            Rollback
          </Button>
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => {
              setSuspendResumeName(name)
              setIsSuspendConfirmOpen(true)
            }}
          >
            <IconPlayerPause className="w-4 h-4" />
            Suspend
          </Button>
          <Button 
            variant="outline" 
            size="sm"
            onClick={() => {
              setSuspendResumeName(name)
              setIsResumeConfirmOpen(true)
            }}
          >
            <IconPlayerPlay className="w-4 h-4" />
            Resume
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setIsDeleteDialogOpen(true)}
          >
            <IconTrash className="w-4 h-4" />
            Delete
          </Button>
        </div>
      </div>
      {/* Tabs */}
      <ResponsiveTabs
        tabs={[
          {
            value: 'overview',
            label: 'Overview',
            content: (
              <div className="space-y-4">
                {/* Status Overview */}
                <Card>
                  <CardHeader>
                    <CardTitle>Status Overview</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
                      <div className="flex items-center gap-3">
                        <div className="flex items-center gap-2">
                          <DeploymentStatusIcon
                            status={getDeploymentStatus(deployment)}
                          />
                        </div>
                        <div>
                          <p className="text-xs text-muted-foreground">
                            Status
                          </p>
                          <p className="text-sm font-medium">
                            {getDeploymentStatus(deployment)}
                          </p>
                        </div>
                      </div>

                      <div>
                        <p className="text-xs text-muted-foreground">
                          Ready Replicas
                        </p>
                        <p className="text-sm font-medium">
                          {readyReplicas} / {totalReplicas}
                        </p>
                      </div>

                      <div>
                        <p className="text-xs text-muted-foreground">
                          Updated Replicas
                        </p>
                        <p className="text-sm font-medium">
                          {status?.updatedReplicas || 0}
                        </p>
                      </div>

                      <div>
                        <p className="text-xs text-muted-foreground">
                          Available Replicas
                        </p>
                        <p className="text-sm font-medium">
                          {status?.availableReplicas || 0}
                        </p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
                {/* Deployment Info */}
                <Card>
                  <CardHeader>
                    <CardTitle>Deployment Information</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          Created
                        </Label>
                        <p className="text-sm">
                          {formatDate(
                            deployment.metadata?.creationTimestamp || '',
                            true
                          )}
                        </p>
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          Strategy
                        </Label>
                        <p className="text-sm">
                          {deployment.spec?.strategy?.type || 'RollingUpdate'}
                        </p>
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          Replicas
                        </Label>
                        <p className="text-sm">
                          {deployment.spec?.replicas || 0}
                        </p>
                      </div>
                      <div>
                        <Label className="text-xs text-muted-foreground">
                          Selector
                        </Label>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {Object.entries(
                            deployment.spec?.selector?.matchLabels || {}
                          ).map(([key, value]) => (
                            <Badge
                              key={key}
                              variant="secondary"
                              className="text-xs"
                            >
                              {key}: {value}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    </div>
                    <LabelsAnno
                      labels={deployment.metadata?.labels || {}}
                      annotations={deployment.metadata?.annotations || {}}
                    />
                  </CardContent>
                </Card>

                {deployment.spec?.template.spec?.initContainers?.length &&
                  deployment.spec?.template.spec?.initContainers?.length >
                    0 && (
                    <Card>
                      <CardHeader>
                        <CardTitle>
                          Init Containers (
                          {
                            deployment.spec?.template?.spec?.initContainers
                              ?.length
                          }
                          )
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-6">
                          <div className="space-y-4">
                            {deployment.spec?.template?.spec?.initContainers?.map(
                              (container) => (
                                <ContainerTable
                                  key={container.name}
                                  container={container}
                                  onContainerUpdate={(updatedContainer) =>
                                    handleContainerUpdate(
                                      updatedContainer,
                                      true
                                    )
                                  }
                                />
                              )
                            )}
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  )}
                <Card>
                  <CardHeader>
                    <CardTitle>
                      Containers (
                      {deployment.spec?.template?.spec?.containers?.length || 0}
                      )
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-6">
                      <div className="space-y-4">
                        {deployment.spec?.template?.spec?.containers?.map(
                          (container) => (
                            <ContainerTable
                              key={container.name}
                              container={container}
                              onContainerUpdate={(updatedContainer) =>
                                handleContainerUpdate(updatedContainer)
                              }
                            />
                          )
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Conditions */}
                {status?.conditions && (
                  <Card>
                    <CardHeader>
                      <CardTitle>Conditions</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {status.conditions.map((condition, index) => (
                          <div
                            key={index}
                            className="flex items-center gap-3 p-2 border rounded"
                          >
                            <Badge
                              variant={
                                condition.status === 'True'
                                  ? 'default'
                                  : 'secondary'
                              }
                            >
                              {condition.type}
                            </Badge>
                            <span className="text-sm">{condition.message}</span>
                            <span className="text-xs text-muted-foreground ml-auto">
                              {formatDate(
                                condition.lastTransitionTime ||
                                  condition.lastUpdateTime ||
                                  ''
                              )}
                            </span>
                          </div>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}
              </div>
            ),
          },
          {
            value: 'yaml',
            label: 'YAML',
            content: (
              <YamlEditor<'deployments'>
                key={refreshKey}
                value={yamlContent}
                title="YAML Configuration"
                onSave={handleSaveYaml}
                onChange={handleYamlChange}
                isSaving={isSavingYaml}
                resourceName={name}
                resourceType="Deployment"
                namespace={namespace}
              />
            ),
          },
          ...(relatedPods
            ? [
                {
                  value: 'pods',
                  label: (
                    <>
                      Pods{' '}
                      {relatedPods && (
                        <Badge variant="secondary">{relatedPods.length}</Badge>
                      )}
                    </>
                  ),
                  content: (
                    <PodTable
                      pods={relatedPods}
                      isLoading={isLoadingPods}
                      labelSelector={labelSelector}
                    />
                  ),
                },
                {
                  value: 'logs',
                  label: 'Logs',
                  content: (
                    <div className="space-y-6">
                      <LogViewer
                        namespace={namespace}
                        pods={relatedPods}
                        containers={deployment.spec?.template.spec?.containers}
                        initContainers={
                          deployment.spec?.template.spec?.initContainers
                        }
                        labelSelector={labelSelector}
                      />
                    </div>
                  ),
                },
                {
                  value: 'terminal',
                  label: 'Terminal',
                  content: (
                    <div className="space-y-6">
                      {relatedPods && relatedPods.length > 0 && (
                        <Terminal
                          namespace={namespace}
                          pods={relatedPods}
                          containers={
                            deployment.spec?.template.spec?.containers
                          }
                          initContainers={
                            deployment.spec?.template.spec?.initContainers
                          }
                        />
                      )}
                    </div>
                  ),
                },
              ]
            : []),
          {
            value: 'Related',
            label: 'Related',
            content: (
              <RelatedResourcesTable
                resource={'deployments'}
                name={name}
                namespace={namespace}
              />
            ),
          },
          {
            value: 'history',
            label: 'History',
            content: (
              <ResourceHistoryTable
                resourceType="deployments"
                name={name}
                namespace={namespace}
              />
            ),
          },
          ...(deployment.spec?.template?.spec?.volumes
            ? [
                {
                  value: 'volumes',
                  label: (
                    <>
                      Volumes{' '}
                      <Badge variant="secondary">
                        {deployment.spec.template.spec.volumes.length}
                      </Badge>
                    </>
                  ),
                  content: (
                    <VolumeTable
                      namespace={namespace}
                      volumes={deployment.spec?.template?.spec?.volumes}
                      containers={toSimpleContainer(
                        deployment.spec?.template?.spec?.initContainers,
                        deployment.spec?.template?.spec?.containers
                      )}
                      isLoading={isLoadingDeployment}
                    />
                  ),
                },
              ]
            : []),
          {
            value: 'events',
            label: 'Events',
            content: (
              <EventTable
                resource="deployments"
                name={name}
                namespace={namespace}
              />
            ),
          },
          {
            value: 'monitor',
            label: 'Monitor',
            content: (
              <PodMonitoring
                namespace={namespace}
                pods={relatedPods}
                containers={deployment.spec?.template.spec?.containers}
                initContainers={deployment.spec?.template.spec?.initContainers}
                labelSelector={labelSelector}
              />
            ),
          },
          {
            value: 'helm',
            label: 'Helm',
            content: <HelmTab namespace={namespace} name={name} />,
          },
          {
            value: 'flux',
            label: 'Flux',
            content: <FluxTab namespace={namespace} name={name} />,
          },
        ]}
      />

      <ResourceDeleteConfirmationDialog
        open={isDeleteDialogOpen}
        onOpenChange={setIsDeleteDialogOpen}
        resourceName={name}
        resourceType="deployments"
        namespace={namespace}
      />

      <RollbackConfirmationDialog
        open={isRollbackConfirmOpen}
        onOpenChange={setIsRollbackConfirmOpen}
        releaseName={rollbackReleaseName}
        revision={rollbackRevision || ''}
        namespace={namespace}
        suspendFlux={rollbackSuspendFlux}
        onSuspendFluxChange={setRollbackSuspendFlux}
        onRevisionChange={setRollbackRevision}
        onReleaseNameChange={setRollbackReleaseName}
        onConfirm={async () => {
          await handleRollback()
          setIsRollbackConfirmOpen(false)
        }}
      />

      <ScaleConfirmationDialog
        open={isScaleConfirmOpen}
        onOpenChange={setIsScaleConfirmOpen}
        deploymentName={name}
        currentReplicas={scaleReplicas}
        namespace={namespace}
        onConfirm={async (replicas) => {
          setScaleReplicas(replicas)
          await handleScale()
          setIsScaleConfirmOpen(false)
        }}
      />

      <RestartConfirmationDialog
        open={isRestartConfirmOpen}
        onOpenChange={setIsRestartConfirmOpen}
        deploymentName={name}
        namespace={namespace}
        onConfirm={async () => {
          await handleRestart()
          setIsRestartConfirmOpen(false)
        }}
      />

      <SuspendConfirmationDialog
        open={isSuspendConfirmOpen}
        onOpenChange={setIsSuspendConfirmOpen}
        deploymentName={name}
        namespace={namespace}
        onConfirm={async (releaseName) => {
          await handleSuspend(releaseName)
          setIsSuspendConfirmOpen(false)
        }}
      />

      <ResumeConfirmationDialog
        open={isResumeConfirmOpen}
        onOpenChange={setIsResumeConfirmOpen}
        deploymentName={name}
        namespace={namespace}
        onConfirm={async (releaseName) => {
          await handleResume(releaseName)
          setIsResumeConfirmOpen(false)
        }}
      />
    </div>
  )
}

// HelmTab component for displaying Helm release history
function HelmTab({ namespace, name }: { namespace: string; name: string }) {
  const { currentCluster } = useCluster()
  const [helmHistory, setHelmHistory] = useState<any[]>([])
  const [detectedReleaseName, setDetectedReleaseName] = useState<string>(name)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedRevision, setSelectedRevision] = useState<number | null>(null)
  const [revisionValues, setRevisionValues] = useState<any>(null)
  const [isLoadingValues, setIsLoadingValues] = useState(false)
  const [currentPage, setCurrentPage] = useState(1)
  const pageSize = 10 // Show 10 revisions per page

  const loadHelmHistory = useCallback(async () => {
    if (!currentCluster) return
    
    setIsLoading(true)
    setError(null)
    setHelmHistory([]) // Clear previous data
    try {
      // First, detect the actual helm release name from deployment labels
      const detection = await detectHelmRelease(name, namespace)
      const actualReleaseName = detection.releaseName
      setDetectedReleaseName(actualReleaseName) // Save for display
      
      if (detection.detected) {
        console.log(`Detected Helm release name: ${actualReleaseName} (deployment: ${name})`)
      }
      
      const history = await getHelmHistory(name, namespace, actualReleaseName)
      setHelmHistory(history)
      setCurrentPage(1) // Reset to first page on reload
    } catch (err: any) {
      // Handle "not found" errors gracefully
      const errorMessage = err.message || 'Failed to load Helm history'
      if (errorMessage.includes('not found')) {
        setError(null) // Don't show error for "not found"
        setHelmHistory([]) // Just show empty state
      } else {
        setError(errorMessage)
      }
    } finally {
      setIsLoading(false)
    }
  }, [name, namespace, currentCluster])

  // Reload when cluster changes or component mounts
  useEffect(() => {
    loadHelmHistory()
  }, [loadHelmHistory])

  const loadRevisionValues = useCallback(
    async (revision: number) => {
      setIsLoadingValues(true)
      setSelectedRevision(revision)
      try {
        const values = await getHelmValues(name, namespace, name, revision)
        setRevisionValues(values)
      } catch (err: any) {
        toast.error('Failed to load revision values')
        setRevisionValues(null)
      } finally {
        setIsLoadingValues(false)
      }
    },
    [name, namespace]
  )

  // Pagination logic
  const totalPages = Math.ceil(helmHistory.length / pageSize)
  const startIndex = (currentPage - 1) * pageSize
  const endIndex = startIndex + pageSize
  const paginatedHistory = helmHistory.slice(startIndex, endIndex)

  const goToPage = (page: number) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page)
    }
  }

  return (
    <div className="space-y-4">
      {/* Info Banner */}
      <div className="p-4 border border-blue-200 bg-blue-50 dark:bg-blue-950 dark:border-blue-800 rounded-md">
        <div className="flex gap-3">
          <IconInfoCircle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-2">
            <h4 className="font-semibold text-blue-900 dark:text-blue-100">
              📦 Helm List
            </h4>
            <p className="text-sm text-blue-800 dark:text-blue-200">
              This shows all versions of your app that have been deployed. Each row displays the revision number, deployment time, status, and most importantly - the <strong>image version</strong> that was running.
            </p>
            <p className="text-sm text-blue-800 dark:text-blue-200">
              <strong>💡 Quick tip:</strong> Before rolling back, check this table to see which image version was stable. Click any row to view that revision's full configuration.
            </p>
            <p className="text-sm text-blue-800 dark:text-blue-200">
              <strong>🎯 Common use:</strong> "Which image version was working yesterday?" - Just look at the Image Version column!
            </p>
          </div>
        </div>
      </div>
      
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Helm List</CardTitle>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">
                Release: <span className="font-mono font-medium">{detectedReleaseName}</span>
                {detectedReleaseName !== name && (
                  <span className="ml-1 text-xs text-blue-600 dark:text-blue-400">
                    (detected from {name})
                  </span>
                )}
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={loadHelmHistory}
                disabled={isLoading}
              >
                <IconRefresh className={isLoading ? 'animate-spin' : ''} size={16} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-4 border border-destructive/50 rounded-md bg-destructive/10">
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}

          {isLoading && helmHistory.length === 0 && (
            <div className="flex items-center justify-center py-8">
              <IconLoader className="animate-spin mr-2" size={20} />
              <span className="text-muted-foreground">Loading Helm history...</span>
            </div>
          )}

          {!isLoading && helmHistory.length === 0 && !error && (
            <div className="text-center py-8 text-muted-foreground">
              No Helm release found for <span className="font-mono">{detectedReleaseName}</span>
            </div>
          )}

          {helmHistory.length > 0 && (
            <>
              <div className="rounded-md border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="px-4 py-3 text-left font-medium">Revision</th>
                      <th className="px-4 py-3 text-left font-medium">Helm Release</th>
                      <th className="px-4 py-3 text-left font-medium">Updated</th>
                      <th className="px-4 py-3 text-left font-medium">Status</th>
                      <th className="px-4 py-3 text-left font-medium">Chart</th>
                      <th className="px-4 py-3 text-left font-medium">App Version</th>
                      <th className="px-4 py-3 text-left font-medium">Image</th>
                      <th className="px-4 py-3 text-left font-medium">Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {paginatedHistory.map((rev: any) => (
                      <tr
                        key={rev.revision}
                        className="border-b hover:bg-muted/50 cursor-pointer"
                        onClick={() => loadRevisionValues(rev.revision)}
                      >
                        <td className="px-4 py-3 font-mono">{rev.revision}</td>
                        <td className="px-4 py-3">
                          <span className="font-mono text-xs bg-muted px-2 py-1 rounded">
                            {detectedReleaseName}
                          </span>
                        </td>
                        <td className="px-4 py-3">{formatDate(rev.updated)}</td>
                        <td className="px-4 py-3">
                          <Badge variant={rev.status === 'deployed' ? 'default' : 'secondary'}>
                            {rev.status}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">{rev.chart}</td>
                        <td className="px-4 py-3">{rev.app_version}</td>
                        <td className="px-4 py-3 font-mono text-xs">
                          {rev.imageVersion || '-'}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">{rev.description}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Pagination Controls */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between px-2 py-3 border-t">
                  <div className="text-sm text-muted-foreground">
                    Showing {startIndex + 1} to {Math.min(endIndex, helmHistory.length)} of {helmHistory.length} revisions
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => goToPage(currentPage - 1)}
                      disabled={currentPage === 1}
                    >
                      Previous
                    </Button>
                    <div className="flex items-center gap-1">
                      {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                        <Button
                          key={page}
                          variant={currentPage === page ? 'default' : 'outline'}
                          size="sm"
                          onClick={() => goToPage(page)}
                          className="w-10"
                        >
                          {page}
                        </Button>
                      ))}
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => goToPage(currentPage + 1)}
                      disabled={currentPage === totalPages}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {selectedRevision !== null && (
        <Card>
          <CardHeader>
            <CardTitle>
              Revision {selectedRevision} Values
              {isLoadingValues && (
                <IconLoader className="inline ml-2 animate-spin" size={16} />
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {revisionValues && (
              <pre className="p-4 bg-muted rounded-md overflow-auto text-xs">
                {JSON.stringify(revisionValues, null, 2)}
              </pre>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// FluxTab component for displaying FluxCD HelmRelease status
function FluxTab({ namespace, name }: { namespace: string; name: string }) {
  const { currentCluster } = useCluster()
  const [fluxStatus, setFluxStatus] = useState<any>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadFluxStatus = useCallback(async () => {
    if (!currentCluster) return
    
    setIsLoading(true)
    setError(null)
    setFluxStatus(null) // Clear previous data
    try {
      const status = await getFluxStatus(name, namespace, name)
      setFluxStatus(status)
    } catch (err: any) {
      // Handle "not found" errors gracefully
      const errorMessage = err.message || 'Failed to load Flux status'
      if (errorMessage.includes('not found')) {
        setError(null) // Don't show error for "not found"
        setFluxStatus(null) // Just show empty state
      } else {
        setError(errorMessage)
      }
    } finally {
      setIsLoading(false)
    }
  }, [name, namespace, currentCluster])

  // Reload when cluster changes or component mounts
  useEffect(() => {
    loadFluxStatus()
  }, [loadFluxStatus])

  const refreshStatus = useCallback(() => {
    loadFluxStatus()
  }, [loadFluxStatus])

  return (
    <div className="space-y-4">
      {/* Info Banner */}
      <div className="p-4 border border-green-200 bg-green-50 dark:bg-green-950 dark:border-green-800 rounded-md">
        <div className="flex gap-3">
          <IconInfoCircle className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-2">
            <h4 className="font-semibold text-green-900 dark:text-green-100">
              🔄 FluxCD Status (GitOps Autopilot)
            </h4>
            <p className="text-sm text-green-800 dark:text-green-200">
              FluxCD automatically keeps your app synced with your Git repository. Think of it as cruise control for deployments.
            </p>
            <p className="text-sm text-green-800 dark:text-green-200">
              <strong>▶️ Resume (Active):</strong> Use this to enable auto-updates. Your app will automatically upgrade to the latest image version from Git. Perfect when you want continuous deployment.
            </p>
            <p className="text-sm text-green-800 dark:text-green-200">
              <strong>⏸️ Suspend (Paused):</strong> Use this when rolling back to a specific image version. This prevents FluxCD from auto-upgrading back to latest after your rollback. Your image stays pinned until you resume.
            </p>
            <p className="text-sm text-green-800 dark:text-green-200">
              <strong>🎯 Pro workflow:</strong> Rollback to old image → Suspend FluxCD → Test → When ready, Resume to go back to latest.
            </p>
          </div>
        </div>
      </div>
      
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>FluxCD HelmRelease Status</CardTitle>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">
                Release: <span className="font-mono font-medium">{name}</span>
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={refreshStatus}
                disabled={isLoading}
              >
                <IconRefresh
                  size={16}
                  className={isLoading ? 'animate-spin' : ''}
                />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="p-4 border border-destructive/50 rounded-md bg-destructive/10">
              <p className="text-sm text-destructive">{error}</p>
            </div>
          )}

          {isLoading && !fluxStatus && (
            <div className="flex items-center justify-center py-8">
              <IconLoader className="animate-spin mr-2" size={20} />
              <span className="text-muted-foreground">Loading Flux status...</span>
            </div>
          )}

          {!isLoading && !fluxStatus && !error && (
            <div className="text-center py-8 text-muted-foreground">
              No FluxCD HelmRelease found for <span className="font-mono">{name}</span>
            </div>
          )}

          {fluxStatus && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium">Suspended</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center gap-2">
                      <Badge variant={fluxStatus.suspended ? 'destructive' : 'default'}>
                        {fluxStatus.suspended ? 'True' : 'False'}
                      </Badge>
                      {fluxStatus.reconcileDisabled && (
                        <Badge variant="outline">Reconcile Disabled</Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-2">
                      {fluxStatus.suspended ? (
                        <>
                          ⏸️ <strong>Auto-update is paused.</strong> Your image version is pinned and won't upgrade automatically. FluxCD will not reconcile changes from Git until you resume.
                        </>
                      ) : (
                        <>
                          ▶️ <strong>Auto-update is active.</strong> FluxCD will automatically upgrade to the latest image version defined in your Git repository.
                        </>
                      )}
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="pb-3">
                    <CardTitle className="text-sm font-medium">Ready</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Badge variant={fluxStatus.ready ? 'default' : 'secondary'}>
                      {fluxStatus.ready ? 'True' : 'False'}
                    </Badge>
                    <p className="text-xs text-muted-foreground mt-2">
                      {fluxStatus.ready ? (
                        <>
                          ✅ <strong>Reconciliation successful.</strong> Your HelmRelease is synced and healthy. FluxCD has successfully applied the configuration from Git.
                        </>
                      ) : (
                        <>
                          ⚠️ <strong>Reconciliation failed or in progress.</strong> FluxCD is having issues syncing your HelmRelease. Check the Flux conditions for details.
                        </>
                      )}
                    </p>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium">Message</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground">
                    {fluxStatus.message || 'No message'}
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium">Last Sync Time</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm">
                    {fluxStatus.lastSyncTime
                      ? formatDate(fluxStatus.lastSyncTime)
                      : 'Never synced'}
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm font-medium">Details</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Release Name:</span>
                      <span className="font-mono">{fluxStatus.releaseName}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Namespace:</span>
                      <span className="font-mono">{fluxStatus.namespace}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

import { useCallback, useMemo, useState } from 'react'
import { IconAlertCircle, IconLoader } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { ResourceHistory, ResourceType } from '@/types/api'
import { useResourceHistory } from '@/lib/api'
import { formatDate } from '@/lib/utils'

import { Column, SimpleTable } from './simple-table'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './ui/dialog'

interface ResourceHistoryTableProps<T extends ResourceType> {
  resourceType: T
  name: string
  namespace?: string
}

export function ResourceHistoryTable<T extends ResourceType>({
  resourceType,
  name,
  namespace,
}: ResourceHistoryTableProps<T>) {
  const { t } = useTranslation()
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)
  const [selectedHistory, setSelectedHistory] =
    useState<ResourceHistory | null>(null)
  const [isErrorDialogOpen, setIsErrorDialogOpen] = useState(false)

  const {
    data: historyResponse,
    isLoading,
    isError,
    error,
  } = useResourceHistory(
    resourceType,
    namespace ?? '_all',
    name,
    currentPage,
    pageSize
  )

  const history = historyResponse?.data || []
  const total = historyResponse?.pagination?.total || 0

  // Add row numbers to history data (DESC order - newest gets highest number)
  type HistoryWithRowNumber = ResourceHistory & { rowNumber: number }
  const historyWithRowNumbers = useMemo((): HistoryWithRowNumber[] => {
    return history.map((item, index) => ({
      ...item,
      rowNumber: total - ((currentPage - 1) * pageSize) - index
    }))
  }, [history, currentPage, pageSize, total])

  const handleViewError = (item: ResourceHistory) => {
    setSelectedHistory(item)
    setIsErrorDialogOpen(true)
  }

  const getOperationTypeColor = (operationType: string) => {
    switch (operationType.toLowerCase()) {
      case 'edit':
        return 'default' // Blue
      case 'resume':
        return 'success' // Green (we'll add this variant)
      case 'rollback':
        return 'warning' // Amber/Yellow (we'll add this variant)
      case 'restart':
        return 'secondary' // Purple/Gray
      case 'scale':
        return 'info' // Cyan (we'll add this variant)
      case 'suspend':
        return 'orange' // Orange (we'll add this variant)
      case 'create':
        return 'default'
      case 'update':
        return 'secondary'
      case 'delete':
        return 'destructive'
      case 'apply':
        return 'outline'
      default:
        return 'secondary'
    }
  }

  const getOperationTypeLabel = useCallback(
    (operationType: string) => {
      switch (operationType.toLowerCase()) {
        case 'create':
          return t('resourceHistory.create')
        case 'update':
          return t('resourceHistory.update')
        case 'delete':
          return t('resourceHistory.delete')
        case 'apply':
          return t('resourceHistory.apply')
        default:
          return operationType
      }
    },
    [t]
  )

  // History table columns
  const historyColumns = useMemo(
    (): Column<HistoryWithRowNumber>[] => [
      {
        header: 'No',
        accessor: (item: HistoryWithRowNumber) => item.rowNumber,
        cell: (value: unknown) => (
          <div className="font-mono text-sm">{value as number}</div>
        ),
      },
      {
        header: t('resourceHistory.operator'),
        accessor: (item: HistoryWithRowNumber) => item.operator,
        cell: (value: unknown) => (
          <div className="font-medium">
            {(value as { username: string }).username}
            {(value as { provider: string }).provider === 'api_key' && (
              <span className="ml-2 text-xs text-muted-foreground italic">
                apikey
              </span>
            )}
          </div>
        ),
      },
      {
        header: t('resourceHistory.operationTime'),
        accessor: (item: HistoryWithRowNumber) => item.createdAt,
        cell: (value: unknown) => (
          <span className="text-muted-foreground text-sm">
            {formatDate(value as string)}
          </span>
        ),
      },
      {
        header: t('resourceHistory.operationType'),
        accessor: (item: HistoryWithRowNumber) => item.operationType,
        cell: (value: unknown) => {
          const operationType = value as string
          return (
            <Badge variant={getOperationTypeColor(operationType)}>
              {getOperationTypeLabel(operationType)}
            </Badge>
          )
        },
      },
      {
        header: t('resourceHistory.status', 'Status'),
        accessor: (item: HistoryWithRowNumber) => item,
        cell: (value: unknown) => {
          const item = value as HistoryWithRowNumber
          const isSuccess = item.success

          if (!isSuccess) {
            return (
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleViewError(item)}
                disabled={!item.errorMessage}
              >
                <IconAlertCircle className="w-4 h-4 mr-1" />
                {t('resourceHistory.viewError', 'View Error')}
              </Button>
            )
          }

          return (
            <span className="text-sm text-green-600 dark:text-green-400 font-medium">
              ✓ Success
            </span>
          )
        },
      },
    ],
    [getOperationTypeLabel, t]
  )

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <IconLoader className="animate-spin mr-2" />
        {t('resourceHistory.loadingHistory')}
      </div>
    )
  }

  if (isError) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="text-center text-destructive">
            {t('resourceHistory.failedToLoadHistory')}: {error?.message}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>{t('resourceHistory.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <SimpleTable
            data={historyWithRowNumbers}
            columns={historyColumns}
            emptyMessage={t('resourceHistory.noHistoryFound')}
            pagination={{
              enabled: true,
              pageSize,
              showPageInfo: true,
              currentPage,
              onPageChange: setCurrentPage,
              totalCount: total, // Pass total count from API for server-side pagination
            }}
          />
        </CardContent>
      </Card>

      {selectedHistory && (
        <Dialog open={isErrorDialogOpen} onOpenChange={setIsErrorDialogOpen}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>
                {t('resourceHistory.errorDetails', 'Error Details')}
              </DialogTitle>
            </DialogHeader>
            <div className="mt-4">
              <pre className="bg-destructive/10 text-destructive p-4 rounded-md overflow-auto max-h-96 text-sm">
                {selectedHistory.errorMessage ||
                  t(
                    'resourceHistory.noErrorMessage',
                    'no error message available'
                  )}
              </pre>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  )
}

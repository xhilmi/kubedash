import { useCallback, useMemo, useState } from 'react'
import * as Avatar from '@radix-ui/react-avatar'
import {
  IconEdit,
  IconLock,
  IconLockOpen,
  IconPlus,
  IconShieldCheck,
  IconTrash,
  IconUser,
} from '@tabler/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { UserItem } from '@/types/api'
import {
  createPasswordUser,
  deleteUser,
  resetUserPassword,
  setUserEnabled,
  updateUser,
  useUserList,
} from '@/lib/api'
import { formatDate } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { DeleteConfirmationDialog } from '@/components/delete-confirmation-dialog'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

import { Action, ActionTable } from '../action-table'
import { Badge } from '../ui/badge'
import UserRoleAssignment from './user-role-assignment'

// Helper component for truncated role list with popover
function TruncatedRoleList({
  roles,
  maxLength = 30,
}: {
  roles: Array<{ id: number; name: string }>
  maxLength?: number
}) {
  if (!roles || roles.length === 0) {
    return <span className="text-xs text-muted-foreground">-</span>
  }

  const roleNames = roles.map((r) => r.name)
  const joinedText = roleNames.join(', ')

  // If text is short enough, show it directly
  if (joinedText.length <= maxLength) {
    return <span>{joinedText}</span>
  }

  // Otherwise, truncate and show popover
  const truncated = joinedText.slice(0, maxLength) + '...'

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="text-left hover:underline cursor-pointer text-sm"
        >
          {truncated}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-64 max-h-60 overflow-y-auto">
        <div className="space-y-1">
          <p className="text-sm font-medium">All roles ({roles.length}):</p>
          <div className="text-sm text-muted-foreground space-y-1">
            {roles.map((role) => (
              <div key={role.id} className="py-0.5">
                • {role.name}
              </div>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

export function UserManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const { data, isLoading, error } = useUserList(1, 20)

  const getStatusBadge = useCallback(
    (user: UserItem) => {
      if (!user.enabled) {
        return (
          <Badge variant="secondary">{t('common.disabled', 'Disabled')}</Badge>
        )
      }
      return <Badge variant="default">{t('common.enabled', 'Enabled')}</Badge>
    },
    [t]
  )

  const handleToggleEnable = useCallback(
    async (u: UserItem) => {
      await setUserEnabled(u.id, !u.enabled)
      queryClient.invalidateQueries({ queryKey: ['user-list'] })
      toast.success(t('userManagement.messages.updated', 'User updated'))
    },
    [queryClient, t]
  )

  const actions = useMemo<Action<UserItem>[]>(() => {
    return [
      {
        label: (
          <>
            <IconEdit className="h-4 w-4" />
            {t('common.edit', 'Edit')}
          </>
        ),
        onClick: (item) => setEditingUser(item),
      },
      {
        label: '-',
        dynamicLabel: (item) =>
          item.enabled ? (
            <>
              <IconLock className="h-4 w-4" />
              {t('common.disable', 'Disable')}
            </>
          ) : (
            <>
              <IconLockOpen className="h-4 w-4" />
              {t('common.enable', 'Enable')}
            </>
          ),
        onClick: (item) => handleToggleEnable(item),
      },
      {
        label: (
          <div className="inline-flex items-center gap-2 text-destructive">
            <IconTrash className="h-4 w-4" />
            {t('common.delete', 'Delete')}
          </div>
        ),
        onClick: (item) => setDeletingUser(item),
      },
      {
        label: (
          <>
            <IconLock className="h-4 w-4" />
            {t('common.resetPassword', 'Reset Password')}
          </>
        ),
        shouldDisable: (item) => item.provider !== 'password',
        onClick: (item) => handleResetPassword(item),
      },
      {
        label: (
          <>
            <IconShieldCheck className="h-4 w-4" />
            {t('common.assign', 'Assign')}
          </>
        ),
        onClick: (item) => {
          setAssigning(item)
        },
      },
    ]
  }, [handleToggleEnable, t])

  const [editingUser, setEditingUser] = useState<UserItem | null>(null)
  const [deletingUser, setDeletingUser] = useState<UserItem | null>(null)
  const [assigning, setAssigning] = useState<UserItem | null>(null)
  const [showAddDialog, setShowAddDialog] = useState(false)

  const columns = useMemo<ColumnDef<UserItem>[]>(
    () => [
      {
        id: 'id',
        accessorKey: 'id',
        header: 'ID',
        cell: ({ getValue }) => (
          <div className="text-sm text-muted-foreground">
            {String(getValue())}
          </div>
        ),
      },
      {
        id: 'username',
        accessorKey: 'username',
        header: t('username', 'Username'),
        cell: ({ row }) => (
          <div>
            <div className="flex items-center gap-3">
              <button
                type="button"
                onClick={() => setEditingUser(row.original)}
                aria-label={t('userManagement.actions.editUser', 'Edit user')}
                className="p-0 bg-transparent border-0 inline-flex items-center"
              >
                <Avatar.Root className="inline-block">
                  {row.original.avatar_url ? (
                    <Avatar.Image
                      src={row.original.avatar_url}
                      alt={row.original.username}
                      className="h-8 w-8 rounded-full object-cover"
                    />
                  ) : (
                    <Avatar.Fallback className="h-8 w-8 rounded-full bg-muted-foreground text-white flex items-center justify-center">
                      {row.original.username
                        .split(' ')
                        .map((part) => part[0])
                        .join('')
                        .toUpperCase()
                        .slice(0, 2)}
                    </Avatar.Fallback>
                  )}
                </Avatar.Root>
              </button>
              <div className="flex flex-col min-w-0">
                <span className="font-medium truncate">
                  {row.original.username}
                </span>
                {row.original.name && (
                  <span className="text-sm text-muted-foreground truncate">
                    {row.original.name}
                  </span>
                )}
              </div>
            </div>
          </div>
        ),
      },
      {
        id: 'status',
        accessorFn: (row) => row.enabled ? 'Enabled' : 'Disabled',
        header: t('userManagement.table.status', 'Status'),
        cell: ({ row: { original: user } }) => (
          <div className="flex items-center gap-3">{getStatusBadge(user)}</div>
        ),
      },
      {
        id: 'provider',
        accessorFn: (row) => row.provider || '-',
        header: t('userManagement.table.provider', 'Provider'),
        cell: ({ getValue }) => (
          <div className="code">{String(getValue() || '-')}</div>
        ),
      },
      {
        id: 'createdAt',
        accessorKey: 'createdAt',
        header: t('userManagement.table.createdAt', 'Created At'),
        cell: ({ getValue }) => (
          <div className="text-sm text-muted-foreground">
            {formatDate(getValue() as string) || '-'}
          </div>
        ),
      },
      {
        id: 'lastLoginAt',
        accessorKey: 'lastLoginAt',
        header: t('userManagement.table.lastLoginAt', 'Last Login'),
        cell: ({
          row: {
            original: { lastLoginAt },
          },
        }) => (
          <div className="text-sm text-muted-foreground">
            {lastLoginAt ? formatDate(lastLoginAt) : '-'}
          </div>
        ),
      },
      {
        id: 'roles',
        accessorFn: (row) => row.roles?.map((r) => r.name).join(', ') || '-',
        header: t('userManagement.table.roles', 'Roles'),
        cell: ({ row: { original: user } }) => (
          <div className="text-sm text-muted-foreground">
            <TruncatedRoleList roles={user.roles || []} />
          </div>
        ),
      },
    ],
    [getStatusBadge, t]
  )
  const [newUser, setNewUser] = useState({
    username: '',
    name: '',
    password: '',
  })
  const [showResetDialog, setShowResetDialog] = useState<UserItem | null>(null)
  const [resetPasswordValue, setResetPasswordValue] = useState('')

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-list'] })
      toast.success(t('userManagement.messages.deleted', 'User deleted'))
      setDeletingUser(null)
    },
    onError: (err: Error) => {
      toast.error(
        err.message ||
          t('userManagement.messages.deleteError', 'Failed to delete user')
      )
    },
  })

  const createMutation = useMutation({
    mutationFn: (data: { username: string; name?: string; password: string }) =>
      createPasswordUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-list'] })
      toast.success(t('userManagement.messages.created', 'User created'))
      setShowAddDialog(false)
    },
    onError: (err: Error) => {
      toast.error(
        err.message ||
          t('userManagement.messages.createError', 'Failed to create user')
      )
    },
  })

  const resetPasswordMutation = useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) =>
      resetUserPassword(id, password),
    onSuccess: () => {
      toast.success(
        t('userManagement.messages.resetPassword', 'Password reset')
      )
      setShowResetDialog(null)
    },
    onError: (err: Error) => {
      toast.error(
        err.message ||
          t(
            'userManagement.messages.resetPasswordError',
            'Failed to reset password'
          )
      )
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<UserItem> }) =>
      updateUser(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user-list'] })
      toast.success(t('userManagement.messages.updated', 'User updated'))
      setEditingUser(null)
    },
    onError: (err: Error) => {
      toast.error(
        err.message ||
          t('userManagement.messages.updateError', 'Failed to update user')
      )
    },
  })

  const handleDelete = () => {
    if (!deletingUser) return
    deleteMutation.mutate(deletingUser.id)
  }

  const handleResetPassword = (u: UserItem) => {
    setShowResetDialog(u)
  }

  const handleCreateUser = (e: React.FormEvent) => {
    e.preventDefault()
    createMutation.mutate({
      username: newUser.username,
      name: newUser.name,
      password: newUser.password,
    })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingUser) return
    updateMutation.mutate({
      id: editingUser.id,
      data: { name: editingUser.name, avatar_url: editingUser.avatar_url },
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-muted-foreground">
          {t('common.loading', 'Loading...')}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-destructive">
          {t('userManagement.errors.loadFailed', 'Failed to load users')}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <IconUser className="h-5 w-5" />
                {t('userManagement.title', 'User Management')}
              </CardTitle>
            </div>
            <Button onClick={() => setShowAddDialog(true)} className="gap-2">
              <IconPlus className="h-4 w-4" />
              {t('userManagement.actions.add', 'Add Password User')}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <ActionTable
            data={data?.users || []}
            columns={columns}
            actions={actions}
          />

          {(!data || data.users.length === 0) && (
            <div className="text-center py-8 text-muted-foreground">
              <IconUser className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>{t('userManagement.empty.title', 'No users')}</p>
              <p className="text-sm mt-1">
                {t('userManagement.empty.description', 'No users found')}
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit Dialog */}
      <Dialog open={!!editingUser} onOpenChange={() => setEditingUser(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t('userManagement.dialog.editTitle', 'Edit User')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm">
                {t('username', 'Username')}
              </label>
              <Input value={editingUser?.username || ''} disabled />
            </div>
            <div>
              <label className="block text-sm">
                {t('userManagement.table.avatar', 'Avatar URL')}
              </label>
              <Input
                value={editingUser?.avatar_url || ''}
                onChange={(e) =>
                  setEditingUser({
                    ...(editingUser as UserItem),
                    avatar_url: e.target.value,
                  })
                }
              />
            </div>
            <div>
              <label className="block text-sm">
                {t('userManagement.table.name', 'Name')}
              </label>
              <Input
                value={editingUser?.name || ''}
                onChange={(e) =>
                  setEditingUser({
                    ...(editingUser as UserItem),
                    name: e.target.value,
                  })
                }
              />
            </div>
            <DialogFooter>
              <Button type="submit">{t('common.save', 'Save')}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Role assignment dialog */}
      <UserRoleAssignment
        open={!!assigning}
        onOpenChange={(o) => {
          if (!o) setAssigning(null)
        }}
        subject={
          assigning ? { type: 'user', name: assigning.username } : undefined
        }
      />

      {/* Add Password User Dialog */}
      <Dialog open={showAddDialog} onOpenChange={() => setShowAddDialog(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t('userManagement.dialog.addTitle', 'Add Password User')}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreateUser} className="space-y-4">
            <div>
              <label className="block text-sm">
                {t('username', 'Username')}
              </label>
              <Input
                value={newUser.username}
                onChange={(e) =>
                  setNewUser({ ...newUser, username: e.target.value })
                }
              />
            </div>
            <div>
              <label className="block text-sm">
                {t('userManagement.table.name', 'Name')}
              </label>
              <Input
                value={newUser.name}
                onChange={(e) =>
                  setNewUser({ ...newUser, name: e.target.value })
                }
              />
            </div>
            <div>
              <label className="block text-sm">
                {t('common.password', 'Password')}
              </label>
              <Input
                type="password"
                value={newUser.password}
                onChange={(e) =>
                  setNewUser({ ...newUser, password: e.target.value })
                }
              />
            </div>
            <DialogFooter>
              <Button type="submit">{t('common.create', 'Create')}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Reset Password Dialog */}
      <Dialog
        open={!!showResetDialog}
        onOpenChange={() => setShowResetDialog(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {t('userManagement.dialog.resetPassword', 'Reset Password')}
            </DialogTitle>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              if (showResetDialog)
                resetPasswordMutation.mutate({
                  id: showResetDialog.id,
                  password: resetPasswordValue,
                })
            }}
            className="space-y-4"
          >
            <div>
              <label className="block text-sm">
                {t('common.password', 'Password')}
              </label>
              <Input
                name="password"
                type="password"
                value={resetPasswordValue}
                onChange={(e) => setResetPasswordValue(e.target.value)}
              />
            </div>
            <DialogFooter>
              <Button type="submit">{t('common.save', 'Save')}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <DeleteConfirmationDialog
        open={!!deletingUser}
        onOpenChange={() => setDeletingUser(null)}
        onConfirm={handleDelete}
        resourceName={deletingUser?.username || ''}
        resourceType="user"
      />
    </div>
  )
}

export default UserManagement

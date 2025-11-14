import { useRef } from 'react'
import { Editor } from '@monaco-editor/react'
import { formatHex } from 'culori'
import * as yaml from 'js-yaml'
import { editor as monacoEditor } from 'monaco-editor'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { useAppearance } from './appearance-provider'

interface YamlDiffViewerProps {
  /** Modified YAML content (Historical YAML) */
  modified: string
  /** Current YAML content (not used anymore, kept for compatibility) */
  current?: string
  /** Whether the dialog is open */
  open: boolean
  /** Callback when dialog is closed */
  onOpenChange: (open: boolean) => void
  /** Whether history detail is loading */
  isLoadingDetail?: boolean
  /** Dialog title */
  title?: string
  /** Height of the editor */
  height?: number
}

export function YamlDiffViewer({
  modified,
  open,
  onOpenChange,
  isLoadingDetail = false,
  title = 'Resource History',
  height = 600,
}: YamlDiffViewerProps) {
  const { actualTheme, colorTheme } = useAppearance()

  const getCardBackgroundColor = () => {
    const card = getComputedStyle(document.documentElement)
      .getPropertyValue('--background')
      .trim()
    if (!card) {
      return actualTheme === 'dark' ? '#18181b' : '#ffffff'
    }
    return formatHex(card) || (actualTheme === 'dark' ? '#18181b' : '#ffffff')
  }
  const editorRef = useRef<monacoEditor.IStandaloneCodeEditor | null>(null)

  const handleEditorDidMount = (editor: monacoEditor.IStandaloneCodeEditor) => {
    editorRef.current = editor
  }

  // Remove status field from YAML content
  const removeStatusField = (yamlContent: string): string => {
    if (!yamlContent.trim()) return yamlContent

    try {
      const parsed = yaml.load(yamlContent)
      if (parsed && typeof parsed === 'object') {
        // Remove status field recursively
        const removeStatus = (obj: unknown): unknown => {
          if (obj && typeof obj === 'object') {
            if (Array.isArray(obj)) {
              return obj.map(removeStatus)
            } else {
              const result: Record<string, unknown> = {}
              for (const [key, value] of Object.entries(obj)) {
                if (key !== 'status') {
                  result[key] = removeStatus(value)
                }
              }
              return result
            }
          }
          return obj
        }

        const cleaned = removeStatus(parsed)
        return yaml.dump(cleaned, { indent: 2, sortKeys: true })
      }
    } catch (error) {
      console.error('Failed to remove status field from YAML:', error)
    }

    return yamlContent
  }

  // Show only the historical YAML
  const content = removeStatusField(modified)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-6xl sm:!max-w-6xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center justify-between">
            <div className="flex flex-col">
              <span className="text-lg font-bold">{title}</span>
              {!isLoadingDetail && (
                <span className="text-sm font-normal text-muted-foreground mt-1">
                  Historical YAML
                </span>
              )}
            </div>
          </DialogTitle>
        </DialogHeader>
        <div className="flex-1 min-h-0">
          <Editor
            height={height}
            language="yaml"
            beforeMount={(monaco) => {
              const cardBgColor = getCardBackgroundColor()
              monaco.editor.defineTheme(`custom-dark-${colorTheme}`, {
                base: 'vs-dark',
                inherit: true,
                rules: [],
                colors: {
                  'editor.background': cardBgColor,
                },
              })
              monaco.editor.defineTheme(`custom-vs-${colorTheme}`, {
                base: 'vs',
                inherit: true,
                rules: [],
                colors: {
                  'editor.background': cardBgColor,
                },
              })
            }}
            theme={
              actualTheme === 'dark'
                ? `custom-dark-${colorTheme}`
                : `custom-vs-${colorTheme}`
            }
            options={{
              readOnly: true,
              minimap: { enabled: true },
              scrollBeyondLastLine: false,
              wordWrap: 'on',
              folding: true,
              lineNumbers: 'on',
              fontSize: 14,
              fontFamily:
                "'Maple Mono',Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace",
            }}
            onMount={handleEditorDidMount}
            value={content}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}

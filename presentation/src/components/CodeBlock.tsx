import type { ReactNode } from 'react'

type CodeBlockProps = {
  readonly children: string
  readonly title?: string
}

export function CodeBlock({ children, title }: CodeBlockProps): ReactNode {
  return (
    <div className="rounded-lg overflow-hidden border border-edge">
      <div className="bg-surface/80 border-b border-edge px-4 py-2.5 flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-full bg-red-500" />
          <span className="w-3 h-3 rounded-full bg-yellow-500" />
          <span className="w-3 h-3 rounded-full bg-green-500" />
        </div>
        {title ? (
          <span className="text-muted text-xs font-mono">{title}</span>
        ) : null}
      </div>
      <pre className="bg-panel p-4 overflow-x-auto">
        <code className="font-mono text-sm text-fg whitespace-pre">{children}</code>
      </pre>
    </div>
  )
}

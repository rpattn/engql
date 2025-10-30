import { Link } from '@tanstack/react-router'

import { useState } from 'react'
import {
  Boxes,
  Database,
  GitBranch,
  Menu,
  History,
  Download,
  SquareFunction,
  Upload,
  X,
} from 'lucide-react'

export default function Header() {
  const [isOpen, setIsOpen] = useState(false)

  return (
    <>
      <header className="header-surface sticky top-0 z-40 flex items-center gap-4 px-4 py-3 shadow-sm">
        <button
          onClick={() => setIsOpen(true)}
          className="rounded-lg p-2 transition-colors hover:bg-subtle"
          aria-label="Open menu"
        >
          <Menu size={24} />
        </button>
        <Link to="/entities" className="inline-flex items-center gap-3">
          <img src="/logo512.png" alt="TanStack Logo" className="h-9" />
          <span className="text-lg font-semibold tracking-tight">EngQL</span>
        </Link>
      </header>

      {isOpen ? (
        <div
          className="backdrop-overlay fixed inset-0 z-40 transition-opacity"
          aria-hidden
          onClick={() => setIsOpen(false)}
        />
      ) : null}

      <aside
        className={`drawer-surface fixed top-0 left-0 z-50 flex h-full w-80 transform flex-col shadow-2xl transition-transform duration-300 ease-in-out ${
          isOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between border-b border-subtle px-4 py-3">
          <h2 className="text-base font-semibold">Navigation</h2>
          <button
            onClick={() => setIsOpen(false)}
            className="rounded-lg p-2 transition-colors hover:bg-subtle"
            aria-label="Close menu"
          >
            <X size={24} />
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto px-3 py-4">
          <Link
            to="/entity-schemas"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <Database size={20} />
            <span className="font-medium">Entity Schemas</span>
          </Link>

          <Link
            to="/entities"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <Boxes size={20} />
            <span className="font-medium">Entities</span>
          </Link>

          <Link
            to="/transformations"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <GitBranch size={20} />
            <span className="font-medium">Transformations</span>
          </Link>

          {/* Demo Links Start */}

          <Link
            to="/demo/start/server-funcs"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <SquareFunction size={20} />
            <span className="font-medium">Page</span>
          </Link>

          <Link
            to="/ingestion"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <Upload size={20} />
            <span className="font-medium">Ingestion</span>
          </Link>

          <Link
            to="/ingestion/batches"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <History size={20} />
            <span className="font-medium">Batch Monitor</span>
          </Link>

          <Link
            to="/exports"
            onClick={() => setIsOpen(false)}
            className="mb-2 flex items-center gap-3 rounded-lg p-3 font-medium transition-colors hover:bg-subtle"
            activeProps={{
              className:
                'mb-2 flex items-center gap-3 rounded-lg bg-cyan-600 p-3 font-medium text-white transition-colors hover:bg-cyan-600/90',
            }}
          >
            <Download size={20} />
            <span className="font-medium">Exports</span>
          </Link>

          {/* Demo Links End */}
        </nav>
      </aside>
    </>
  )
}

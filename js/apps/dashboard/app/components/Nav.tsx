// Nav is the shared top bar rendered on every authenticated page.
// Slice 5 doesn't have auth — the bar is purely navigational and lets
// the conference audience see the URL structure at a glance.

import { Link } from "@remix-run/react";

interface Crumb {
  to: string;
  label: string;
}

interface NavProps {
  crumbs?: Crumb[];
}

export function Nav({ crumbs = [] }: NavProps) {
  return (
    <header className="border-b border-gray-200 bg-white">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
        <Link
          to="/projects"
          className="text-lg font-semibold text-falseflag-900 hover:text-falseflag-500"
        >
          FalseFlag
        </Link>
        <nav
          aria-label="breadcrumbs"
          className="flex items-center gap-2 text-sm text-falseflag-900/70"
        >
          {crumbs.map((c, i) => (
            <span key={c.to} className="flex items-center gap-2">
              {i > 0 && <span aria-hidden>›</span>}
              <Link to={c.to} className="hover:text-falseflag-500">
                {c.label}
              </Link>
            </span>
          ))}
        </nav>
      </div>
    </header>
  );
}

export function Page({
  children,
  crumbs,
}: {
  children: React.ReactNode;
  crumbs?: Crumb[];
}) {
  return (
    <>
      <Nav crumbs={crumbs} />
      <main className="mx-auto max-w-6xl px-6 py-8">{children}</main>
    </>
  );
}

export function ErrorBanner({ error }: { error?: string }) {
  if (!error) return null;
  return (
    <div
      className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-900"
      data-testid="error"
    >
      API unreachable: {error}
    </div>
  );
}

export function EmptyState({ message }: { message: string }) {
  return (
    <div
      className="rounded-md border border-dashed border-gray-300 p-6 text-center text-sm text-falseflag-900/60"
      data-testid="empty"
    >
      {message}
    </div>
  );
}

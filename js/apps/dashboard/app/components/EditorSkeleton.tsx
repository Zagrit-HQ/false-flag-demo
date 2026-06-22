// Visible during the React.lazy() chunk fetch on the edit route.
// Matches the editor's outer dimensions so the layout doesn't jump
// when Monaco lands.
export function EditorSkeleton() {
  return (
    <div
      className="h-[480px] animate-pulse rounded-md border border-gray-200 bg-gray-50"
      aria-busy="true"
      data-testid="editor-skeleton"
    />
  );
}

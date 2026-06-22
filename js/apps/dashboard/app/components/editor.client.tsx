import Editor, { type OnMount } from "@monaco-editor/react";
import { useEffect, useRef } from "react";

export interface CompileErrorDetail {
  line: number;
  column?: number;
  text: string;
}

interface Props {
  value: string;
  language: "typescript" | "javascript" | "json";
  onChange: (next: string) => void;
  errors?: CompileErrorDetail[];
}

// editor.client.tsx is a Remix 2.x browser-only file (the .client.tsx
// suffix replaces the module with an empty stub during SSR). The
// wrapping route imports this lazily via React.lazy + <Suspense>, so
// Monaco's ~5MB chunks only loads on the edit route.
export default function CodeEditor({
  value,
  language,
  onChange,
  errors,
}: Props) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);
  const monacoRef = useRef<Parameters<OnMount>[1] | null>(null);

  const handleMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    applyMarkers(editor, monaco, errors);
  };

  // Re-apply markers whenever the error list changes (e.g. after a
  // 422 from the publishFlagVersion action).
  useEffect(() => {
    if (editorRef.current && monacoRef.current) {
      applyMarkers(editorRef.current, monacoRef.current, errors);
    }
  }, [errors]);

  return (
    <div className="h-[480px] overflow-hidden rounded-md border border-gray-200">
      <Editor
        height="100%"
        language={language}
        value={value}
        onChange={(v) => onChange(v ?? "")}
        onMount={handleMount}
        theme="vs"
        options={{
          minimap: { enabled: false },
          fontSize: 13,
          scrollBeyondLastLine: false,
        }}
      />
    </div>
  );
}

function applyMarkers(
  editor: Parameters<OnMount>[0],
  monaco: Parameters<OnMount>[1],
  errors: CompileErrorDetail[] = [],
): void {
  const model = editor.getModel();
  if (!model) return;
  monaco.editor.setModelMarkers(
    model,
    "falseflag-compile",
    errors.map((e) => ({
      severity: monaco.MarkerSeverity.Error,
      startLineNumber: e.line,
      endLineNumber: e.line,
      startColumn: e.column ?? 1,
      endColumn: (e.column ?? 1) + 1,
      message: e.text,
    })),
  );
}

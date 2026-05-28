// TraceTree renders the response of GET
// /v1/projects/{slug}/flags/{key}/evaluate-trace. The trace is a tree
// of predicate evaluations rooted at each rule.

export interface TraceNode {
  kind?: string;
  attr?: string;
  matched?: boolean;
  reason?: string;
  rule_id?: string;
  source?: string;
  children?: TraceNode[];
}

export interface TraceRoot {
  rules?: Array<{
    id?: string;
    matched?: boolean;
    when?: TraceNode;
  }>;
}

export function TraceTree({ trace }: { trace: TraceRoot | undefined }) {
  if (!trace) {
    return (
      <div className="text-sm text-falseflag-900/60">
        no trace returned by the API
      </div>
    );
  }
  return (
    <ol className="space-y-3" data-testid="trace-rules">
      {(trace.rules ?? []).map((rule, i) => (
        <li
          key={rule.id ?? i}
          className={`rounded-md border p-3 ${rule.matched ? "border-emerald-300 bg-emerald-50" : "border-gray-200 bg-white"}`}
          data-testid={`trace-rule-${rule.id ?? i}`}
        >
          <div className="flex items-baseline justify-between">
            <code className="text-sm font-medium">
              {rule.id ?? `rule-${i}`}
            </code>
            <span
              className={`text-xs ${rule.matched ? "text-emerald-700" : "text-falseflag-900/60"}`}
            >
              {rule.matched ? "matched" : "skipped"}
            </span>
          </div>
          {rule.when && <PredicateNode node={rule.when} depth={0} />}
        </li>
      ))}
    </ol>
  );
}

function PredicateNode({ node, depth }: { node: TraceNode; depth: number }) {
  const indent = { paddingLeft: `${depth * 16}px` };
  const tone = node.matched
    ? "text-emerald-700"
    : node.matched === false
      ? "text-falseflag-900/60"
      : "text-falseflag-900/80";
  return (
    <div className="mt-1" style={indent}>
      <div className={`text-xs ${tone}`}>
        <code>{node.kind ?? "?"}</code>
        {node.attr && <span> {node.attr}</span>}
        {node.source && (
          <span className="ml-1 text-falseflag-900/60">
            <code>{node.source}</code>
          </span>
        )}
        {node.matched != null && (
          <span className="ml-2">{node.matched ? "✓" : "✗"}</span>
        )}
      </div>
      {(node.children ?? []).map((c, idx) => (
        <PredicateNode
          key={`${c.kind ?? "node"}-${idx}`}
          node={c}
          depth={depth + 1}
        />
      ))}
    </div>
  );
}

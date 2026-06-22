// StrategyBadge shows the configuration strategy of a project or flag
// version as a colored pill. Keep the palette in sync with the Tailwind
// extension in tailwind.config.ts (strategy-json / strategy-cel /
// strategy-typescript).

export type Strategy = "json" | "cel" | "typescript";

const tones: Record<Strategy, string> = {
  json: "bg-strategy-json/15 text-strategy-json border-strategy-json/40",
  cel: "bg-strategy-cel/15 text-strategy-cel border-strategy-cel/40",
  typescript:
    "bg-strategy-typescript/15 text-strategy-typescript border-strategy-typescript/40",
};

export function StrategyBadge({ strategy }: { strategy: Strategy }) {
  return (
    <span
      data-testid={`strategy-${strategy}`}
      className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium ${tones[strategy] ?? ""}`}
    >
      {strategy}
    </span>
  );
}

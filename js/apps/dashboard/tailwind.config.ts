import type { Config } from "tailwindcss";

export default {
  content: ["./app/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        falseflag: {
          50: "#f4f7fb",
          500: "#3b82f6",
          900: "#0b1f33",
        },
        strategy: {
          json: "#2563eb",
          cel: "#7c3aed",
          typescript: "#0891b2",
        },
      },
    },
  },
  plugins: [],
} satisfies Config;

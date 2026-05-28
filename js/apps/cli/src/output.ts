// Shared output helpers for CLI subcommands.

export async function okOrLog(
  call: Promise<{ status: number; data: unknown }>,
  err: NodeJS.WritableStream,
): Promise<unknown | undefined> {
  try {
    const res = await call;
    if (res.status >= 200 && res.status < 300) {
      return res.data;
    }
    err.write(`error: HTTP ${res.status}: ${JSON.stringify(res.data)}\n`);
    return undefined;
  } catch (e) {
    err.write(`error: ${(e as Error).message}\n`);
    return undefined;
  }
}

export function printItems<T>(
  out: NodeJS.WritableStream,
  items: T[] | undefined,
  fmt: (item: T) => string,
): void {
  if (!items || items.length === 0) {
    out.write("(no items)\n");
    return;
  }
  for (const item of items) {
    out.write(`${fmt(item)}\n`);
  }
}

export function writeJSON(
  out: NodeJS.WritableStream,
  value: unknown,
  pretty = true,
): void {
  out.write(`${JSON.stringify(value, null, pretty ? 2 : 0)}\n`);
}

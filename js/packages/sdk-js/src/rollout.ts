// FNV-1a 64-bit. Identical to internal/eval/rollout.go so Go and JS
// always land in the same bucket for a given (salt, value) pair.

const FNV_OFFSET = 0xcbf29ce484222325n;
const FNV_PRIME = 0x100000001b3n;
const MASK_64 = 0xffffffffffffffffn;

function fnv1a64(bytes: Uint8Array): bigint {
  let hash = FNV_OFFSET;
  for (const byte of bytes) {
    hash ^= BigInt(byte);
    hash = (hash * FNV_PRIME) & MASK_64;
  }
  return hash;
}

const encoder = new TextEncoder();

export function rolloutBucket(salt: string, attrValue: string): number {
  const saltBytes = encoder.encode(salt);
  const sepBytes = new Uint8Array([0x3a]); // ':'
  const valBytes = encoder.encode(attrValue);
  const combined = new Uint8Array(saltBytes.length + 1 + valBytes.length);
  combined.set(saltBytes, 0);
  combined.set(sepBytes, saltBytes.length);
  combined.set(valBytes, saltBytes.length + 1);
  const h = fnv1a64(combined);
  return Number(h % 10000n);
}

export function inBucket(
  salt: string,
  attrValue: string,
  percent: number,
): boolean {
  if (percent <= 0) return false;
  if (percent >= 100) return true;
  return rolloutBucket(salt, attrValue) < percent * 100;
}

/**
 * Zero-dependency pure TypeScript QR Code SVG generator
 * Supports Byte mode (UTF-8) with Error Correction Level L
 */

const EXP: number[] = new Array(512);
const LOG: number[] = new Array(256);
(() => {
  let x = 1;
  for (let i = 0; i < 255; i++) {
    EXP[i] = x;
    EXP[i + 255] = x;
    LOG[x] = i;
    x = (x << 1) ^ (x & 128 ? 285 : 0);
  }
})();

function gfMul(x: number, y: number): number {
  return x === 0 || y === 0 ? 0 : EXP[LOG[x] + LOG[y]];
}

function rsPoly(numEc: number): number[] {
  let poly = [1];
  for (let i = 0; i < numEc; i++) {
    const root = EXP[i];
    const next = new Array(poly.length + 1).fill(0);
    for (let j = 0; j < poly.length; j++) {
      next[j] ^= poly[j];
      next[j + 1] ^= gfMul(poly[j], root);
    }
    poly = next;
  }
  return poly;
}

function rsCalc(data: number[], numEc: number): number[] {
  const poly = rsPoly(numEc);
  const res = [...data, ...new Array(numEc).fill(0)];
  for (let i = 0; i < data.length; i++) {
    const lead = res[i];
    if (lead !== 0) {
      for (let j = 0; j < poly.length; j++) {
        res[i + j] ^= gfMul(poly[j], lead);
      }
    }
  }
  return res.slice(data.length);
}

// Byte capacities for EC Level L
const CAPACITIES_L = [0, 17, 32, 53, 78, 106, 134, 154, 192, 230, 271];
const TOTAL_CODEWORDS = [0, 26, 44, 70, 100, 134, 172, 196, 242, 292, 346];
const EC_CODEWORDS = [0, 7, 10, 15, 20, 26, 36, 40, 48, 60, 72];
const ALIGN_POS = [
  [],
  [],
  [6, 18],
  [6, 22],
  [6, 26],
  [6, 30],
  [6, 34],
  [6, 22, 38],
  [6, 24, 42],
  [6, 26, 46],
  [6, 28, 50],
];

export function generateQRCodeSVGPath(text: string): { path: string; size: number } {
  // 1. Encode text to UTF-8 bytes
  const encoder = new TextEncoder();
  let bytes = Array.from(encoder.encode(text));

  // Determine Version
  let version = 1;
  while (version <= 10 && bytes.length > CAPACITIES_L[version]) {
    version++;
  }
  if (version > 10) {
    version = 10;
    bytes = bytes.slice(0, CAPACITIES_L[10]);
  }

  const totalCodewords = TOTAL_CODEWORDS[version];
  const ecCodewords = EC_CODEWORDS[version];
  const dataCodewords = totalCodewords - ecCodewords;

  // 2. Build BitBuffer: Byte Mode (0100) + Count Indicator + Data + Terminator
  const bits: number[] = [];
  const pushBits = (val: number, len: number) => {
    for (let i = len - 1; i >= 0; i--) {
      bits.push((val >> i) & 1);
    }
  };

  pushBits(0b0100, 4); // Byte Mode
  pushBits(bytes.length, version < 10 ? 8 : 16);
  for (const b of bytes) {
    pushBits(b, 8);
  }

  // Terminator
  const maxBits = dataCodewords * 8;
  const termLen = Math.min(4, maxBits - bits.length);
  pushBits(0, termLen);

  // Pad to byte
  while (bits.length % 8 !== 0) {
    bits.push(0);
  }

  // Pad bytes 0xEC, 0x11
  const padPatterns = [0xec, 0x11];
  let padIdx = 0;
  while (bits.length < maxBits) {
    pushBits(padPatterns[padIdx % 2], 8);
    padIdx++;
  }

  // Convert bits to data codeword array
  const rawData: number[] = [];
  for (let i = 0; i < bits.length; i += 8) {
    let byteVal = 0;
    for (let j = 0; j < 8; j++) {
      byteVal = (byteVal << 1) | bits[i + j];
    }
    rawData.push(byteVal);
  }

  // 3. Compute Reed-Solomon EC Codewords
  const ec = rsCalc(rawData, ecCodewords);
  const finalCodewords = [...rawData, ...ec];

  // 4. Matrix Initialization
  const size = version * 4 + 17;
  const matrix: (boolean | null)[][] = Array.from({ length: size }, () =>
    new Array(size).fill(null)
  );
  const reserved: boolean[][] = Array.from({ length: size }, () =>
    new Array(size).fill(false)
  );

  // Finder Patterns
  const addFinder = (r: number, c: number) => {
    for (let dr = -1; dr <= 7; dr++) {
      for (let dc = -1; dc <= 7; dc++) {
        const nr = r + dr;
        const nc = c + dc;
        if (nr >= 0 && nr < size && nc >= 0 && nc < size) {
          reserved[nr][nc] = true;
          if (dr >= 0 && dr <= 6 && dc >= 0 && dc <= 6) {
            matrix[nr][nc] =
              dr === 0 ||
              dr === 6 ||
              dc === 0 ||
              dc === 6 ||
              (dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4);
          } else {
            matrix[nr][nc] = false;
          }
        }
      }
    }
  };

  addFinder(0, 0);
  addFinder(0, size - 7);
  addFinder(size - 7, 0);

  // Timing Patterns
  for (let i = 8; i < size - 8; i++) {
    reserved[6][i] = true;
    matrix[6][i] = i % 2 === 0;
    reserved[i][6] = true;
    matrix[i][6] = i % 2 === 0;
  }

  // Alignment Patterns (version >= 2)
  const coords = ALIGN_POS[version];
  for (const r of coords) {
    for (const c of coords) {
      if (reserved[r][c]) continue;
      for (let dr = -2; dr <= 2; dr++) {
        for (let dc = -2; dc <= 2; dc++) {
          reserved[r + dr][c + dc] = true;
          matrix[r + dr][c + dc] =
            Math.max(Math.abs(dr), Math.abs(dc)) !== 1;
        }
      }
    }
  }

  // Format info area reservation
  for (let i = 0; i < 9; i++) {
    reserved[8][i] = true;
    reserved[i][8] = true;
  }
  for (let i = 0; i < 8; i++) {
    reserved[8][size - 1 - i] = true;
    reserved[size - 1 - i][8] = true;
  }
  reserved[size - 8][8] = true;
  matrix[size - 8][8] = true;

  // 5. Fill Data Codewords in Zigzag Order
  const allBits: number[] = [];
  for (const cw of finalCodewords) {
    for (let b = 7; b >= 0; b--) {
      allBits.push((cw >> b) & 1);
    }
  }

  let bitIdx = 0;
  let upward = true;
  for (let right = size - 1; right > 0; right -= 2) {
    if (right === 6) right--; // skip vertical timing column
    const cols = [right, right - 1];
    const rows = upward
      ? Array.from({ length: size }, (_, i) => size - 1 - i)
      : Array.from({ length: size }, (_, i) => i);

    for (const r of rows) {
      for (const c of cols) {
        if (!reserved[r][c]) {
          const bit = bitIdx < allBits.length ? allBits[bitIdx++] : 0;
          // Apply Standard Mask 0: (row + col) % 2 === 0
          const mask = (r + c) % 2 === 0;
          matrix[r][c] = (bit === 1) !== mask;
        }
      }
    }
    upward = !upward;
  }

  // 6. Format Information (EC Level L, Mask 0 = 0b111011111000100)
  const formatBits = [1, 1, 1, 0, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0];
  // Top-left
  const tlCoords: [number, number][] = [
    [8, 0], [8, 1], [8, 2], [8, 3], [8, 4], [8, 5],
    [8, 7], [8, 8], [7, 8], [5, 8], [4, 8], [3, 8],
    [2, 8], [1, 8], [0, 8],
  ];
  for (let i = 0; i < 15; i++) {
    const [r, c] = tlCoords[i];
    matrix[r][c] = formatBits[i] === 1;
  }
  // Bottom-left & Top-right
  for (let i = 0; i < 7; i++) {
    matrix[size - 1 - i][8] = formatBits[i] === 1;
  }
  for (let i = 7; i < 15; i++) {
    matrix[8][size - 15 + i] = formatBits[i] === 1;
  }

  // 7. Generate SVG path
  let path = "";
  for (let r = 0; r < size; r++) {
    for (let c = 0; c < size; c++) {
      if (matrix[r][c]) {
        path += `M${c},${r}h1v1h-1z `;
      }
    }
  }

  return { path, size };
}

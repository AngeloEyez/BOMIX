/**
 * @file src/main/utils/phase-sorter.js
 * @description Phase sorting utility
 * @module utils/phase-sorter
 */

/**
 * 預設 Phase 順序
 * @type {string[]}
 */
export const DEFAULT_PHASE_ORDER = [
  'RFI',
  'RFP',
  'RFQ, RFx',
  'DB, EVT',
  'SI, DVT',
  'PV, PVT',
  'TLD, PRD',
  'MVB, MP'
];

/**
 * Parses the user-defined phase order from JSON array string.
 * @param {string|null} phaseOrderJson
 * @returns {string[]} The array of phases
 */
export function getPhaseOrderArray(phaseOrderJson) {
  if (!phaseOrderJson) return DEFAULT_PHASE_ORDER;
  try {
    const parsed = JSON.parse(phaseOrderJson);
    if (Array.isArray(parsed) && parsed.length > 0) {
      return parsed;
    }
  } catch (e) {
    console.error('Failed to parse phase_order JSON', e);
  }
  return DEFAULT_PHASE_ORDER;
}

/**
 * Splits a phase string into its alphabetic base and numeric suffix.
 * e.g., 'DB1' -> { base: 'DB', num: 1 }
 * e.g., 'DB' -> { base: 'DB', num: -1 } (no number means it comes before those with numbers)
 *
 * @param {string} phaseName
 * @returns {{base: string, num: number}}
 */
export function parsePhaseName(phaseName) {
  if (!phaseName) return { base: '', num: -1 };

  const match = phaseName.match(/^([A-Za-z]+)(\d*)$/);
  if (match) {
    const base = match[1].toUpperCase();
    const numStr = match[2];
    const num = numStr ? parseInt(numStr, 10) : -1; // -1 ensures it sorts before 0, 1, 2...
    return { base, num };
  }

  // Fallback if the format is not matched
  return { base: phaseName.toUpperCase(), num: -1 };
}

/**
 * Gets the precedence (index) of a base phase string against the ordered list.
 * Note: Each element in orderArray can be a comma-separated list of synonyms (e.g. "DB, EVT").
 *
 * @param {string} basePhase
 * @param {string[]} orderArray
 * @returns {number} The index (lower is higher precedence). Returns Infinity if not found.
 */
export function getPhaseBaseIndex(basePhase, orderArray) {
  for (let i = 0; i < orderArray.length; i++) {
    const synonyms = orderArray[i].split(',').map(s => s.trim().toUpperCase());
    if (synonyms.includes(basePhase)) {
      return i;
    }
  }
  return Infinity; // Unknown phase goes to the end
}

/**
 * Sorts an array of BOM revisions based on phase-version-suffix logic.
 *
 * 1. Phase Base (by order array)
 * 2. Phase Num (no num < 0 < 1 < ...)
 * 3. Version string (descending usually, or parsing numbers?)
 *    - Example: 0.1 > 0.2 (Wait, usually 0.2 is newer.
 *      User requested: "例如資料庫有以下BOM: DB1-0.1, DB1-0.2, SI-0.2... SI-0.2的前一版是 DB1-0.2"
 *      This implies sorting in chronological order: DB1-0.1, DB1-0.2, SI-0.2
 *      So earlier version numbers come first.
 * 4. Suffix (No suffix comes BEFORE suffix: e.g. PV-0.1 > PV-0.1-1)
 *
 * @param {Array<Object>} boms
 * @param {string[]} orderArray
 * @returns {Array<Object>} Sorted BOMs
 */
export function sortBoms(boms, orderArray) {
  return [...boms].sort((a, b) => {
    // 1. Phase comparison
    const phaseA = parsePhaseName(a.phase_name);
    const phaseB = parsePhaseName(b.phase_name);

    const indexA = getPhaseBaseIndex(phaseA.base, orderArray);
    const indexB = getPhaseBaseIndex(phaseB.base, orderArray);

    if (indexA !== indexB) {
      return indexA - indexB;
    }

    // 2. Phase Num comparison
    if (phaseA.num !== phaseB.num) {
      return phaseA.num - phaseB.num;
    }

    // 3. Version comparison (String comparison usually works for "0.1" vs "0.2")
    // For proper semver/numeric comparison, we should split by '.'
    const verCompare = compareVersions(a.version, b.version);
    if (verCompare !== 0) {
      return verCompare;
    }

    // 4. Suffix comparison
    // No suffix means it is earlier than having a suffix.
    if (!a.suffix && b.suffix) return -1;
    if (a.suffix && !b.suffix) return 1;
    if (a.suffix && b.suffix) {
      // both have suffix, string compare
      return a.suffix.localeCompare(b.suffix);
    }

    // Equal
    return 0;
  });
}

/**
 * Compares two version strings (e.g., "0.1", "1.0.2")
 * Returns -1 if v1 < v2, 1 if v1 > v2, 0 if equal.
 */
function compareVersions(v1, v2) {
  if (!v1) return -1;
  if (!v2) return 1;

  const parts1 = v1.split('.').map(p => parseInt(p, 10) || 0);
  const parts2 = v2.split('.').map(p => parseInt(p, 10) || 0);

  const len = Math.max(parts1.length, parts2.length);
  for (let i = 0; i < len; i++) {
    const num1 = parts1[i] || 0;
    const num2 = parts2[i] || 0;
    if (num1 !== num2) {
      return num1 - num2;
    }
  }
  return 0;
}

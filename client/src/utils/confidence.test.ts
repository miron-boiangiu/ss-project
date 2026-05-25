import { describe, it, expect } from 'vitest';
import { confidenceTier, type ConfidenceTier } from './confidence';

describe('confidenceTier', () => {
  // Boundary cases pinned per the PR 3 spec. The 95 boundary separates
  // green/yellow; the 80 boundary separates yellow/red and matches the
  // server's default OCR_REVIEW_THRESHOLD. 0 and undefined both collapse to
  // gray ("not extracted") so the UI can distinguish absent from low.
  const cases: Array<[number | undefined, ConfidenceTier]> = [
    [94.99, 'yellow'],
    [95.0, 'green'],
    [95.01, 'green'],
    [79.99, 'red'],
    [80.0, 'yellow'],
    [80.01, 'yellow'],
    [0, 'gray'],
    [undefined, 'gray'],
    [100, 'green'],
  ];

  for (const [input, expected] of cases) {
    it(`${input} → ${expected}`, () => {
      expect(confidenceTier(input)).toBe(expected);
    });
  }
});

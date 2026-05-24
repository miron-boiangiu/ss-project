// Confidence tier helper for displaying per-field OCR confidence in the UI.
//
// The backend returns a `field_confidences: Record<string, number>` map on every
// Photo (since PR 1). A field absent from the map — or present with value 0 —
// means "we have no useful information about this field"; see
// docs/ocr-extraction.md "Confidence model" section. We surface that as a
// distinct gray tier (rendered as "—") so reviewers can tell "low confidence"
// apart from "not extracted".
//
// The three numeric tiers match the server's default OCR_REVIEW_THRESHOLD of 80:
// yellow-or-worse means "below the review threshold". 95 is the spec's
// nominal threshold and is reserved for the green tier.

export type ConfidenceTier = 'green' | 'yellow' | 'red' | 'gray';

export function confidenceTier(score: number | undefined): ConfidenceTier {
  if (score === undefined || score === 0) return 'gray';
  if (score >= 95) return 'green';
  if (score >= 80) return 'yellow';
  return 'red';
}

// Static Tailwind class maps. Must be string literals at the lookup site so
// Tailwind's purger keeps them in the production bundle — do NOT rewrite as
// `border-${tier}-400` template strings (the purger can't see those).
export const TIER_BORDER_CLASS: Record<ConfidenceTier, string> = {
  green: 'border-l-4 border-green-400',
  yellow: 'border-l-4 border-yellow-400',
  red: 'border-l-4 border-red-400',
  gray: 'border-l-4 border-gray-300',
};

export const TIER_BADGE_CLASS: Record<ConfidenceTier, string> = {
  green: 'bg-green-100 text-green-800',
  yellow: 'bg-yellow-100 text-yellow-800',
  red: 'bg-red-100 text-red-800',
  gray: 'bg-gray-100 text-gray-500',
};

// Static badge text. Only the gray slot has a meaningful constant value ("—");
// for the numeric tiers the consumer formats the actual percentage via
// `confidenceLabel` below, since a static Record can't carry a per-call number.
export const TIER_LABEL: Record<ConfidenceTier, string> = {
  green: '',
  yellow: '',
  red: '',
  gray: '—',
};

// Format the badge text for a given confidence score. Returns TIER_LABEL.gray
// ("—") for the gray tier (undefined or 0), and a rounded percentage like
// "92%" for the three numeric tiers.
export function confidenceLabel(score: number | undefined): string {
  const tier = confidenceTier(score);
  if (tier === 'gray') return TIER_LABEL.gray;
  return `${Math.round(score as number)}%`;
}

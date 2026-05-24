// Structured per-field grid for the photo zoom modal.
//
// Renders one row per extracted text field in reading order matching the
// underlying Fișă de Aptitudine form, with a colored left border + inline
// confidence badge driven by the `field_confidences` map on the Photo.
// Checkboxes/booleans deliberately get a grouped plain-text summary (no
// border, no badge) — per the PR 1 a-lite scope, only regex text fields
// carry per-field confidence.
//
// The component emits only the field stack; it doesn't wrap itself in a
// card/shadow so it can be slotted into the existing modal body.

import React from 'react';
import {
  TIER_BADGE_CLASS,
  TIER_BORDER_CLASS,
  confidenceLabel,
  confidenceTier,
} from '../../utils/confidence';
import type { Photo } from '../../pages/photosPage';

// snake_case key shared between the Photo interface and the field_confidences
// map. Restricting to `keyof Photo` lets TypeScript catch typos at compile time
// without forcing every consumer to widen to `string`.
type FieldKey = Extract<keyof Photo, string>;

interface FieldDef {
  key: FieldKey;
  label: string;
}

// Sections rendered above the grouped checkbox summary.
const SECTIONS_BEFORE_CHECKBOXES: FieldDef[][] = [
  // Issuing medical unit
  [
    { key: 'unitate_medicala', label: 'Unitate medicală:' },
    { key: 'adresa_unitate_medicala', label: 'Adresa:' },
    { key: 'telefon_unitate_medicala', label: 'Telefon:' },
  ],
  // Form metadata
  [{ key: 'numar_fisa', label: 'Număr fișă:' }],
  // Employer
  [
    { key: 'societate_unitate', label: 'Societate:' },
    { key: 'adresa_angajator', label: 'Adresa angajator:' },
    { key: 'telefon_angajator', label: 'Telefon angajator:' },
  ],
  // Personal
  [
    { key: 'nume', label: 'Nume:' },
    { key: 'prenume', label: 'Prenume:' },
    { key: 'cnp', label: 'CNP:' },
  ],
  // Professional
  [
    { key: 'profesie_functie', label: 'Profesie:' },
    { key: 'loc_de_munca', label: 'Loc de muncă:' },
  ],
];

// Sections rendered below the grouped checkbox summary.
const SECTIONS_AFTER_CHECKBOXES: FieldDef[][] = [
  // Recommendations
  [{ key: 'recomandari', label: 'Recomandări:' }],
  // Dates
  [
    { key: 'data', label: 'Data:' },
    { key: 'data_urm_examinari', label: 'Data urm. examinări:' },
  ],
];

// Read a string-typed field off the photo, treating non-strings (undefined,
// booleans on a wrong key) as empty so the row falls into the gray "—" tier.
function readString(photo: Photo, key: FieldKey): string {
  const raw = photo[key];
  return typeof raw === 'string' ? raw : '';
}

interface FieldRowProps {
  photo: Photo;
  field: FieldDef;
}

const FieldRow: React.FC<FieldRowProps> = ({ photo, field }) => {
  const value = readString(photo, field.key);
  const score = photo.field_confidences?.[field.key];
  const isEmpty = value.trim() === '';

  // Empty values force gray regardless of the confidence map. Reviewers see
  // "—" for both the value and the badge so absent fields are unambiguous.
  const tier = isEmpty ? confidenceTier(undefined) : confidenceTier(score);
  const display = isEmpty ? '—' : value;
  const badgeText = isEmpty ? '—' : confidenceLabel(score);

  return (
    <div className={`${TIER_BORDER_CLASS[tier]} pl-3 py-2 flex items-start justify-between gap-3`}>
      <div className="flex-1 min-w-0">
        <div className="text-xs font-medium text-gray-500 dark:text-gray-400">{field.label}</div>
        <div
          className={`text-sm break-words ${
            isEmpty
              ? 'text-gray-400 dark:text-gray-500'
              : 'text-gray-900 dark:text-gray-100'
          }`}
        >
          {display}
        </div>
      </div>
      <span
        className={`shrink-0 inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${TIER_BADGE_CLASS[tier]}`}
      >
        {badgeText}
      </span>
    </div>
  );
};

interface CheckboxSummaryProps {
  photo: Photo;
}

// Grouped checkbox summary: two plain lines, no border, no badge. PR 1's
// a-lite scope intentionally excludes booleans from per-field confidence;
// reviewers see the derived textual values (tip_control, aviz_medical) and
// trust the parser's own boolean roll-up.
const CheckboxSummary: React.FC<CheckboxSummaryProps> = ({ photo }) => {
  const tipControl = photo.tip_control?.trim() ? photo.tip_control : '—';
  const avizMedical = photo.aviz_medical?.trim() ? photo.aviz_medical : '—';
  return (
    <div className="py-2 text-sm text-gray-800 dark:text-gray-200 space-y-1">
      <div>
        <span className="font-medium">Tip control:</span> {tipControl}
      </div>
      <div>
        <span className="font-medium">Aviz medical:</span> {avizMedical}
      </div>
    </div>
  );
};

interface MedicalFieldGridProps {
  photo: Photo;
}

const MedicalFieldGrid: React.FC<MedicalFieldGridProps> = ({ photo }) => {
  return (
    <div className="space-y-2">
      {SECTIONS_BEFORE_CHECKBOXES.flat().map((field) => (
        <FieldRow key={field.key} photo={photo} field={field} />
      ))}
      <CheckboxSummary photo={photo} />
      {SECTIONS_AFTER_CHECKBOXES.flat().map((field) => (
        <FieldRow key={field.key} photo={photo} field={field} />
      ))}
    </div>
  );
};

export default MedicalFieldGrid;

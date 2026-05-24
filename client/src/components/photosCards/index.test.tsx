import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import PhotoCard from './index';
import type { Photo } from '../../pages/photosPage';

// Minimal Photo factory — only fills the fields PhotoCard reads on the card
// surface (id, presigned_url, timestamp, text). Everything else stays
// undefined so we also exercise the component's tolerance for legacy rows
// missing the post-PR-1 fields.
function makePhoto(overrides: Partial<Photo>): Photo {
  return {
    id: 'p1',
    timestamp: '2026-01-20T12:00:00Z',
    image_type: 'image/png',
    presigned_url: 'http://example.com/p1.png',
    device_id: 'd1',
    text: 'sample extracted text',
    ...overrides,
  };
}

describe('PhotoCard "Needs Review" pill', () => {
  it('renders the pill when needs_review is true', () => {
    const photo = makePhoto({ needs_review: true });
    render(<PhotoCard photo={photo} />);
    expect(screen.getByText('Needs Review')).toBeInTheDocument();
  });

  it('does not render the pill when needs_review is false', () => {
    const photo = makePhoto({ needs_review: false });
    render(<PhotoCard photo={photo} />);
    expect(screen.queryByText('Needs Review')).toBeNull();
  });
});

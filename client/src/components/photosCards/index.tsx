import React, { useState } from 'react';
import fallbackImage from '../../assets/photo-fallback.svg';
import MedicalFieldGrid from '../medicalFieldGrid';
import type { Photo } from '../../pages/photosPage';

// PhotoCard now consumes the full Photo object (PR 3) instead of piecemeal
// imageUrl / extractedText / altText props. The data fields collapse into
// `photo`; `isAdmin` and `onDelete` stay as separate parent-controlled props
// since they're auth/callback wiring, not photo data.
interface PhotoCardProps {
  photo: Photo;
  isAdmin?: boolean;
  onDelete?: (photoId: string) => void;
}

const PhotoCard: React.FC<PhotoCardProps> = ({ photo, isAdmin = false, onDelete }) => {
  const [isZoomed, setIsZoomed] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  // Derived view of the photo for rendering. altText was previously passed in
  // by the parent; computing it locally keeps the call site simple and the
  // formatting consistent across pages.
  const imageUrl = photo.presigned_url;
  const extractedText = photo.text ?? '';
  const altText = `Photo from ${new Date(photo.timestamp).toLocaleDateString()}`;
  const needsReview = photo.needs_review === true;

  const handleImageError = () => {
    setImageError(true);
  };

  const toggleZoom = () => {
    setIsZoomed(!isZoomed);
  };

  // Handle click outside the zoomed image to close it
  const handleModalClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      setIsZoomed(false);
    }
  };

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = async () => {
    setIsDeleting(true);
    if (onDelete) {
      await onDelete(photo.id);
    }
    setShowDeleteConfirm(false);
    setIsDeleting(false);
  };

  return (
    <>
      <div className="bg-white rounded-lg shadow-md overflow-hidden transition-all hover:shadow-lg relative">
        <div className="relative h-48 cursor-pointer" onClick={toggleZoom}>
          <img
            src={imageError ? fallbackImage : imageUrl}
            alt={altText}
            onError={handleImageError}
            className="w-full h-full object-cover"
          />
          {/* "Needs Review" pill — top-left, mirrors the top-right delete button. */}
          {needsReview && (
            <span
              className="absolute top-2 left-2 bg-yellow-400 text-yellow-900 text-xs font-medium px-2 py-1 rounded-full"
              title="OCR confidence below review threshold"
            >
              Needs Review
            </span>
          )}
          {isAdmin && (
            <button
              onClick={handleDeleteClick}
              className="absolute top-2 right-2 bg-red-500 hover:bg-red-600 text-white rounded-full p-2 shadow-lg transition-all duration-200 opacity-80 hover:opacity-100"
              title="Delete photo"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          )}
        </div>

        {extractedText && (
          <div className="p-3 border-t border-gray-100">
            <p className="text-sm text-gray-600 truncate">{extractedText}</p>
          </div>
        )}

        {/* Delete confirmation dialog */}
        {showDeleteConfirm && (
          <div className="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center">
            <div className="bg-white rounded-lg p-4 m-4 shadow-xl">
              <p className="text-gray-800 mb-4">Delete this photo?</p>
              <div className="flex gap-2 justify-center">
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="px-4 py-2 bg-gray-300 hover:bg-gray-400 rounded-md transition-colors"
                  disabled={isDeleting}
                >
                  Cancel
                </button>
                <button
                  onClick={handleConfirmDelete}
                  className="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-md transition-colors"
                  disabled={isDeleting}
                >
                  {isDeleting ? 'Deleting...' : 'Delete'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Improved zoom modal overlay with animations and better styling */}
      {isZoomed && (
        <div
          className="fixed inset-0 bg-black bg-opacity-60 backdrop-blur-sm flex items-center justify-center z-50 transition-opacity duration-300 ease-in-out"
          onClick={handleModalClick}
        >
          <div
            className="relative bg-white dark:bg-gray-900 rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden transform transition-all duration-300 ease-in-out animate-scaleIn"
          >
            {/* Top bar — absolutely positioned so it floats above the scrolling body. */}
            <div className="absolute top-0 right-0 left-0 bg-gradient-to-b from-black/50 to-transparent h-20 z-10 flex justify-between items-start p-4">
              <div className="text-white text-lg font-medium truncate pr-10">{altText}</div>
              <button
                className="bg-white/20 hover:bg-white/40 text-white rounded-full p-2 backdrop-blur-sm transition-all duration-200"
                onClick={toggleZoom}
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* Scrollable body — image, structured grid, collapsed raw OCR. */}
            <div className="overflow-y-auto max-h-[90vh]">
              <div className="p-4 pt-20">
                <img
                  src={imageError ? fallbackImage : imageUrl}
                  alt={altText}
                  className="max-w-full max-h-[65vh] object-contain mx-auto rounded-md"
                />
              </div>

              <div className="bg-gray-50 dark:bg-gray-800 p-6 border-t border-gray-100 dark:border-gray-700">
                <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">Extracted Fields</h3>
                <MedicalFieldGrid photo={photo} />
              </div>

              {extractedText && (
                <details className="bg-white dark:bg-gray-900 p-6 border-t border-gray-100 dark:border-gray-700">
                  <summary className="cursor-pointer text-sm font-medium text-gray-700 dark:text-gray-300">
                    Show raw OCR text
                  </summary>
                  <pre className="mt-3 whitespace-pre-wrap text-sm text-gray-600 dark:text-gray-400 font-mono">
                    {extractedText}
                  </pre>
                </details>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
};

export default PhotoCard;

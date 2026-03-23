// TODO: Implement authentication - See docs/AUTH_IMPLEMENTATION.md
// This component currently allows all access. To implement real auth:
// 1. Uncomment the authentication checks below
// 2. Redirect unauthenticated users to login page

import React from 'react';
import { Outlet } from 'react-router-dom';

interface ProtectedRouteProps {
  authRequired: boolean;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ authRequired: _authRequired }) => {
  // TODO: Implement authentication checks
  // Currently bypassing all auth - allowing access to all routes

  // Original implementation (uncomment when implementing real auth):
  /*
  import { Navigate } from 'react-router-dom';
  import { useAuth } from '../contexts/AuthContext';
  
  const { isLoggedIn, loading } = useAuth();
  
  // While auth state is loading, show nothing (or could add a loading spinner here)
  if (loading) {
    return <div className="flex justify-center items-center min-h-[60vh]">
      <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500"></div>
    </div>;
  }
  
  // If auth is required and user is not logged in, redirect to login
  if (authRequired && !isLoggedIn) {
    return <Navigate to="/login" replace />;
  }
  
  // If auth is not required and user is logged in, redirect to root
  if (!authRequired && isLoggedIn) {
    return <Navigate to="/" replace />;
  }
  */

  // Always render the children (no authentication check)
  return <Outlet />;
};

export default ProtectedRoute; 
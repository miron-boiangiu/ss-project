import React, { createContext, useState, useContext, useEffect } from 'react';

interface AuthContextType {
  isLoggedIn: boolean;
  token: string | null;
  loading: boolean;
  isAdmin: boolean;
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType>({
  isLoggedIn: false,
  token: null,
  loading: true,
  isAdmin: false,
  login: () => { },
  logout: () => { },
});

export const useAuth = () => useContext(AuthContext);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [loading, setLoading] = useState(true);
  const [isAdmin, setIsAdmin] = useState(false);

  useEffect(() => {
    const storedToken = localStorage.getItem('token');
    if (storedToken) {
      setToken(storedToken);
      setIsLoggedIn(true);
      try {
        const payload = JSON.parse(atob(storedToken.split('.')[1]));
        setIsAdmin(payload.role === 'admin');
      } catch {
        setIsAdmin(false);
      }
    }
    setLoading(false);
  }, []);

  const login = (newToken: string) => {
    localStorage.setItem('token', newToken);
    setToken(newToken);
    setIsLoggedIn(true);
    try {
      const payload = JSON.parse(atob(newToken.split('.')[1]));
      setIsAdmin(payload.role === 'admin');
    } catch {
      setIsAdmin(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setIsLoggedIn(false);
    setIsAdmin(false);
  };

  const value = {
    token,
    isLoggedIn,
    loading,
    isAdmin,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export default AuthContext; 
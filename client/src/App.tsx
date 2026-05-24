import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet, useNavigate } from 'react-router-dom';
import Navbar from './components/navbar';
import HomePage from './pages/homePage';
import LoginPage from './pages/loginPage';
import RegisterPage from './pages/register';
import PhotosPage from './pages/photosPage';
import DevicesPage from './pages/devicesPage';
import StatisticsPage from './pages/statisticsPage';
import ProtectedRoute from './components/ProtectedRoute';
import { AuthProvider, useAuth } from './contexts/AuthContext';

const Layout = () => {
  const navigate = useNavigate();
  const { isLoggedIn, logout } = useAuth();

  // --- LOGICA PENTRU DARK MODE ---
  const [isDarkMode, setIsDarkMode] = useState(false);

  useEffect(() => {
    // La încărcare, verificăm dacă utilizatorul are deja o preferință salvată
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'dark') {
      setIsDarkMode(true);
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, []);

  const toggleTheme = () => {
    setIsDarkMode((prev: boolean) => {
      const newTheme = !prev;
      if (newTheme) {
        document.documentElement.classList.add('dark');
        localStorage.setItem('theme', 'dark');
      } else {
        document.documentElement.classList.remove('dark');
        localStorage.setItem('theme', 'light');
      }
      return newTheme;
    });
  };
  // -------------------------------

  // Left-side buttons (only shown when logged in)
  const leftButtons = isLoggedIn
    ? [
      {
        text: 'Photos',
        variant: 'secondary' as const,
        onClick: () => navigate('/photos')
      },
      {
        text: 'Devices',
        variant: 'secondary' as const,
        onClick: () => navigate('/devices')
      },
      {
        text: 'Statistics',
        variant: 'secondary' as const,
        onClick: () => navigate('/statistics')
      }
    ]
    : [];

  // Right-side buttons (different based on login status)
  const rightButtons = isLoggedIn
    ? [
      {
        // Butonul de temă pentru utilizatori logați
        text: isDarkMode ? '☀️ Light' : '🌙 Dark',
        variant: 'secondary' as const,
        onClick: toggleTheme
      },
      {
        text: 'Logout',
        variant: 'outline' as const,
        onClick: () => {
          logout();
          navigate('/');
        }
      }
    ]
    : [
      {
        // Butonul de temă pentru vizitatori
        text: isDarkMode ? '☀️ Light' : '🌙 Dark',
        variant: 'secondary' as const,
        onClick: toggleTheme
      },
      {
        text: 'Login',
        variant: 'outline' as const,
        onClick: () => navigate('/login')
      },
      {
        text: 'Register',
        variant: 'primary' as const,
        onClick: () => navigate('/register')
      }
    ];

  return (
    // Am adăugat și aici o clasă pentru a ne asigura că fundalul întregii pagini se schimbă
    <div className="min-h-screen bg-white text-black dark:bg-gray-900 dark:text-white transition-colors duration-300">
      <Navbar
        title="Security of Systems - First Force"
        leftButtons={leftButtons}
        rightButtons={rightButtons}
      />
      <div className="pt-16 px-4">
        <Outlet />
      </div>
    </div>
  );
};

const App = () => {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Common layout for all routes */}
          <Route element={<Layout />}>
            {/* Public routes - accessible to everyone */}
            <Route path="/" element={<HomePage />} />

            {/* Auth routes - only for non-authenticated users */}
            <Route element={<ProtectedRoute authRequired={false} />}>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
            </Route>

            {/* Protected routes - only for authenticated users */}
            <Route element={<ProtectedRoute authRequired={true} />}>
              <Route path="/photos" element={<PhotosPage />} />
              <Route path="/devices" element={<DevicesPage />} />
              <Route path="/statistics" element={<StatisticsPage />} />
            </Route>

            {/* Fallback route */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
};

export default App;
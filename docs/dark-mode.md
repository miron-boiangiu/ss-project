# Dark/Light Mode Toggle

## Overview

The application supports a manual dark/light mode toggle persisted to `localStorage`. The implementation uses Tailwind's `class`-based dark mode strategy: toggling adds or removes the `.dark` class on `document.documentElement`, and all components use `dark:` variant prefixes for their dark-theme styles.

The toggle button is rendered in the `Navbar` via the `Layout` component in `App.tsx`, and is accessible to both authenticated and anonymous users.

## Configuration & Tooling

### Vite + Tailwind Bridge (`client/postcss.config.js`)

PostCSS acts as the bridge between Vite and TailwindCSS. Created from scratch to ensure Tailwind directives are processed by the Vite bundler:

```js
export default {
  plugins: {
    '@tailwindcss/postcss': {},
    autoprefixer: {},
  },
}
```

### Tailwind Configuration (`client/tailwind.config.js`)

```js
export default {
  darkMode: 'class',    // enables manual .dark class toggling
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: { extend: {} },
  plugins: [],
}
```

The `darkMode: 'class'` setting tells Tailwind to apply `dark:` variants only when a parent element has the `.dark` class — no system-preference detection.

### Base Styles (`client/src/index.css`)

```css
@import "tailwindcss";
@tailwind base;
@tailwind components;
@tailwind utilities;
@custom-variant dark (&:where(.dark, .dark *));

:root {
  color-scheme: light;
}
.dark {
  color-scheme: dark;
}
```

The `@custom-variant dark (&:where(.dark, .dark *));` directive ensures the manual class-based toggle works with the latest Tailwind version. The `.dark` class also sets `color-scheme: dark` so native form elements and scrollbars render in dark mode.

## Theme Toggle Logic (`client/src/App.tsx`)

Theme state is managed in the `Layout` component:

```typescript
const [isDarkMode, setIsDarkMode] = useState(false);

useEffect(() => {
  const savedTheme = localStorage.getItem('theme');
  if (savedTheme === 'dark') {
    setIsDarkMode(true);
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
}, []);

const toggleTheme = () => {
  setIsDarkMode((prev) => {
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
```

On mount, the saved preference is read from `localStorage` and applied. The toggle function flips the state, updates `document.documentElement.classList`, and persists the choice.

The root wrapper applies global transition colors:

```html
<div className="min-h-screen bg-white text-black dark:bg-gray-900 dark:text-white transition-colors duration-300">
```

### Button Label

The toggle button displays the current state:

| State | Button Text |
|---|---|
| Light mode active | `🌙 Dark` |
| Dark mode active | `☀️ Light` |

## Per-Component Dark Mode Classes

### Navbar (`client/src/components/navbar/index.tsx`)

| Element | Light | Dark |
|---|---|---|
| Container `<nav>` | `bg-sky-50` | `dark:bg-gray-900 dark:border-b dark:border-gray-700` |
| Title `<h1>` | `text-sky-700` | `dark:text-sky-300` |
| Title hover | `hover:text-sky-800` | `dark:hover:text-sky-200` |

### Authentication Forms (LoginPage, RegisterPage)

| Element | Light | Dark |
|---|---|---|
| Form container | `bg-white` | `dark:bg-gray-800` |
| Input fields | `border-gray-300` | `dark:bg-gray-700 dark:text-white dark:border-gray-600` |
| Labels | `text-sky-700` | `dark:text-sky-400` |

### Home Page (`client/src/pages/homePage/index.tsx`)

| Element | Light | Dark |
|---|---|---|
| Title | `text-sky-700` | `dark:text-sky-400` |
| Subtitle | `text-gray-600` | `dark:text-gray-300` |
| Feature cards | `bg-sky-50` | `dark:bg-gray-800` |
| Card icons | `bg-sky-100` | `dark:bg-gray-700` |
| Card icon SVG | `text-sky-700` | `dark:text-sky-400` |
| Card headings | `text-sky-700` | `dark:text-sky-300` |
| Card text | `text-gray-600` | `dark:text-gray-400` |

### Photos Page (`client/src/pages/photosPage/index.tsx`)

| Element | Light | Dark |
|---|---|---|
| Search/date filter panel | `bg-white` | `dark:bg-gray-800` |
| Filter labels | `text-gray-700` | `dark:text-sky-700` |
| Input/select fields | `border-gray-300` | `dark:bg-gray-700 dark:text-white dark:border-gray-600` |
| Device badge | `bg-gray-50 border-gray-200` | `dark:bg-gray-700 dark:border-gray-600` |
| Device badge text | `text-gray-700` | `dark:text-gray-200` |
| Photo list container | `bg-gray-50` | `dark:bg-gray-800` |
| Empty state text | `text-gray-500` | `dark:text-gray-400` |

### Statistics Page (`client/src/pages/statisticsPage/index.tsx`)

| Element | Light | Dark |
|---|---|---|
| Chart container | `bg-white` | `dark:bg-gray-800` |
| Chart heading | `text-gray-800` | `dark:text-white` |
| View toggle bar | `bg-gray-100` | `dark:bg-gray-700` |
| Active toggle | `bg-white text-sky-600` | `dark:bg-gray-600 dark:text-sky-400` |
| Inactive toggle | `text-gray-500` | `dark:text-gray-300` |
| Search panel | `bg-white` | `dark:bg-gray-800` |
| Date/device labels | `text-gray-700` | `dark:text-gray-200` |
| Date inputs | `border-gray-300` | `dark:bg-gray-700 dark:text-white dark:border-gray-600` |

### Devices Page (`client/src/pages/devicesPage/index.tsx`)

| Element | Light | Dark |
|---|---|---|
| Device list container | `bg-gray-50` | `dark:bg-gray-800` |

## Dependency Lockfile

`yarn.lock` is committed to version control. After installing PostCSS plugins (`@tailwindcss/postcss`, `autoprefixer`), the lockfile was tracked to guarantee deterministic dependency resolution across all developer environments.

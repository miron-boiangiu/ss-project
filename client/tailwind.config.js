/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class', // ACEASTA ESTE LINIA MAGICĂ
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
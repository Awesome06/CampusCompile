/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Custom dark theme colors for our arena
        dark: {
          bg: '#1a1a1a',
          surface: '#2d2d2d',
          border: '#404040',
          text: '#e0e0e0',
          accent: '#3b82f6', // Tailwind blue-500
          success: '#10b981', // Tailwind emerald-500
          error: '#ef4444', // Tailwind red-500
        }
      }
    },
  },
  plugins: [],
}
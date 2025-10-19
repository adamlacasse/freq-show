/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{html,ts}', './server.ts'],
  theme: {
    extend: {
      colors: {
        'freq-ink': '#0f172a',
        'freq-midnight': '#111827',
        'freq-rose': '#fb7185',
        'freq-amber': '#fbbf24',
        'freq-teal': '#2dd4bf',
        'freq-cream': '#f5f1e0',
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
        display: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
      },
      boxShadow: {
        'freq-card': '0 25px 50px -20px rgba(15, 23, 42, 0.55)',
      },
      backgroundImage: {
        'freq-noise': 'radial-gradient(circle at 20% 20%, rgba(251, 113, 133, 0.18), transparent 60%), radial-gradient(circle at 80% 0%, rgba(59, 130, 246, 0.16), transparent 55%)',
      },
    },
  },
  plugins: [],
};


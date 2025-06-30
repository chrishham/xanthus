/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js"
  ],
  safelist: [
    // Height classes used in JavaScript template strings
    'h-32',
    'h-96',
    // Add other JavaScript-only classes here as needed
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}


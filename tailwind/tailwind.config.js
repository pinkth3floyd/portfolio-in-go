/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["../web/templates/**/*.html"],
  darkMode: ["class"],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: { "2xl": "1400px" },
    },
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" },
        secondary: { DEFAULT: "hsl(var(--secondary))", foreground: "hsl(var(--secondary-foreground))" },
        destructive: { DEFAULT: "hsl(var(--destructive))", foreground: "hsl(var(--destructive-foreground))" },
        muted: { DEFAULT: "hsl(var(--muted))", foreground: "hsl(var(--muted-foreground))" },
        accent: { DEFAULT: "hsl(var(--accent))", foreground: "hsl(var(--accent-foreground))" },
        cyber: {
          background: "#0a0a0a",
          purple: "#a020f0",
          cyan: "#00ffff",
          pink: "#ff007f",
          green: "#00ff00",
          text: "#d9d9d9",
          lavender: "#e0ccff",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      keyframes: {
        glow: {
          "0%, 100%": { textShadow: "0 0 5px #a020f0, 0 0 15px #a020f0, 0 0 20px #a020f0" },
          "50%": { textShadow: "0 0 10px #ff007f, 0 0 30px #ff007f, 0 0 40px #ff007f" },
        },
        float: {
          "0%, 100%": { transform: "translateY(0)" },
          "50%": { transform: "translateY(-10px)" },
        },
        glitch: {
          "0%, 100%": { transform: "translate(0)" },
          "10%": { transform: "translate(-5px, 5px)" },
          "20%": { transform: "translate(-5px, -5px)" },
          "30%": { transform: "translate(5px, 5px)" },
          "40%": { transform: "translate(5px, -5px)" },
          "50%": { transform: "translate(-5px, 5px)" },
          "60%": { transform: "translate(-5px, -5px)" },
          "70%": { transform: "translate(5px, 5px)" },
          "80%": { transform: "translate(-5px, -5px)" },
          "90%": { transform: "translate(5px, -5px)" },
        },
        scanline: {
          "0%": { transform: "translateY(0%)" },
          "100%": { transform: "translateY(100%)" },
        },
        "spin-slow": {
          from: { transform: "rotate(0deg)" },
          to: { transform: "rotate(360deg)" },
        },
      },
      animation: {
        glow: "glow 2s ease-in-out infinite",
        float: "float 6s ease-in-out infinite",
        glitch: "glitch 0.5s ease-in-out infinite",
        scanline: "scanline 8s linear infinite",
        "spin-slow": "spin-slow 8s linear infinite",
      },
      fontFamily: {
        orbitron: ["Orbitron", "sans-serif"],
        rajdhani: ["Rajdhani", "sans-serif"],
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";
import path from "path";
import { componentTagger } from "lovable-tagger";
import { VitePWA } from "vite-plugin-pwa";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    host: "::",
    port: 5173,
    hmr: {
      overlay: false,
      host: "localhost",
    },
  },
  plugins: [
    react(),
    mode === "development" && componentTagger(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.ico", "manifest.json", "placeholder.svg"],
      manifest: {
        name: "SkillZone",
        short_name: "SkillZone",
        description: "Skill verification and opportunity platform — works offline",
        start_url: "/",
        display: "standalone",
        background_color: "#ffffff",
        theme_color: "#1a6dea",
        orientation: "portrait-primary",
        icons: [
          {
            src: "/favicon.ico",
            sizes: "64x64",
            type: "image/x-icon",
          },
        ],
      },
      workbox: {
        // Precache all built assets (JS chunks, CSS, fonts, images)
        globPatterns: ["**/*.{js,css,html,ico,png,svg,woff2}"],
        // Network-first for API calls — falls back to cache when offline
        runtimeCaching: [
          {
            // Match /api/* regardless of host — works for localhost AND LAN IPs
            urlPattern: /\/api\//,
            handler: "NetworkFirst",
            options: {
              cacheName: "api-cache",
              networkTimeoutSeconds: 5,
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },
      devOptions: {
        // Enable SW in dev so we can test offline mode in Chrome DevTools
        enabled: true,
      },
    }),
  ].filter(Boolean),
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // React core — changes rarely, long cache life
          "vendor-react": ["react", "react-dom", "react-router-dom"],
          // UI component library — large but stable
          "vendor-ui": ["@radix-ui/react-dialog", "@radix-ui/react-dropdown-menu",
            "@radix-ui/react-popover", "@radix-ui/react-select", "@radix-ui/react-tooltip",
            "lucide-react", "react-hot-toast"],
          // Heavy QR / camera deps — only used on Events page
          "vendor-qr": ["qrcode", "html5-qrcode"],
          // Offline data layer
          "vendor-dexie": ["dexie"],
        },
      },
    },
  },}));

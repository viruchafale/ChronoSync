import { defineConfig } from "vite";
export default defineConfig({
  root: "tests/visual",
  server: {
    host: "127.0.0.1",
    port: 1421,
    strictPort: true,
    fs: { allow: ["."] },
  },
});

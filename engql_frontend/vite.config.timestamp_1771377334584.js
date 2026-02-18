// vite.config.ts
import { defineConfig } from "@tanstack/react-start/config";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact from "@vitejs/plugin-react";
import viteTsConfigPaths from "vite-tsconfig-paths";
import tailwindcss from "@tailwindcss/vite";
var vite_config_default = defineConfig({
  plugins: [
    viteTsConfigPaths({
      projects: ["./tsconfig.json"]
    }),
    tailwindcss(),
    tanstackStart(),
    viteReact()
  ]
});
export {
  vite_config_default as default
};

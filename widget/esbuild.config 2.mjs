import { build } from "esbuild";

await build({
  entryPoints: ["src/widget.ts"],
  bundle: true,
  minify: true,
  format: "iife",
  globalName: "CRMChatWidget",
  outfile: "dist/widget.js",
  target: ["es2020"],
  sourcemap: true,
});

console.log("Widget bundle built successfully: dist/widget.js");

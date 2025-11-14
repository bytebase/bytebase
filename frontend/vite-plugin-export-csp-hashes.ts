import { createHash } from "crypto";
import { readdirSync, readFileSync, statSync, writeFileSync } from "fs";
import { dirname, join, resolve } from "path";
import { fileURLToPath } from "url";
import type { Plugin } from "vite";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * Extract inline script content from HTML
 */
function extractInlineScripts(html: string): string[] {
  const scripts: string[] = [];
  // Match <script>...</script> but NOT <script src="...">
  // This regex matches script tags without src attribute
  const scriptRegex = /<script(?![^>]*\ssrc=)([^>]*)>([\s\S]*?)<\/script>/gi;

  let match;
  while ((match = scriptRegex.exec(html)) !== null) {
    const scriptContent = match[2].trim();
    if (scriptContent) {
      scripts.push(scriptContent);
    }
  }

  return scripts;
}

/**
 * Recursively find all HTML files in a directory
 */
function findHtmlFiles(dir: string): string[] {
  const files: string[] = [];

  try {
    const entries = readdirSync(dir);

    for (const entry of entries) {
      const fullPath = join(dir, entry);
      const stat = statSync(fullPath);

      if (stat.isDirectory()) {
        files.push(...findHtmlFiles(fullPath));
      } else if (entry.endsWith(".html")) {
        files.push(fullPath);
      }
    }
  } catch (error) {
    // Ignore errors (e.g., permission denied)
  }

  return files;
}

/**
 * Compute SHA-256 hash of a string
 */
function computeSha256(content: string): string {
  return createHash("sha256").update(content).digest("base64");
}

/**
 * Vite plugin to export CSP hashes from @vitejs/plugin-legacy and inline scripts
 *
 * This plugin runs after the build and:
 * 1. Collects CSP hashes from @vitejs/plugin-legacy
 * 2. Scans built HTML files for inline scripts
 * 3. Generates SHA-256 hashes for all inline scripts
 * 4. Exports everything to a JSON file that the backend reads
 */
export function exportCspHashes(): Plugin {
  let outDir = "";

  return {
    name: "export-csp-hashes",
    apply: "build",
    enforce: "post", // Run after other plugins including @vitejs/plugin-legacy

    configResolved(config) {
      outDir = config.build.outDir;
    },

    async closeBundle() {
      try {
        const allHashes = new Set<string>();
        const inlineScriptSources: Array<{
          file: string;
          content: string;
          hash: string;
        }> = [];

        // 1. Get CSP hashes from @vitejs/plugin-legacy
        try {
          const legacyPlugin = await import("@vitejs/plugin-legacy");
          const cspHashes = legacyPlugin.cspHashes;

          if (cspHashes && cspHashes.length > 0) {
            cspHashes.forEach((hash: string) => {
              allHashes.add(`'sha256-${hash}'`);
            });
            console.log(
              `✓ Loaded ${cspHashes.length} hashes from @vitejs/plugin-legacy`
            );
          }
        } catch (error) {
          console.warn(
            "⚠️  Could not load hashes from @vitejs/plugin-legacy:",
            error
          );
        }

        // 2. Hash dynamically-injected scripts (e.g., iframe content)
        const dynamicScripts = [
          "src/components/MarkdownEditor/resize-observer.ts",
        ];
        console.log(
          `✓ Computing hashes for ${dynamicScripts.length} dynamically-injected scripts...`
        );

        for (const scriptPath of dynamicScripts) {
          try {
            const fullPath = resolve(__dirname, scriptPath);
            const scriptContent = readFileSync(fullPath, "utf-8");
            const hash = computeSha256(scriptContent);
            const cspHash = `'sha256-${hash}'`;

            allHashes.add(cspHash);
            inlineScriptSources.push({
              file: scriptPath,
              content:
                scriptContent.length > 60
                  ? scriptContent.substring(0, 60) + "..."
                  : scriptContent,
              hash: cspHash,
            });
            console.log(`  ✓ ${scriptPath} -> ${cspHash}`);
          } catch (error) {
            console.warn(`  ⚠️  Failed to hash ${scriptPath}:`, error);
          }
        }

        // 3. Scan built HTML files for inline scripts
        const htmlFiles = findHtmlFiles(outDir);
        console.log(
          `✓ Scanning ${htmlFiles.length} HTML files for inline scripts...`
        );

        for (const htmlFile of htmlFiles) {
          const html = readFileSync(htmlFile, "utf-8");
          const inlineScripts = extractInlineScripts(html);

          for (const scriptContent of inlineScripts) {
            const hash = computeSha256(scriptContent);
            const cspHash = `'sha256-${hash}'`;

            allHashes.add(cspHash);
            inlineScriptSources.push({
              file: htmlFile.replace(outDir, "").replace(/\\/g, "/"),
              content:
                scriptContent.length > 60
                  ? scriptContent.substring(0, 60) + "..."
                  : scriptContent,
              hash: cspHash,
            });
          }
        }

        // Get plugin version
        let pluginVersion = "unknown";
        try {
          const pkgPath = resolve(
            __dirname,
            "node_modules/@vitejs/plugin-legacy/package.json"
          );
          const pkgData = readFileSync(pkgPath, "utf-8");
          const pkg = JSON.parse(pkgData);
          pluginVersion = pkg.version;
        } catch {
          pluginVersion = "unknown";
        }

        // Prepare the output
        const output = {
          scriptHashes: Array.from(allHashes),
          generatedAt: new Date().toISOString(),
          pluginVersion,
          sources: inlineScriptSources,
        };

        // Write to backend server directory
        const outputPath = join(outDir, "csp-hashes.json");
        writeFileSync(outputPath, JSON.stringify(output, null, 2));

        console.log("✓ CSP hashes exported to:", outputPath);
        console.log(`  Total script hashes: ${output.scriptHashes.length}`);
        console.log(`  Inline scripts found: ${inlineScriptSources.length}`);

        if (inlineScriptSources.length > 0) {
          console.log("\n  Inline script sources:");
          inlineScriptSources.forEach((src, i) => {
            console.log(`    ${i + 1}. ${src.file}`);
            console.log(`       Content: ${src.content}`);
            console.log(`       Hash: ${src.hash}`);
          });
        }
      } catch (error) {
        console.error("❌ Failed to export CSP hashes:", error);
        // Don't fail the build, just warn
      }
    },
  };
}

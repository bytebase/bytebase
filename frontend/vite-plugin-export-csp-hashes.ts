import type { Plugin } from 'vite';
import { writeFileSync, readFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/**
 * Vite plugin to export CSP hashes from @vitejs/plugin-legacy
 *
 * This plugin runs after the legacy plugin and exports the CSP hashes
 * to a JSON file that the backend can read to construct the CSP header.
 */
export function exportCspHashes(): Plugin {
  return {
    name: 'export-csp-hashes',
    apply: 'build',
    enforce: 'post', // Run after other plugins including @vitejs/plugin-legacy

    async closeBundle() {
      try {
        // Import the CSP hashes from @vitejs/plugin-legacy
        const legacyPlugin = await import('@vitejs/plugin-legacy');
        const cspHashes = legacyPlugin.cspHashes;

        if (!cspHashes || cspHashes.length === 0) {
          console.warn('⚠️  No CSP hashes found from @vitejs/plugin-legacy');
          return;
        }

        // Get plugin version
        let pluginVersion = 'unknown';
        try {
          const pkgPath = resolve(__dirname, 'node_modules/@vitejs/plugin-legacy/package.json');
          const pkgData = readFileSync(pkgPath, 'utf-8');
          const pkg = JSON.parse(pkgData);
          pluginVersion = pkg.version;
        } catch {
          pluginVersion = 'unknown';
        }

        // Prepare the output
        const output = {
          scriptHashes: cspHashes.map((hash: string) => `'sha256-${hash}'`),
          generatedAt: new Date().toISOString(),
          pluginVersion
        };

        // Write to backend server directory
        const outputPath = resolve(__dirname, '../backend/server/dist/csp-hashes.json');
        writeFileSync(outputPath, JSON.stringify(output, null, 2));

        console.log('✓ CSP hashes exported to:', outputPath);
        console.log(`  Script hashes (${output.scriptHashes.length}):`, output.scriptHashes);
      } catch (error) {
        console.error('❌ Failed to export CSP hashes:', error);
        // Don't fail the build, just warn
      }
    }
  };
}

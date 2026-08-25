// Builds the loadable unpacked extension packages into dist/chrome and
// dist/firefox: esbuild bundles the shared core into per-browser IIFE
// background + popup scripts, then the manifest, popup assets, and generated
// icons are copied next to them. There is no bundler config beyond esbuild —
// the extension stays dependency-light by design.
import { build } from 'esbuild'
import { cpSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { generateIcon } from './gen-icons.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const dist = join(root, 'dist')
const popupEntry = join(root, 'src/popup/popup.tsx')

rmSync(dist, { recursive: true, force: true })

for (const browser of ['chrome', 'firefox']) {
    const outDir = join(dist, browser)
    mkdirSync(join(outDir, 'icons'), { recursive: true })

    await build({
        entryPoints: [join(root, `src/${browser}/background.ts`), popupEntry],
        bundle: true,
        format: 'iife',
        platform: 'browser',
        target: 'chrome105',
        outdir: outDir,
        entryNames: '[name]',
        // The popup is React (matching the dashboard); esbuild injects the
        // react/jsx-runtime automatic transform.
        jsx: 'automatic',
        define: { 'process.env.NODE_ENV': '"production"' },
        logLevel: 'info',
    })

    cpSync(join(root, `src/${browser}/manifest.json`), join(outDir, 'manifest.json'))
    cpSync(join(root, 'src/popup/popup.html'), join(outDir, 'popup.html'))
    cpSync(join(root, 'src/popup/popup.css'), join(outDir, 'popup.css'))
    for (const size of [16, 48, 128]) {
        writeFileSync(join(outDir, 'icons', `icon${size}.png`), generateIcon(size))
    }
}

console.log('browser-ext: built dist/chrome and dist/firefox (unpacked)')

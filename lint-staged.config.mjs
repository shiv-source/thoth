import path from 'node:path'

// golangci-lint needs whole packages to typecheck — passing individual files
// breaks cross-file references (e.g. Version() in a sibling file), so map
// staged files to their deduped package dirs.
export default {
  'web/src/**/*.{ts,tsx}': ['pnpm --filter thoth-web exec eslint --fix', 'prettier --write'],
  'web/src/**/*.{js,mjs,cjs,json,md,yml,yaml,css,html}': ['prettier --write'],
  '**/*.go': (files) => {
    const dirs = [...new Set(files.map((f) => path.dirname(f)))]
    return [`golangci-lint run --fix ${dirs.join(' ')}`]
  },
}

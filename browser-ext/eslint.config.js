import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import eslintConfigPrettier from 'eslint-config-prettier'

export default tseslint.config(
    { ignores: ['dist'] },
    js.configs.recommended,
    {
        files: ['**/*.{ts,tsx}'],
        extends: [...tseslint.configs.recommendedTypeChecked],
        languageOptions: {
            globals: globals.browser,
            // projectService auto-resolves the tsconfig; pin the root so a run
            // that starts above this package (e.g. from the workspace root)
            // cannot confuse it with web/'s tsconfig projects.
            parserOptions: { projectService: true, tsconfigRootDir: import.meta.dirname },
        },
        rules: {
            '@typescript-eslint/no-explicit-any': 'error',
            '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
            // Async test doubles must return Promises to satisfy their
            // interfaces, so an await-less async body is not a smell here.
            '@typescript-eslint/require-await': 'off',
        },
    },
    {
        // The build/icon scripts and the e2e mock server run under Node, not
        // a browser.
        files: ['scripts/**/*.mjs', 'e2e/**/*.mjs'],
        languageOptions: { globals: globals.node },
    },
    // Prettier owns formatting — disable the eslint rules that conflict with it.
    eslintConfigPrettier,
)

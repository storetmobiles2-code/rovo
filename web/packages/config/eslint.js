// Shared ESLint flat config for the rovo web workspace.
// Usage (web/eslint.config.js): `export default rovoEslintConfig();`
import js from '@eslint/js';
import jsxA11y from 'eslint-plugin-jsx-a11y';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';
import tseslint from 'typescript-eslint';

/** @returns {import('eslint').Linter.Config[]} */
export function rovoEslintConfig() {
  return tseslint.config(
    {
      ignores: [
        '**/node_modules/**',
        '**/dist/**',
        '**/dev-dist/**',
        '**/coverage/**',
        '**/schema.d.ts',
        '**/*.d.ts',
      ],
    },
    js.configs.recommended,
    ...tseslint.configs.recommended,
    {
      files: ['**/*.{ts,tsx}'],
      languageOptions: {
        ecmaVersion: 2023,
        sourceType: 'module',
        globals: { ...globals.browser },
      },
      plugins: {
        'react-hooks': reactHooks,
        'react-refresh': reactRefresh,
      },
      rules: {
        'react-hooks/rules-of-hooks': 'error',
        'react-hooks/exhaustive-deps': 'warn',
        '@typescript-eslint/no-unused-vars': [
          'error',
          { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
        ],
        '@typescript-eslint/consistent-type-imports': ['error', { fixStyle: 'inline-type-imports' }],
        '@typescript-eslint/no-explicit-any': 'error',
        'no-console': ['warn', { allow: ['warn', 'error'] }],
        eqeqeq: ['error', 'always'],
        // Apps never import from each other, and nobody deep-imports package internals.
        'no-restricted-imports': [
          'error',
          {
            patterns: [
              {
                group: ['**/apps/*', '**/apps/**'],
                message: 'Apps must not import from other apps. Move shared code to packages/.',
              },
              {
                group: ['@rovo/*/src/*'],
                message: 'Import from the package entry point (e.g. @rovo/ui), not its src/.',
              },
            ],
          },
        ],
      },
    },
    {
      files: ['**/*.tsx'],
      ...jsxA11y.flatConfigs.recommended,
      languageOptions: {
        ...jsxA11y.flatConfigs.recommended.languageOptions,
        globals: { ...globals.browser },
      },
    },
    {
      files: ['apps/*/src/**/*.tsx'],
      rules: {
        'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      },
    },
    {
      files: ['**/*.{js,mjs,cjs}', '**/vite.config.ts', '**/vitest.config.ts', '**/scripts/**'],
      languageOptions: { globals: { ...globals.node } },
    },
    {
      files: ['**/*.test.{ts,tsx}', '**/test/**', 'packages/testing/**'],
      rules: {
        '@typescript-eslint/no-non-null-assertion': 'off',
      },
    },
  );
}

export default rovoEslintConfig;

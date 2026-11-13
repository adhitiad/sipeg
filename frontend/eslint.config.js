import js from '@eslint/js'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import prettier from 'eslint-config-prettier'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      '.tanstack/**',
      'coverage/**',
      'playwright-report/**',
      'test-results/**',
      'src/routeTree.gen.ts',
    ],
  },
  js.configs.recommended,
  tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      ...reactHooks.configs.flat['recommended-latest'].rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },
  {
    files: ['src/components/ui/**/*.tsx'],
    rules: {
      // Komponen shadcn/ui sengaja mengekspor konstanta varian (misalnya
      // buttonVariants) bersama komponennya. Aturan fast refresh tidak
      // berlaku pada berkas yang disalin dari paket shadcn.
      'react-refresh/only-export-components': 'off',
    },
  },
  {
    files: ['src/routes/**/*.{ts,tsx}'],
    rules: {
      // File rute mengekspor konstanta Route di samping komponen; export
      // konstanta ini memang diwajibkan oleh generator rute TanStack.
      'react-refresh/only-export-components': 'off',
    },
  },
  prettier,
)

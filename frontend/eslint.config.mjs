import next from 'eslint-config-next'
import coreWebVitals from 'eslint-config-next/core-web-vitals'
import typescript from 'eslint-config-next/typescript'
import prettier from 'eslint-config-prettier'

const config = [
  ...next,
  ...coreWebVitals,
  ...typescript,
  {
    ignores: ['.next/**', 'out/**', 'build/**', 'next-env.d.ts', 'dist/**'],
  },
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_' },
      ],
    },
  },
  prettier,
]

export default config

const { FlatCompat } = require('@eslint/eslintrc')
const js = require('@eslint/js')

require('@rushstack/eslint-patch/modern-module-resolution')

const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended
})

module.exports = [
  {
    ignores: [
      'node_modules/**',
      'dist/**',
      'dist-ssr/**',
      'coverage/**',
      '*.local',
      '.env',
      'cypress/videos/**',
      'cypress/screenshots/**',
      '.idea/**'
    ]
  },
  ...compat.config({
    extends: [
      'plugin:vue/vue3-recommended',
      'eslint:recommended',
      '@vue/eslint-config-typescript'
    ],
    parserOptions: {
      ecmaVersion: 'latest'
    }
  })
]


const js = require('@eslint/js')
const vue = require('eslint-plugin-vue')
const { defineConfigWithVueTs, vueTsConfigs } = require('@vue/eslint-config-typescript')
const globals = require('globals')

module.exports = defineConfigWithVueTs(
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
      '.idea/**',
      '.eslintrc.cjs',
      'eslint.config.cjs',
      'vue.config.js'
    ]
  },
  js.configs.recommended,
  vue.configs['flat/essential'],
  vueTsConfigs.base,
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      }
    },
    rules: {
      'no-undef': 'off',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
      'no-useless-assignment': 'off',
      'no-constant-condition': 'off',
      'no-constant-binary-expression': 'off'
    }
  }
)

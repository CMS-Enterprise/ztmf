// eslint-disable-next-line prettier/prettier
import { defineConfig, transformWithEsbuild, loadEnv, type PluginOption } from 'vite'
import react from '@vitejs/plugin-react-swc'
import EnvironmentPlugin from 'vite-plugin-environment'
import { visualizer } from 'rollup-plugin-visualizer'
import sass from 'sass'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  return {
    define: {
      'process.env': {},
      global: {},
      _global: {},
    },

    resolve: {
      alias: {
        '@': '/src',
        'npm:': '/node_modules/',
      },
    },

    css: {
      preprocessorOptions: {
        scss: {
          implementation: sass,
        },
      },
    },
    plugins: [
      react(),
      EnvironmentPlugin('all'),
      // @ts-ignore-next-line
      visualizer() as PluginOption,
      {
        name: 'load+transform-js-files-as-jsx',
        async transform(code, id) {
          if (!id.match(/src\/.*\.js$/)) {
            return null
          }
          return transformWithEsbuild(code, id, {
            loader: 'jsx',
            jsx: 'automatic',
          })
        },
      },
    ],

    server: {
      host: true,
      port: 5174,
      proxy: {
        '/graphql': {
          target: 'http://localhost:3000/',
          changeOrigin: true,
          secure: false,
        },
        '/whoami': {
          target: 'http://localhost:3000/',
          changeOrigin: true,
          secure: false,
        },
      },
      watch: {
        ignored: ['**/coverage/**'],
      },
    },

    optimizeDeps: {
      esbuildOptions: {
        loader: {
          '.js': 'jsx',
        },
      },
    },

    ...(mode === 'development' && {
      build: {
        sourcemap: true,
        minify: false,
      },
    }),
  }
})

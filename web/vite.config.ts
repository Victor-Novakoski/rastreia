/// <reference types="vitest/config" />
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv, type Plugin } from 'vite'
import { contentSecurityPolicy } from './csp.ts'

// A CSP entra só no build: o servidor de desenvolvimento do Vite injeta
// scripts inline para o hot reload, que ela bloquearia.
function csp(apiURL: string): Plugin {
  return {
    name: 'rastreia-csp',
    apply: 'build',
    transformIndexHtml: () => [
      {
        tag: 'meta',
        attrs: { 'http-equiv': 'Content-Security-Policy', content: contentSecurityPolicy(apiURL) },
        injectTo: 'head-prepend',
      },
    ],
  }
}

export default defineConfig(({ mode }) => {
  const env = { ...loadEnv(mode, process.cwd()), ...process.env }
  return {
    plugins: [react(), tailwindcss(), csp(env.VITE_API_URL ?? 'http://localhost:8080')],
    server: { port: 5173 },
    test: {
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      restoreMocks: true,
    },
  }
})

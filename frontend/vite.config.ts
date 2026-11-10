import { fileURLToPath, URL } from 'node:url'
import { resolve, dirname, relative } from 'node:path'
import { existsSync } from 'node:fs'
import { readdir, stat } from 'node:fs/promises'
import { execSync } from 'node:child_process'
import { copyPdfJsWasm } from './scripts/copy-pdfjs-wasm.mjs'
import { createRequire } from 'node:module'
import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'

const __dirname = dirname(fileURLToPath(import.meta.url))
const require = createRequire(import.meta.url)

const pkg = require('./package.json') as { version?: string }
const FRONTEND_VERSION = pkg.version ?? 'unknown'

function resolveFrontendCommit(): string {
  const fromEnv = process.env.VITE_FRONTEND_COMMIT || process.env.GITHUB_SHA
  if (fromEnv) {
    return fromEnv.slice(0, 7)
  }
  try {
    return execSync('git rev-parse --short HEAD', { stdio: ['ignore', 'pipe', 'ignore'] })
      .toString()
      .trim()
  } catch {
    return 'unknown'
  }
}

const FRONTEND_COMMIT = resolveFrontendCommit()

/** Dev parity with nginx: serve embed.html for /embed/:channelId (not the main SPA). */
function embedHtmlDevFallback(): Plugin {
  return {
    name: 'embed-html-dev-fallback',
    configureServer(server) {
      server.middlewares.use((req, _res, next) => {
        const raw = req.url ?? ''
        const qIdx = raw.indexOf('?')
        const path = qIdx >= 0 ? raw.slice(0, qIdx) : raw
        const qs = qIdx >= 0 ? raw.slice(qIdx) : ''
        if (path.startsWith('/embed/') && path !== '/embed.html' && !path.includes('.')) {
          req.url = `/embed.html${qs}`
        }
        next()
      })
    },
  }
}
const DEV_PROXY_TARGET =
  process.env.VITE_DEV_PROXY_TARGET ||
  process.env.FRONTEND_BACKEND_URL ||
  'http://localhost:8080'

// Docker/WSL bind mounts swallow fs events AND chokidar's fs.watchFile polling
// never observes host writes (statSync does). When CHOKIDAR_USEPOLLING=true is
// injected (dev containers), watch src/ with an async stat sweep and trigger a
// full reload on any add/change/delete. Uses fs/promises (libuv thread pool) so
// the 9p stat latency never blocks the dev server's event loop. Host-native dev
// keeps event-driven HMR.
function pollingFullReloadWatcher(): Plugin {
  if (process.env.CHOKIDAR_USEPOLLING !== 'true') {
    return { name: 'dev-polling-watcher-disabled' }
  }
  const rootDir = __dirname
  const watchRoots = [resolve(rootDir, 'src'), resolve(rootDir, 'index.html')]
  const SWEEP_CONCURRENCY = 32
  const snapshot = async (): Promise<Map<string, number>> => {
    const out = new Map<string, number>()
    const statQueue: string[] = []
    const walk = async (dir: string) => {
      for (const entry of await readdir(dir, { withFileTypes: true })) {
        const p = resolve(dir, entry.name)
        if (entry.isDirectory()) await walk(p)
        else statQueue.push(p)
      }
    }
    for (const target of watchRoots) {
      if ((await stat(target)).isDirectory()) await walk(target)
      else out.set(target, (await stat(target)).mtimeMs)
    }
    for (let i = 0; i < statQueue.length; i += SWEEP_CONCURRENCY) {
      const batch = statQueue.slice(i, i + SWEEP_CONCURRENCY)
      const mtimes = await Promise.all(
        batch.map(async (p) => {
          try {
            return (await stat(p)).mtimeMs
          } catch {
            return -1 // deleted mid-sweep; treated as removed next sweep
          }
        }),
      )
      for (let j = 0; j < batch.length; j++) out.set(batch[j], mtimes[j])
    }
    return out
  }
  return {
    name: 'dev-polling-full-reload',
    apply: 'serve',
    configureServer(server) {
      let sweeping = false
      let snap = new Map<string, number>()
      const sweep = async () => {
        if (sweeping) return
        sweeping = true
        try {
          const next = await snapshot()
          if (snap.size === 0) {
            snap = next // first sweep only primes the baseline
            return
          }
          let changed = false
          for (const [p, m] of next) {
            if (snap.get(p) !== m) {
              changed = true
              console.log(`[polling-watcher] ${snap.has(p) ? 'change' : 'add'}: ${relative(rootDir, p)}`)
              break
            }
          }
          if (!changed) {
            for (const p of snap.keys()) {
              if (!next.has(p)) {
                changed = true
                console.log(`[polling-watcher] remove: ${relative(rootDir, p)}`)
                break
              }
            }
          }
          if (changed) {
            snap = next
            server.hot.send({ type: 'full-reload' })
          }
        } catch {
          // files mid-write on the bind mount; retry next sweep
        } finally {
          sweeping = false
        }
      }
      const timer = setInterval(sweep, 3000)
      server.httpServer?.on('close', () => clearInterval(timer))
    },
  }
}

function resolveVueOfficePptxEntry(): string {
  try {
    const pkgDir = dirname(require.resolve('@vue-office/pptx/package.json'))
    const candidates = [
      // Use ESM so setup hooks and render helpers share the application's Vue
      // runtime. The UMD entry's require('vue') can produce a second runtime in
      // Vite's dev optimizer, leaving template refs null and PPTX previews blank.
      resolve(pkgDir, 'lib/v3/vue-office-pptx.mjs'),
      resolve(pkgDir, 'lib/v3/index.js'),
      resolve(pkgDir, 'lib/index.js'),
    ]
    const matched = candidates.find((candidate) => existsSync(candidate))
    return matched ?? '@vue-office/pptx'
  } catch {
    return '@vue-office/pptx'
  }
}

function pdfJsWasmAssets(): Plugin {
  let wasmSource = ''
  let wasmOutput = ''
  return {
    name: 'pdfjs-wasm-assets',
    apply: 'build',
    configResolved(config) {
      const pdfjsBuild = require.resolve('pdfjs-dist/legacy/build/pdf.mjs')
      wasmSource = resolve(dirname(pdfjsBuild), '../../wasm')
      wasmOutput = resolve(config.root, config.build.outDir, 'pdfjs', 'wasm')
    },
    closeBundle() {
      copyPdfJsWasm(wasmSource, wasmOutput)
    },
  }
}

export default defineConfig({
  define: {
    __FRONTEND_VERSION__: JSON.stringify(FRONTEND_VERSION),
    __FRONTEND_COMMIT__: JSON.stringify(FRONTEND_COMMIT),
  },
  build: {
    modulePreload: {
      resolveDependencies(_filename, deps, { hostId }) {
        // Embed iframe bootstraps with token exchange only; defer heavy chat chunks.
        if (hostId?.includes('embed')) {
          return deps.filter((dep) => !(
            dep.includes('vendor-mermaid')
            || dep.includes('vendor-highlight')
            || dep.includes('vendor-markdown')
            || dep.includes('vendor-tdesign')
            || dep.includes('botmsg')
            || dep.includes('usermsg')
            || dep.includes('EmbedBotMessage')
            || dep.includes('EmbedUserMessage')
            || dep.includes('AgentStreamDisplay')
            || dep.includes('EmbedChatCore')
            || dep.includes('vendor-markdown')
            || dep.includes('fonts-')
          ))
        }
        return deps
      },
    },
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        embed: resolve(__dirname, 'embed.html'),
      },
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('mermaid') || id.includes('/dagre') || id.includes('cytoscape')) {
            return 'vendor-mermaid'
          }
          if (id.includes('marked') || id.includes('katex')) {
            return 'vendor-markdown'
          }
          if (id.includes('highlight.js')) {
            return 'vendor-highlight'
          }
        },
      },
    },
  },
  plugins: [
    vue(),
    vueJsx(),
    embedHtmlDevFallback(),
    pdfJsWasmAssets(),
    pollingFullReloadWatcher(),
  ],
  resolve: {
    dedupe: ['vue', 'vue-demi'],
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@vue-office/pptx': resolveVueOfficePptxEntry(),
    },
  },
  server: {
    port: 5173,
    host: true,
    // Docker/WSL 绑定挂载不转发文件事件，且 chokidar 的 fs.watchFile 轮询在该
    // 挂载上永远读到过期 stat（导致 vite 内 statSync 也异常）。注入
    // CHOKIDAR_USEPOLLING=true 时让 chokidar 不监视任何文件，改由下方
    // pollingFullReloadWatcher 插件用 statSync 扫描并触发 full-reload；
    // 本机原生开发不受影响（保持事件驱动 HMR）。
    watch: process.env.CHOKIDAR_USEPOLLING === 'true'
      ? { ignored: ['**'] }
      : {},
    // 代理配置，用于开发环境
    proxy: {
      '/mcp/': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
        // Streamable HTTP may keep an SSE response open for long-running tools.
        timeout: 3_600_000,
        proxyTimeout: 3_600_000,
      },
      '/api': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
        // 沙箱终端等 WebSocket 升级请求也走 /api，必须开启 WS 转发，
        // 否则浏览器侧握手失败、前端表现为"一直正在连接"。
        ws: true,
        // Cube fork snapshots pause a live MicroVM; 30s axios/proxy defaults
        // abort the POST and the backend then 500s on a canceled persist.
        timeout: 180_000,
        proxyTimeout: 180_000,
      },
      '/files': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
      }
    }
  },
  // `vite preview` 用生产构建产物(dist)本地起服务，是最接近 release 镜像的环境：
  // 同样的压缩 / 拆包 / CSS 加载顺序，可提前暴露只在生产构建出现的问题
  // （如主题变量被打包顺序覆盖）。用法：npm run build && npm run preview
  preview: {
    port: 4173,
    host: true,
    proxy: {
      '/mcp/': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
        timeout: 3_600_000,
        proxyTimeout: 3_600_000,
      },
      '/api': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
        ws: true,
        timeout: 180_000,
        proxyTimeout: 180_000,
      },
      '/files': {
        target: DEV_PROXY_TARGET,
        changeOrigin: true,
        secure: false,
      }
    }
  }
})

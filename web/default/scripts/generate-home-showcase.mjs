#!/usr/bin/env node
/**
 * Build /home/showcase.json for the marketing homepage marquee and upload to R2.
 *
 * Usage:
 *   node scripts/generate-home-showcase.mjs --upload
 *   node scripts/generate-home-showcase.mjs --out public/home/showcase.json --no-upload
 *   node scripts/generate-home-showcase.mjs --upload --env-file ../../../cangyuan-stack/.env
 */
import { readFileSync, existsSync } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { PutObjectCommand, S3Client } from '@aws-sdk/client-s3'
import imageSize from 'image-size'

const __dirname = dirname(fileURLToPath(import.meta.url))
const WEB_ROOT = resolve(__dirname, '..')
const REPO_ROOT = resolve(WEB_ROOT, '../..')
const WORKSPACE_ROOT = resolve(REPO_ROOT, '..')

const R2_OBJECT_KEY = 'home/showcase.json'
const SHOWCASE_PAGE_SIZE = 100
const SHOWCASE_MAX_PAGES = 3
const SHOWCASE_MAX_ITEMS = 48
const DEFAULT_ASPECT_RATIO = 3 / 4

const EXCLUDED_COVER_PATTERNS = [
  /\/home\/tools\//i,
  /\/site\/logo/i,
  /claude-code/i,
  /codex-cli/i,
  /gemini-cli/i,
  /image-api\.png/i,
]

function parseArgs() {
  const args = process.argv.slice(2)
  let out = resolve(WEB_ROOT, 'public/home/showcase.json')
  let envFile = ''
  let upload = false
  for (let i = 0; i < args.length; i++) {
    if (args[i] === '--out' && args[i + 1]) {
      out = resolve(args[i + 1])
      i++
    } else if (args[i] === '--env-file' && args[i + 1]) {
      envFile = resolve(args[i + 1])
      i++
    } else if (args[i] === '--upload') {
      upload = true
    } else if (args[i] === '--no-upload') {
      upload = false
    }
  }
  return { out, envFile, upload }
}

function loadEnvFile(path) {
  if (!existsSync(path)) return false
  for (const line of readFileSync(path, 'utf8').split('\n')) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#') || !trimmed.includes('=')) continue
    const idx = trimmed.indexOf('=')
    const key = trimmed.slice(0, idx).trim()
    let value = trimmed.slice(idx + 1).trim()
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1)
    }
    if (!process.env[key]) process.env[key] = value
  }
  return true
}

function loadEnv(envFile) {
  const candidates = [
    envFile,
    resolve(WEB_ROOT, '.env.local'),
    resolve(WEB_ROOT, '.env'),
    resolve(REPO_ROOT, '.env'),
    resolve(WORKSPACE_ROOT, 'cangyuan-stack/.env'),
  ].filter(Boolean)

  for (const file of candidates) {
    if (loadEnvFile(file)) {
      console.log(`env: ${file}`)
      return
    }
  }
}

function getRuntimeConfig() {
  const canvasBase = (
    process.env.CANVAS_BASE_URL || 'https://canvas.cangyuansuanli.cn'
  ).replace(/\/$/, '')
  const cdnBase = (
    process.env.VITE_STATIC_CDN ||
    process.env.R2_PUBLIC_BASE_URL ||
    'https://assets.cangyuansuanli.cn'
  ).replace(/\/$/, '')
  return { canvasBase, cdnBase }
}

function getR2Config() {
  const accountId = process.env.R2_ACCOUNT_ID?.trim() || ''
  const accessKeyId = process.env.R2_ACCESS_KEY_ID?.trim() || ''
  const secretAccessKey = process.env.R2_SECRET_ACCESS_KEY?.trim() || ''
  const bucket = process.env.R2_BUCKET?.trim() || ''
  const publicBase = (
    process.env.R2_PUBLIC_BASE_URL ||
    process.env.VITE_STATIC_CDN ||
    'https://assets.cangyuansuanli.cn'
  ).replace(/\/$/, '')

  if (!accountId || !accessKeyId || !secretAccessKey || !bucket) {
    return null
  }

  return { accountId, accessKeyId, secretAccessKey, bucket, publicBase }
}

function createR2Client(config) {
  return new S3Client({
    region: 'auto',
    endpoint: `https://${config.accountId}.r2.cloudflarestorage.com`,
    credentials: {
      accessKeyId: config.accessKeyId,
      secretAccessKey: config.secretAccessKey,
    },
  })
}

function isValidCoverUrl(url) {
  if (!url.startsWith('http://') && !url.startsWith('https://')) return false
  return !EXCLUDED_COVER_PATTERNS.some((pattern) => pattern.test(url))
}

function coverPriority(url) {
  if (url.includes('assets.cangyuansuanli.cn')) return 0
  if (url.includes('raw.githubusercontent.com')) return 1
  return 2
}

function mirrorCoverUrl(url, map) {
  return map[url] ?? url
}

async function probeImageDimensions(url) {
  try {
    const res = await fetch(url, {
      signal: AbortSignal.timeout(20_000),
      headers: { 'User-Agent': 'new-api-showcase-generator/1.0' },
    })
    if (!res.ok) return null
    const buffer = Buffer.from(await res.arrayBuffer())
    const size = imageSize(buffer)
    if (!size?.width || !size?.height) return null
    return { width: size.width, height: size.height }
  } catch {
    return null
  }
}

async function enrichWithDimensions(assets, concurrency = 8) {
  const results = [...assets]
  let cursor = 0

  async function worker() {
    while (cursor < results.length) {
      const index = cursor++
      const asset = results[index]
      const dims = await probeImageDimensions(asset.image)
      if (dims) {
        results[index] = {
          ...asset,
          width: dims.width,
          height: dims.height,
          aspectRatio: dims.width / dims.height,
        }
      }
    }
  }

  await Promise.all(Array.from({ length: concurrency }, worker))
  return results
}

async function loadPromptMediaMap(cdnBase) {
  const res = await fetch(`${cdnBase}/prompt-library/media-map.json`)
  if (!res.ok) return {}
  const data = await res.json()
  return data.urls ?? {}
}

async function fetchPromptPage(canvasBase, page) {
  const url = new URL('/api/prompts', canvasBase)
  url.searchParams.set('previewType', 'image')
  url.searchParams.set('pageSize', String(SHOWCASE_PAGE_SIZE))
  url.searchParams.set('page', String(page))
  const res = await fetch(url.toString())
  if (!res.ok) return []
  const data = await res.json()
  return data.items ?? []
}

async function buildAssets(canvasBase, cdnBase) {
  const [mediaMap, ...pages] = await Promise.all([
    loadPromptMediaMap(cdnBase),
    ...Array.from({ length: SHOWCASE_MAX_PAGES }, (_, i) =>
      fetchPromptPage(canvasBase, i + 1)
    ),
  ])

  const seenImages = new Set()
  const baseAssets = []

  for (const items of pages) {
    for (const item of items) {
      if (baseAssets.length >= SHOWCASE_MAX_ITEMS) break
      const mirrored = mirrorCoverUrl((item.coverUrl || '').trim(), mediaMap)
      if (!mirrored || !isValidCoverUrl(mirrored)) continue
      if (seenImages.has(mirrored)) continue
      seenImages.add(mirrored)
      baseAssets.push({
        id: item.id,
        image: mirrored,
        title: item.title,
        tags: (item.tags ?? []).slice(0, 3).join(' / '),
        aspectRatio: DEFAULT_ASPECT_RATIO,
      })
    }
    if (baseAssets.length >= SHOWCASE_MAX_ITEMS) break
  }

  baseAssets.sort((a, b) => coverPriority(a.image) - coverPriority(b.image))
  console.log(`Probing dimensions for ${baseAssets.length} covers…`)
  return enrichWithDimensions(baseAssets)
}

async function uploadToR2(body, config) {
  const client = createR2Client(config)
  await client.send(
    new PutObjectCommand({
      Bucket: config.bucket,
      Key: R2_OBJECT_KEY,
      Body: body,
      ContentType: 'application/json; charset=utf-8',
      CacheControl: 'public, max-age=86400, s-maxage=86400',
    })
  )
  return `${config.publicBase}/${R2_OBJECT_KEY}`
}

async function main() {
  const { out, envFile, upload } = parseArgs()
  loadEnv(envFile)
  const { canvasBase, cdnBase } = getRuntimeConfig()

  const assets = await buildAssets(canvasBase, cdnBase)
  if (assets.length === 0) {
    console.error('No showcase assets built; manifest not written.')
    process.exit(1)
  }

  const manifest = {
    version: 1,
    generatedAt: new Date().toISOString(),
    assets,
  }
  const body = `${JSON.stringify(manifest, null, 2)}\n`

  await mkdir(dirname(out), { recursive: true })
  await writeFile(out, body, 'utf8')
  console.log(`Wrote ${assets.length} assets to ${out}`)

  if (!upload) {
    console.log(`Skip upload (pass --upload to publish ${R2_OBJECT_KEY})`)
    return
  }

  const r2 = getR2Config()
  if (!r2) {
    console.error(
      'R2 not configured. Set R2_ACCOUNT_ID / R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY / R2_BUCKET (e.g. source cangyuan-stack/.env).'
    )
    process.exit(1)
  }

  const publicUrl = await uploadToR2(body, r2)
  console.log(`Uploaded to R2: ${publicUrl}`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})

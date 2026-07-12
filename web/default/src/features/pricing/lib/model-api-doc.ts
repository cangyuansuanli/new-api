/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { DEFAULT_API_BASE_URL } from '@/features/canvas/lib/canvas-config'
import type { PricingModel } from '../types'
import {
  getModelDisplayName,
  stripModelVendorPrefix,
} from './model-display-name'

type UiParamFieldConfig = {
  enabled?: boolean
  hint?: string
  fixedLabel?: string
  min?: number
  max?: number
  numericOptions?: number[]
  options?: Array<{ value: string; label?: string; size?: string }>
}

type VideoUiParamsDoc = {
  id?: string
  apiMode?: string
  params?: Record<string, UiParamFieldConfig>
  referenceLimits?: {
    images?: number
    videos?: number
    audios?: number
  }
  hints?: Array<{ text?: string } | string>
  requiresReferenceMedia?: boolean
}

type ImageUiParamsDoc = {
  id?: string
  apiMode?: string
  params?: Record<string, UiParamFieldConfig>
  hints?: Array<{ text?: string } | string>
}

function extractUiHintTexts(hints?: VideoUiParamsDoc['hints']): string[] {
  if (!hints?.length) return []
  return hints
    .map((item) => (typeof item === 'string' ? item : item.text?.trim()))
    .filter(Boolean) as string[]
}

function pickDefaultRatio(config?: UiParamFieldConfig): string | undefined {
  if (!config?.enabled) return undefined
  const fromOptions = config.options?.[0]?.value
  if (fromOptions) return fromOptions
  return '16:9'
}

function pickDefaultDuration(config?: UiParamFieldConfig): number | undefined {
  if (!config?.enabled) return undefined
  if (config.numericOptions?.length) return config.numericOptions[0]
  if (config.min != null) return config.min
  return 8
}

function pickDefaultResolution(
  config?: UiParamFieldConfig
): string | undefined {
  if (!config?.enabled) return undefined
  const fromOptions = config.options?.[0]?.value
  if (fromOptions) return fromOptions
  return '720p'
}

export type ModelDocParam = { name: string; description: string }

export type ModelDocEndpoint = {
  method: string
  path: string
  description: string
}

export type ModelDocExample = {
  title: string
  requestJson: string
}

export type ModelDocGenerationMode = {
  label: string
  minimum: string
  trigger: string
  promptRefs?: string
  notes?: string
}

export type ModelApiDocVariant = {
  mode: 'async' | 'sync'
  intro: string
  generationModes: ModelDocGenerationMode[]
  endpoints: ModelDocEndpoint[]
  requestJson: string
  basicRequestJson: string | null
  examples: ModelDocExample[]
  params: ModelDocParam[]
  createResponseJson: string
  queryResponseJson: string | null
  queryFailedResponseJson: string | null
}

/** 单模型 API 文档；variants 可含 async + sync 两种（如 gpt-image-2） */
export type ModelApiDoc = {
  displayName: string
  modelName: string
  variants: ModelApiDocVariant[]
}

export type RawModelApiDocExample = {
  title: string
  request_json?: unknown
}

export type RawModelDocGenerationMode = {
  label?: string
  name?: string
  minimum?: string
  min_required?: string
  trigger?: string
  fields?: string
  prompt_refs?: string
  promptRefs?: string
  notes?: string
}

export type RawModelApiDocSlice = {
  dispatch_mode?: 'async' | 'sync'
  intro?: string
  generation_modes?: RawModelDocGenerationMode[]
  endpoints?: ModelDocEndpoint[]
  request_json?: unknown
  doc_request_json?: unknown
  basic_request_json?: unknown
  examples?: RawModelApiDocExample[]
  params?: ModelDocParam[]
  doc_params_json?: ModelDocParam[]
  create_response_json?: unknown
  query_response_json?: unknown
  query_failed_response_json?: unknown
}

export type RawModelApiDoc = RawModelApiDocSlice & {
  modes?: {
    async?: RawModelApiDocSlice
    sync?: RawModelApiDocSlice
  }
}

const DUAL_IMAGE_MATCH = [
  'gpt-image-2',
  'gpt-image-1.5',
  'gpt-image-1',
  'gpt-image-2-1k',
  'gpt-image-2-2k',
]

const VIDEO_POLL_CREATE = JSON.stringify(
  {
    id: 'task_01HZX8A2...',
    status: 'queued',
    created_at: '2026-05-17T08:00:00Z',
  },
  null,
  2
)

const UNIFIED_VIDEO_ENDPOINTS = (base: string): ModelDocEndpoint[] => [
  {
    method: 'POST',
    path: `${base}/videos`,
    description: '创建视频任务（application/json 或 multipart/form-data）。',
  },
  {
    method: 'GET',
    path: `${base}/videos/{task_id}`,
    description: '查询任务状态。',
  },
  {
    method: 'GET',
    path: `${base}/videos/{task_id}/content`,
    description: '下载成片（部分模型）。',
  },
]

const UNIFIED_IMAGE_ASYNC_ENDPOINTS = (base: string): ModelDocEndpoint[] => [
  {
    method: 'POST',
    path: `${base}/images/generations`,
    description: '创建图像任务（application/json，async 必须为 true）。',
  },
  {
    method: 'GET',
    path: `${base}/images/generations/{task_id}`,
    description: '查询任务状态与结果地址。',
  },
  {
    method: 'GET',
    path: `${base}/images/{task_id}/content`,
    description: '下载图片（部分模型）。',
  },
]

const UNIFIED_IMAGE_SYNC_ENDPOINTS = (base: string): ModelDocEndpoint[] => [
  {
    method: 'POST',
    path: `${base}/images/generations`,
    description: '同步出图（application/json，勿传 async 或 async: false）。',
  },
]

function formatJson(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** api_doc 常存渠道注册名；pricing 对外已是 public 名，渲染时统一替换。 */
function rewriteChannelPrefixedModelNames(
  text: string,
  publicModelName: string
): string {
  if (!text.trim() || !publicModelName.trim()) return text
  const pattern = new RegExp(
    `[a-z0-9][a-z0-9-]*-${escapeRegExp(publicModelName)}`,
    'gi'
  )
  return text.replace(pattern, (match) => {
    const stripped = stripModelVendorPrefix(match)
    return stripped.toLowerCase() === publicModelName.toLowerCase()
      ? publicModelName
      : match
  })
}

function replacePlaceholders(
  value: string,
  modelName: string,
  base: string
): string {
  return rewriteChannelPrefixedModelNames(
    value.replaceAll('{{model}}', modelName).replaceAll('{{base}}', base),
    modelName
  )
}

function applyPlaceholdersToJson(
  value: unknown,
  modelName: string,
  base: string
): string {
  const raw = formatJson(value)
  if (!raw) return ''
  return replacePlaceholders(raw, modelName, base)
}

function normalizeGenerationModes(raw: unknown): ModelDocGenerationMode[] {
  if (!Array.isArray(raw)) return []
  return raw
    .map((item) => {
      if (!item || typeof item !== 'object') return null
      const row = item as RawModelDocGenerationMode
      const label = String(row.label ?? row.name ?? '').trim()
      const minimum = String(row.minimum ?? row.min_required ?? '').trim()
      const trigger = String(row.trigger ?? row.fields ?? '').trim()
      if (!label) return null
      const promptRefs = String(row.prompt_refs ?? row.promptRefs ?? '').trim()
      const notes = String(row.notes ?? '').trim()
      return {
        label,
        minimum,
        trigger,
        ...(promptRefs ? { promptRefs } : {}),
        ...(notes ? { notes } : {}),
      }
    })
    .filter(Boolean) as ModelDocGenerationMode[]
}

function normalizeParams(raw: unknown): ModelDocParam[] {
  if (!Array.isArray(raw)) return []
  return raw
    .filter((item) => item && typeof item === 'object')
    .map((item) => {
      const row = item as Record<string, unknown>
      return {
        name: String(row.name ?? ''),
        description: String(row.description ?? ''),
      }
    })
    .filter((item) => item.name)
}

function normalizeEndpoints(
  raw: unknown,
  base: string,
  modelName: string
): ModelDocEndpoint[] {
  if (!Array.isArray(raw)) return []
  return raw
    .filter((item) => item && typeof item === 'object')
    .map((item) => {
      const row = item as Record<string, unknown>
      return {
        method: String(row.method ?? 'POST'),
        path: replacePlaceholders(String(row.path ?? ''), modelName, base),
        description: String(row.description ?? ''),
      }
    })
    .filter((item) => item.path)
}

function inferImageDispatchMode(
  doc?: RawModelApiDocSlice | RawModelApiDoc,
  ui?: ImageUiParamsDoc
): 'async' | 'sync' {
  const mode = doc?.dispatch_mode
  if (mode === 'async' || mode === 'sync') return mode
  return isAsyncImageUi(ui) ? 'async' : 'sync'
}

function isAsyncImageUi(ui?: ImageUiParamsDoc): boolean {
  if (!ui) return true
  const mode = (ui.apiMode ?? '').trim()
  if (mode === 'images-json-async' || mode === 'images-edits-async') return true
  if (mode && !mode.includes('async')) return false
  if (ui.id?.startsWith('image-tpl')) return true
  return true
}

function supportsDualImageMode(model: PricingModel): boolean {
  const name = (model.model_name || '').toLowerCase()
  return DUAL_IMAGE_MATCH.some((token) => name.includes(token.toLowerCase()))
}

function paramNote(
  name: string,
  config?: UiParamFieldConfig,
  fallback?: string
): ModelDocParam {
  const parts: string[] = []
  if (config?.hint) parts.push(config.hint)
  if (config?.fixedLabel) parts.push(`固定：${config.fixedLabel}`)
  if (config?.options?.length) {
    parts.push(`支持 ${config.options.map((o) => o.value).join('、')}`)
  }
  if (config?.numericOptions?.length) {
    parts.push(`支持 ${config.numericOptions.join('、')} 秒`)
  }
  if (config?.min != null && config?.max != null) {
    parts.push(`范围 ${config.min}–${config.max}`)
  }
  return {
    name,
    description: parts.join('；') || fallback || '',
  }
}

function upsertParam(params: ModelDocParam[], row: ModelDocParam) {
  const idx = params.findIndex((p) => p.name === row.name)
  if (idx === -1) {
    params.push(row)
    return
  }
  if (!params[idx].description) {
    params[idx].description = row.description
  } else if (
    row.description &&
    !params[idx].description.includes(row.description)
  ) {
    params[idx].description = `${params[idx].description}；${row.description}`
  }
}

function normalizeSingleVariant(
  slice: RawModelApiDocSlice,
  model: PricingModel,
  siteOrigin: string | undefined,
  mode: 'async' | 'sync'
): ModelApiDocVariant | null {
  const origin = (siteOrigin?.trim() || DEFAULT_API_BASE_URL).replace(/\/$/, '')
  const base = `${origin}/v1`
  const modelName = model.model_name || ''

  const requestSource = slice.request_json ?? slice.doc_request_json
  const paramsSource = slice.params ?? slice.doc_params_json
  const requestJson = applyPlaceholdersToJson(requestSource, modelName, base)
  const hasContent =
    Boolean(slice.intro?.trim()) ||
    Boolean(requestJson.trim()) ||
    normalizeParams(paramsSource).length > 0 ||
    normalizeEndpoints(slice.endpoints, base, modelName).length > 0

  if (!hasContent) return null

  const queryRaw = slice.query_response_json
  const queryResponseJson =
    queryRaw == null || queryRaw === ''
      ? null
      : applyPlaceholdersToJson(queryRaw, modelName, base)

  const basicRaw = slice.basic_request_json
  const basicRequestJson =
    basicRaw == null || basicRaw === ''
      ? null
      : applyPlaceholdersToJson(basicRaw, modelName, base)

  const examples: ModelDocExample[] = (slice.examples ?? [])
    .map((item) => {
      const title = item.title?.trim()
      if (!title) return null
      const json = applyPlaceholdersToJson(item.request_json, modelName, base)
      if (!json.trim()) return null
      return { title, requestJson: json }
    })
    .filter(Boolean) as ModelDocExample[]

  const queryFailedRaw = slice.query_failed_response_json
  const queryFailedResponseJson =
    queryFailedRaw == null || queryFailedRaw === ''
      ? null
      : applyPlaceholdersToJson(queryFailedRaw, modelName, base)

  const endpoints = normalizeEndpoints(slice.endpoints, base, modelName)
  const isVideo =
    model.supported_endpoint_types?.includes('openai-video') ||
    Boolean(model.video_ui_params)
  const isImage =
    model.supported_endpoint_types?.includes('image-generation') ||
    Boolean(model.image_ui_params)

  const defaultEndpoints = isVideo
    ? UNIFIED_VIDEO_ENDPOINTS(base)
    : isImage
      ? mode === 'async'
        ? UNIFIED_IMAGE_ASYNC_ENDPOINTS(base)
        : UNIFIED_IMAGE_SYNC_ENDPOINTS(base)
      : []

  return {
    mode,
    intro: sanitizeCustomerFacingText(
      rewriteChannelPrefixedModelNames(
        slice.intro?.trim() ||
          model.description?.trim() ||
          (mode === 'async'
            ? '提交异步任务后轮询获取结果。'
            : '单次请求直接返回结果。'),
        modelName
      )
    ),
    generationModes: normalizeGenerationModes(slice.generation_modes),
    endpoints: endpoints.length > 0 ? endpoints : defaultEndpoints,
    requestJson,
    basicRequestJson,
    examples,
    params: filterGulie2KImageParams(
      mergeBananaImageParamNotes(
        normalizeParams(paramsSource),
        model.image_ui_params as ImageUiParamsDoc | undefined
      ),
      model.image_ui_params as ImageUiParamsDoc | undefined
    ),
    createResponseJson:
      applyPlaceholdersToJson(slice.create_response_json, modelName, base) ||
      (mode === 'async'
        ? VIDEO_POLL_CREATE
        : formatJson({
            data: [{ url: 'https://example.com/image.png' }],
          })),
    queryResponseJson,
    queryFailedResponseJson,
  }
}

export function normalizeModelApiDoc(
  raw: RawModelApiDoc | Record<string, unknown> | undefined,
  model: PricingModel,
  siteOrigin?: string
): ModelApiDoc | null {
  if (!raw || typeof raw !== 'object') return null

  const doc = raw as RawModelApiDoc
  const displayName = getModelDisplayName(model) || model.model_name || ''
  const modelName = model.model_name || ''

  if (doc.modes && typeof doc.modes === 'object') {
    const variants: ModelApiDocVariant[] = []
    if (doc.modes.async) {
      const v = normalizeSingleVariant(
        doc.modes.async,
        model,
        siteOrigin,
        'async'
      )
      if (v) variants.push(v)
    }
    if (doc.modes.sync) {
      const v = normalizeSingleVariant(
        doc.modes.sync,
        model,
        siteOrigin,
        'sync'
      )
      if (v) variants.push(v)
    }
    if (variants.length === 0) return null
    return { displayName, modelName, variants }
  }

  const mode = inferImageDispatchMode(
    doc,
    model.image_ui_params as ImageUiParamsDoc
  )
  const single = normalizeSingleVariant(doc, model, siteOrigin, mode)
  if (!single) return null
  return { displayName, modelName, variants: [single] }
}

function buildUnifiedVideoDoc(
  model: PricingModel,
  base: string,
  displayName: string,
  modelName: string
): ModelApiDoc {
  const ui = model.video_ui_params as VideoUiParamsDoc | undefined
  const hints = extractUiHintTexts(ui?.hints)
  const paramsConfig = ui?.params ?? {}
  const limits = ui?.referenceLimits ?? {}

  const metadata: Record<string, unknown> = {}
  const ratio = pickDefaultRatio(paramsConfig.ratio)
  const resolution = pickDefaultResolution(paramsConfig.resolution)
  if (ratio) metadata.aspect_ratio = ratio
  if (resolution) metadata.resolution = resolution

  const body: Record<string, unknown> = {
    model: modelName,
    prompt: '雨夜城市街道，电影感镜头缓慢推进',
    duration: pickDefaultDuration(paramsConfig.duration) ?? 5,
  }
  if (Object.keys(metadata).length > 0) body.metadata = metadata
  if ((limits.images ?? 0) > 0) {
    body.reference_image_urls = ['https://example.com/ref.png']
  }

  const params: ModelDocParam[] = [
    { name: 'model', description: `必填，固定传 ${modelName}。` },
    { name: 'prompt', description: '必填，视频描述提示词。' },
    paramNote('duration', paramsConfig.duration, '视频时长（秒）。'),
    paramNote('metadata.aspect_ratio', paramsConfig.ratio, '画幅比例。'),
    paramNote('metadata.resolution', paramsConfig.resolution, '清晰度档位。'),
    { name: 'size', description: '画幅像素，如 1280x720。' },
    {
      name: 'reference_image_urls',
      description: '参考图 URL 数组（图生视频，Seedance 等）。',
    },
    { name: 'images', description: '参考图 URL 数组。' },
    {
      name: 'image_url',
      description: '单张参考图 URL 或 Base64（JSON 图生视频，Omni 等）。',
    },
    {
      name: 'input_reference',
      description: 'multipart 参考图文件（可多张）；JSON 亦兼容单张 string。',
    },
    { name: 'first_image_url', description: '首帧参考图 URL（JSON）。' },
    { name: 'last_image_url', description: '末帧参考图 URL（JSON）。' },
  ].filter((p) => p.description)

  if ((limits.images ?? 0) > 0) {
    upsertParam(params, {
      name: 'reference_image_urls',
      description: `参考图最多 ${limits.images} 张。`,
    })
  }
  if ((limits.videos ?? 0) > 0) {
    upsertParam(params, {
      name: 'reference_videos',
      description: `参考视频最多 ${limits.videos} 个。`,
    })
  }

  return {
    displayName,
    modelName,
    variants: [
      {
        mode: 'async',
        intro:
          hints.join(' ') ||
          model.description?.trim() ||
          '统一视频接口：POST /v1/videos 提交任务，GET 轮询至完成后取片。',
        generationModes: [],
        endpoints: UNIFIED_VIDEO_ENDPOINTS(base),
        requestJson: formatJson(body),
        basicRequestJson: formatJson({
          model: modelName,
          prompt: body.prompt,
          duration: body.duration,
          ...(Object.keys(metadata).length > 0 ? { metadata } : {}),
        }),
        examples: [],
        params,
        createResponseJson: VIDEO_POLL_CREATE,
        queryResponseJson: formatJson({
          id: 'task_01HZX8A2...',
          status: 'completed',
          data: [{ url: `${base}/videos/task_01HZX8A2.../content` }],
        }),
        queryFailedResponseJson: null,
      },
    ],
  }
}

function pickImageSize(paramsConfig: ImageUiParamsDoc['params']): string {
  const sizeOption = paramsConfig?.size?.options?.[0] as
    | { size?: string; value?: string }
    | undefined
  return sizeOption?.size ?? sizeOption?.value ?? '1024x1024'
}

function pickImageAspectRatio(
  paramsConfig: ImageUiParamsDoc['params']
): string {
  const options = paramsConfig?.aspectRatio?.options ?? []
  const option = options.find((item) => item.value && item.value !== 'auto')
  if (!option?.value) return '1:1'
  const raw = option.value.trim()
  if (raw.includes(':')) {
    const ratioPart = raw.replace(/-(4k|2k|1k)$/i, '')
    if (/^\d+:\d+$/.test(ratioPart)) return ratioPart
    if (!raw.includes('-')) return raw
  }
  return '1:1'
}

function imageQualityToOutputResolution(
  value: string | undefined
): string | null {
  const normalized = (value || '').trim().toLowerCase()
  if (normalized === 'high' || normalized === 'hd' || normalized === '4k')
    return '4K'
  if (normalized === 'medium' || normalized === '2k') return '2K'
  if (normalized === 'low' || normalized === 'standard' || normalized === '1k')
    return '1K'
  return null
}

function pickImageOutputResolution(
  paramsConfig: ImageUiParamsDoc['params'],
  modelName: string
): string | null {
  const options = paramsConfig?.quality?.options ?? []
  const values = new Set(options.map((item) => item.value))
  if (modelName.toLowerCase().includes('4k') && values.has('high')) return '4K'
  if (values.has('medium')) return '2K'
  if (values.has('high')) return '4K'
  if (values.has('low')) return '1K'
  const firstMapped = options
    .map((item) => imageQualityToOutputResolution(item.value))
    .find(Boolean)
  return firstMapped ?? null
}

function usesBananaStyleImageParams(ui?: ImageUiParamsDoc): boolean {
  const id = (ui?.id || '').toLowerCase()
  return id.includes('banana') || id.includes('adobe2api')
}

function usesGulie2KImageParams(ui?: ImageUiParamsDoc): boolean {
  return (ui?.id || '').toLowerCase() === 'image-tpl-gulie-2k'
}

const GULIE_2K_FORBIDDEN_IMAGE_PARAMS = new Set([
  'quality',
  'image_size',
  'output_resolution',
  'resolution',
  'aspect_ratio',
])

const GULIE_2K_SIZE_PARAM_NOTE =
  '画幅比例：1:1、3:2、2:3 或 auto。本模型固定 2K 档位，请勿传 quality、image_size、output_resolution、resolution 或像素尺寸；传入后平台会忽略。'

function buildGulie2KImageParams(
  paramsConfig: ImageUiParamsDoc['params']
): ModelDocParam[] {
  const aspectNote = paramNote('aspectRatio', paramsConfig?.aspectRatio)
  const sizeDescription = [aspectNote.description, GULIE_2K_SIZE_PARAM_NOTE]
    .filter(Boolean)
    .join(' ')
  return [
    { name: 'size', description: sizeDescription },
    paramNote('n', paramsConfig?.count, '生成张数，默认 1。'),
    { name: 'stream', description: '建议 false（非 SSE JSON 响应）。' },
  ].filter((p) => p.description)
}

function filterGulie2KImageParams(
  params: ModelDocParam[],
  ui?: ImageUiParamsDoc
): ModelDocParam[] {
  if (!usesGulie2KImageParams(ui)) {
    return params
  }
  const filtered = params
    .filter((p) => !GULIE_2K_FORBIDDEN_IMAGE_PARAMS.has(p.name))
    .map((p) => {
      if (p.name !== 'size') {
        return {
          ...p,
          description: sanitizeCustomerFacingText(p.description),
        }
      }
      const cleaned = sanitizeCustomerFacingText(p.description)
        .replace(/兼容传像素[^；。]*/g, '')
        .replace(/1:1\s*@\s*1K[^；。]*/gi, '')
        .replace(/Gulie\s*线路/g, '')
        .replace(/\s{2,}/g, ' ')
        .trim()
      return {
        name: 'size',
        description: [cleaned, GULIE_2K_SIZE_PARAM_NOTE]
          .filter(Boolean)
          .join(' '),
      }
    })
  if (!filtered.some((p) => p.name === 'size')) {
    filtered.unshift({
      name: 'size',
      description: GULIE_2K_SIZE_PARAM_NOTE,
    })
  }
  return filtered
}

const BANANA_IMAGE_PARAM_NOTES = {
  aspectRatio:
    '画幅比例支持任意正整数 W:H（如 7:6、110:73）；列出的值只是常用预设。请显式传 aspect_ratio；勿把 16:9-4k 等 UI 标签写进 size。',
  outputResolution:
    '推荐 1K / 2K / 4K。image_size 为兼容别名；若同时传入，须与 output_resolution 保持一致。',
  quality:
    'OpenAI 风格别名：low=1K、medium=2K、high=4K；推荐直接传 output_resolution。',
  size: '兼容旧 OpenAI 像素尺寸（如 1024x1024）；仅用于推断 aspect_ratio / output_resolution，勿与 aspect_ratio 混用。',
  resolution:
    '视频专用（如 720p、1080p）。图像请求请勿使用，否则可能无法得到预期的 4K 档位。',
  jsonReference:
    'JSON 图生图：在 POST /v1/images/generations 中传 image / images / reference_images（URL 或 data URI）。',
  multipartReference:
    'multipart 图生图：POST /v1/images/edits；多图参考请重复字段名 image（勿用 image[]）；image 只传文件，不要填 URL。',
} as const

function sanitizeCustomerFacingText(text: string): string {
  return text
    .replace(/Adobe2API\s*\/?\s*Manju\s*均会读取该字段/g, '平台会读取该字段')
    .replace(/Adobe2API/g, '平台')
    .replace(/上游[^，。；\n]*/g, '')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

function mergeBananaImageParamNotes(
  params: ModelDocParam[],
  ui?: ImageUiParamsDoc
): ModelDocParam[] {
  if (!usesBananaStyleImageParams(ui)) {
    return params.map((p) => ({
      ...p,
      description: sanitizeCustomerFacingText(p.description),
    }))
  }
  const merged = params.map((p) => ({
    ...p,
    description: sanitizeCustomerFacingText(p.description),
  }))
  for (const row of buildBananaStyleImageParams(ui?.params ?? {})) {
    const idx = merged.findIndex((p) => p.name === row.name)
    if (idx === -1) {
      merged.push(row)
    } else {
      merged[idx] = { ...merged[idx], description: row.description }
    }
  }
  return merged
}

function buildBananaStyleImageParams(
  paramsConfig: ImageUiParamsDoc['params']
): ModelDocParam[] {
  const aspectPresets = (paramsConfig?.aspectRatio?.options ?? [])
    .map((option) => option.value)
    .filter(Boolean)
    .join('、')
  const aspectDescription = [
    BANANA_IMAGE_PARAM_NOTES.aspectRatio,
    aspectPresets ? `常用预设：${aspectPresets}。` : '',
  ]
    .filter(Boolean)
    .join(' ')
  return [
    { name: 'aspect_ratio', description: aspectDescription },
    {
      name: 'output_resolution',
      description: BANANA_IMAGE_PARAM_NOTES.outputResolution,
    },
    {
      name: 'image_size',
      description: 'output_resolution 的兼容别名；若同时传入，须保持一致。',
    },
    {
      name: 'quality',
      description: BANANA_IMAGE_PARAM_NOTES.quality,
    },
    {
      name: 'size',
      description: BANANA_IMAGE_PARAM_NOTES.size,
    },
    {
      name: 'resolution',
      description: BANANA_IMAGE_PARAM_NOTES.resolution,
    },
    {
      name: 'image / images',
      description: BANANA_IMAGE_PARAM_NOTES.jsonReference,
    },
    {
      name: 'multipart image',
      description: BANANA_IMAGE_PARAM_NOTES.multipartReference,
    },
  ]
}

function buildAsyncImageVariant(
  model: PricingModel,
  base: string,
  modelName: string,
  ui?: ImageUiParamsDoc
): ModelApiDocVariant {
  const hints = extractUiHintTexts(ui?.hints)
  const paramsConfig = ui?.params ?? {}
  const useBananaParams = usesBananaStyleImageParams(ui)
  const useGulie2KParams = usesGulie2KImageParams(ui)
  const size = pickImageSize(paramsConfig)
  const aspectRatio = pickImageAspectRatio(paramsConfig)
  const outputResolution = pickImageOutputResolution(paramsConfig, modelName)
  const resolutionFields = outputResolution
    ? { output_resolution: outputResolution, image_size: outputResolution }
    : {}
  const requestFields = useBananaParams
    ? {
        aspect_ratio: aspectRatio,
        ...resolutionFields,
      }
    : useGulie2KParams
      ? { size: aspectRatio }
      : { size }

  const params: ModelDocParam[] = [
    { name: 'model', description: `必填，固定传 ${modelName}。` },
    { name: 'prompt', description: '必填，图像描述提示词。' },
    { name: 'async', description: '必填 true，启用异步任务模式。' },
    ...(useBananaParams
      ? buildBananaStyleImageParams(paramsConfig)
      : useGulie2KParams
        ? buildGulie2KImageParams(paramsConfig)
        : [
            paramNote('size', paramsConfig?.size, '输出尺寸。'),
            paramNote('quality', paramsConfig?.quality, '画质档位。'),
          ]),
    paramNote('n', paramsConfig?.count, '生成张数，默认 1。'),
  ].filter((p) => p.description)

  return {
    mode: 'async',
    intro:
      hints.join(' ') ||
      model.description?.trim() ||
      '异步出图：POST（async: true）提交任务，GET 轮询至 completed 后取图。',
    generationModes: [],
    endpoints: UNIFIED_IMAGE_ASYNC_ENDPOINTS(base),
    requestJson: formatJson({
      model: modelName,
      prompt: '一只橘猫坐在窗台上，午后阳光',
      ...requestFields,
      n: 1,
      async: true,
    }),
    basicRequestJson: formatJson({
      model: modelName,
      prompt: '一只橘猫坐在窗台上，午后阳光',
      ...requestFields,
      n: 1,
      async: true,
    }),
    examples: [],
    params,
    createResponseJson: formatJson({
      id: 'task_img_01HZX8A2...',
      object: 'image.generation',
      model: modelName,
      status: 'queued',
      progress: '10%',
      created_at: 1715923200,
    }),
    queryResponseJson: formatJson({
      id: 'task_img_01HZX8A2...',
      object: 'image.generation',
      status: 'completed',
      progress: '100%',
      data: [{ url: 'https://example.com/image.png' }],
    }),
    queryFailedResponseJson: null,
  }
}

function buildSyncImageVariant(
  model: PricingModel,
  base: string,
  modelName: string,
  ui?: ImageUiParamsDoc
): ModelApiDocVariant {
  const hints = extractUiHintTexts(ui?.hints)
  const paramsConfig = ui?.params ?? {}
  const useBananaParams = usesBananaStyleImageParams(ui)
  const useGulie2KParams = usesGulie2KImageParams(ui)
  const size = pickImageSize(paramsConfig)
  const aspectRatio = pickImageAspectRatio(paramsConfig)
  const outputResolution = pickImageOutputResolution(paramsConfig, modelName)
  const resolutionFields = outputResolution
    ? { output_resolution: outputResolution, image_size: outputResolution }
    : {}
  const requestFields = useBananaParams
    ? {
        aspect_ratio: aspectRatio,
        ...resolutionFields,
      }
    : useGulie2KParams
      ? { size: aspectRatio }
      : { size }

  const params: ModelDocParam[] = [
    { name: 'model', description: `必填，固定传 ${modelName}。` },
    { name: 'prompt', description: '必填，图像描述提示词。' },
    ...(useBananaParams
      ? buildBananaStyleImageParams(paramsConfig)
      : useGulie2KParams
        ? buildGulie2KImageParams(paramsConfig)
        : [
            paramNote('size', paramsConfig?.size, '输出尺寸。'),
            paramNote('quality', paramsConfig?.quality, '画质档位。'),
          ]),
    paramNote('n', paramsConfig?.count, '生成张数，默认 1。'),
    {
      name: 'response_format',
      description: 'url 返回图片地址；b64_json 返回 base64。',
    },
  ].filter((p) => p.description)

  return {
    mode: 'sync',
    intro:
      hints.join(' ') ||
      model.description?.trim() ||
      '同步出图：单次 POST 直接返回图片，无需轮询。',
    generationModes: [],
    endpoints: UNIFIED_IMAGE_SYNC_ENDPOINTS(base),
    requestJson: formatJson({
      model: modelName,
      prompt: '一只橘猫坐在窗台上，午后阳光',
      ...requestFields,
      n: 1,
      response_format: 'url',
    }),
    basicRequestJson: formatJson({
      model: modelName,
      prompt: '一只橘猫坐在窗台上，午后阳光',
      ...requestFields,
      n: 1,
      response_format: 'url',
    }),
    examples: [],
    params,
    createResponseJson: formatJson({
      created: 1715923200,
      data: [{ url: 'https://example.com/image.png' }],
    }),
    queryResponseJson: null,
    queryFailedResponseJson: null,
  }
}

function buildUnifiedImageDoc(
  model: PricingModel,
  base: string,
  displayName: string,
  modelName: string
): ModelApiDoc {
  const ui = model.image_ui_params as ImageUiParamsDoc | undefined

  if (supportsDualImageMode(model)) {
    return {
      displayName,
      modelName,
      variants: [
        buildAsyncImageVariant(model, base, modelName, ui),
        buildSyncImageVariant(model, base, modelName, ui),
      ],
    }
  }

  const dispatch = inferImageDispatchMode(model.api_doc, ui)
  return {
    displayName,
    modelName,
    variants: [
      dispatch === 'async'
        ? buildAsyncImageVariant(model, base, modelName, ui)
        : buildSyncImageVariant(model, base, modelName, ui),
    ],
  }
}

function buildMinimalFallback(
  model: PricingModel,
  siteOrigin?: string
): ModelApiDoc {
  const origin = (siteOrigin?.trim() || DEFAULT_API_BASE_URL).replace(/\/$/, '')
  const base = `${origin}/v1`
  const modelName = model.model_name || ''
  const displayName = getModelDisplayName(model) || modelName
  const endpoints = model.supported_endpoint_types || []

  if (endpoints.includes('openai-video')) {
    return buildUnifiedVideoDoc(model, base, displayName, modelName)
  }

  if (
    endpoints.includes('image-generation') ||
    endpoints.includes('openai-image') ||
    model.image_ui_params
  ) {
    return buildUnifiedImageDoc(model, base, displayName, modelName)
  }

  if (model.video_ui_params) {
    return buildUnifiedVideoDoc(model, base, displayName, modelName)
  }

  return {
    displayName,
    modelName,
    variants: [
      {
        mode: 'sync',
        intro: model.description?.trim() || 'OpenAI 兼容 Chat 接口。',
        generationModes: [],
        endpoints: [
          {
            method: 'POST',
            path: `${base}/chat/completions`,
            description: '对话补全。',
          },
        ],
        requestJson: formatJson({
          model: modelName,
          messages: [{ role: 'user', content: '你好，请介绍一下你自己。' }],
        }),
        basicRequestJson: null,
        examples: [],
        params: [
          { name: 'model', description: `固定传 ${modelName}。` },
          { name: 'messages', description: '对话消息数组。' },
        ],
        createResponseJson: formatJson({
          choices: [{ message: { role: 'assistant', content: '...' } }],
        }),
        queryResponseJson: null,
        queryFailedResponseJson: null,
      },
    ],
  }
}

export function buildModelApiDoc(
  model: PricingModel,
  siteOrigin?: string
): ModelApiDoc {
  const fromConfig = normalizeModelApiDoc(model.api_doc, model, siteOrigin)
  if (fromConfig) return fromConfig
  return buildMinimalFallback(model, siteOrigin)
}

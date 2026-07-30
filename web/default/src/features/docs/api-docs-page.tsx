/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useSystemConfig } from '@/hooks/use-system-config'
import { DEFAULT_API_BASE_URL } from '@/features/canvas/lib/canvas-config'
import { ModelDocPicker } from '@/features/pricing/components/model-doc-picker'
import { CodeBlock } from './components/code-block'
import { DocsSection } from './components/docs-section'
import { DocsTable } from './components/docs-table'
import { DocsNavLink, DocsShell } from './docs-shell'

const apiDocsNavItems = [
  { id: 'api-video-api', titleKey: 'apiDocs.nav.videoApi' },
  { id: 'api-image-api', titleKey: 'apiDocs.nav.imageApi' },
  { id: 'api-model-docs', titleKey: 'apiDocs.nav.modelDocs' },
] as const

const PRICING_NOTE = '具体单价与计费方式（按次 / 按秒）以模型广场为准；失败任务通常不计费。'

export function ApiDocsPage() {
  const { t } = useTranslation()
  const { systemName } = useSystemConfig()

  const siteOrigin = useMemo(() => {
    if (typeof window === 'undefined') return DEFAULT_API_BASE_URL
    return window.location.origin || DEFAULT_API_BASE_URL
  }, [])

  const base = `${siteOrigin.trim() || DEFAULT_API_BASE_URL}/v1`
  const displayName = systemName?.trim() || '沧元算力'

  useEffect(() => {
    document.title = t('apiDocs.pageTitle', { siteName: displayName })
  }, [displayName, t])

  return (
    <DocsShell
      mode='api'
      eyebrow={t('apiDocs.eyebrow')}
      title={t('apiDocs.title', { siteName: displayName })}
      subtitle={t('apiDocs.subtitle')}
      sidebarLabel={t('apiDocs.sidebarLabel')}
      nav={
        <>
          {apiDocsNavItems.map((item) => (
            <DocsNavLink key={item.id} href={`#${item.id}`}>
              {t(item.titleKey)}
            </DocsNavLink>
          ))}
        </>
      }
    >
      <DocsSection
        id='api-video-api'
        title='视频生成 API'
        description='所有视频模型共用同一套对外接口：POST /v1/videos 提交 → GET 轮询 → 取片。各模型仅参数范围不同，见下方弹窗说明。'
      >
        <p className='text-muted-foreground text-sm'>{PRICING_NOTE}</p>

        <p className='text-sm'>
          鉴权：<code className='text-sm'>Authorization: Bearer sk-你的令牌</code>。模型名与模型广场展示名一致。
        </p>

        <h3 className='text-lg font-semibold'>调用流程</h3>
        <DocsTable
          headers={['步骤', '方法', '说明']}
          rows={[
            ['1. 提交任务', 'POST /v1/videos', 'JSON 或 multipart；不同模型参数由平台内部适配'],
            ['2. 轮询进度', 'GET /v1/videos/{task_id}', 'status: queued / in_progress / completed / failed'],
            ['3. 下载成片', 'GET /v1/videos/{task_id}/content', '或取响应 data[0].url'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>对外接口</h3>
        <DocsTable
          headers={['接口', '说明']}
          rows={[
            ['POST /v1/videos', '所有视频模型统一入口'],
            ['GET /v1/videos/{task_id}', '查询任务状态'],
            ['GET /v1/videos/{task_id}/content', '下载成片（部分模型）'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>快速示例</h3>
        <CodeBlock
          title='文生视频（POST /v1/videos）'
          code={`curl -X POST ${base}/videos \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "seedance-2.0-720p",
    "prompt": "雨夜霓虹街道，镜头缓慢推进，电影感光影",
    "aspect_ratio": "16:9",
    "duration": 8
  }'`}
        />
        <CodeBlock
          title='轮询取片'
          code={`curl ${base}/videos/{task_id} \\
  -H "Authorization: Bearer sk-xxx"`}
        />

        <ul className='list-disc space-y-2 pl-5'>
          <li>视频生成通常 30 秒–5 分钟，轮询间隔建议 5–10 秒，客户端超时 ≥300 秒</li>
          <li>仅成功出片才计费；失败不扣费</li>
          <li>resolution（如 720p、1080p）为视频专用参数；图像 API 请勿使用</li>
          <li>Omni / Veo 等 Gemini 视频模型内容审查较严，避免真人正脸、版权 IP、敏感题材</li>
        </ul>
      </DocsSection>

      <DocsSection
        id='api-image-api'
        title='图像生成 API'
        description='统一入口 POST /v1/images/generations：文生图、图生图（JSON 传 image/images）与可选 async 均走同一路径；multipart 图生图可用 POST /v1/images/edits。各模型 sync/async 与参数见下方弹窗。'
      >
        <p className='text-muted-foreground text-sm'>{PRICING_NOTE}</p>

        <p className='text-sm'>
          鉴权：<code className='text-sm'>Authorization: Bearer sk-你的令牌</code>。模型名与模型广场展示名一致。
        </p>

        <h3 className='text-lg font-semibold'>调用流程</h3>
        <DocsTable
          headers={['步骤', '方法', '说明']}
          rows={[
            ['1. 提交', 'POST /v1/images/generations', '文生图/图生图 JSON；图生图加 image 或 images；async: true 走异步'],
            ['1b. 图生图 multipart', 'POST /v1/images/edits', '上传 image/mask 文件（与 JSON 图生图等价）'],
            ['2. 轮询（仅 async）', 'GET /v1/images/generations/{task_id}', 'status: queued / in_progress / completed / failed'],
            ['3. 取图', '响应 data[0].url', '同步直接返回；异步 completed 后取；或 GET /v1/images/{task_id}/content'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>出图模式</h3>
        <DocsTable
          headers={['模式', '适用', '说明']}
          rows={[
            ['同步 sync', 'Gemini Banana 等推荐', 'POST 勿传 async，stream: false，单次返回 data[].url'],
            ['异步 async', '大图/批量/画布部分模型', 'POST 带 async: true，再 GET 轮询 task_id'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>对外接口</h3>
        <DocsTable
          headers={['端点', '说明']}
          rows={[
            ['POST /v1/images/generations', '统一入口（文生图、JSON 图生图、sync/async）'],
            ['POST /v1/images/edits', 'multipart 图生图（参考图/蒙版）'],
            ['GET /v1/images/generations/{task_id}', '异步任务查询'],
            ['GET /v1/images/{task_id}/content', '下载图片（部分模型）'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>快速示例</h3>
        <CodeBlock
          title='同步文生图（Gemini Banana 等）'
          code={`curl -X POST ${base}/images/generations \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-banana-pro-4k",
    "prompt": "一只橘猫趴在窗台上晒太阳，水彩画风格",
    "aspect_ratio": "1:1",
    "output_resolution": "4K",
    "image_size": "4K",
    "stream": false
  }'`}
        />
        <CodeBlock
          title='JSON 图生图（参考图）'
          code={`curl -X POST ${base}/images/generations \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-banana-pro-4k",
    "prompt": "将 @图片1 的风格应用到新场景",
    "aspect_ratio": "16:9",
    "output_resolution": "4K",
    "image_size": "4K",
    "image": "https://example.com/ref.png",
    "stream": false
  }'`}
        />
        <CodeBlock
          title='multipart 图生图（多参考图）'
          code={`curl -X POST ${base}/images/edits \\
  -H "Authorization: Bearer sk-xxx" \\
  -F "model=gemini-banana-pro-4k" \\
  -F "prompt=将参考图风格应用到新场景" \\
  -F "aspect_ratio=16:9" \\
  -F "output_resolution=4K" \\
  -F "image=@/path/to/ref1.png" \\
  -F "image=@/path/to/ref2.png"`}
        />
        <CodeBlock
          title='异步文生图'
          code={`curl -X POST ${base}/images/generations \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "一只橘猫趴在窗台上晒太阳，水彩画风格",
    "size": "1024x1024",
    "n": 1,
    "async": true,
    "stream": false
  }'`}
        />
        <CodeBlock
          title='轮询取图'
          code={`curl ${base}/images/generations/{task_id} \\
  -H "Authorization: Bearer sk-xxx"`}
        />

        <h3 className='mt-8 text-lg font-semibold'>注意事项</h3>
        <DocsTable
          headers={['场景', '说明']}
          rows={[
            [
              '模型档位与计费',
              'GPT Image 2 的 1K / 2K / 4K 由模型名选择并分别计费。quality 只控制画质，不会切换模型档位、像素预算或计费。',
            ],
            [
              'GPT Image 2 quality',
              '可选 low / medium / high；省略或传 auto 时默认为 medium。quality 与 1K / 2K / 4K 售卖档位相互独立。',
            ],
            [
              'GPT Image 2 精确 size',
              '传 WIDTHxHEIGHT 时校验后原样转发，不推断比例、不重算尺寸。两边须为 16 的倍数，最长边 ≤3840px，长短边比例 ≤3:1，总像素至少 655360；1K / 2K / 4K 上限分别为 1048576 / 4194304 / 8294400。',
            ],
            [
              '比例输入',
              '未传精确 size 时，可传 aspect_ratio 或比例形式的 size（如 7:6），平台会在所选模型档位的像素预算内计算尺寸。Adobe Banana 系列的常见与自定义正整数 W:H 比例也使用 aspect_ratio。',
            ],
            [
              'resolution 字段',
              'resolution（如 720p）为视频专用；图像请求请勿使用，否则可能无法得到预期的 4K 档位。',
            ],
            [
              'JSON 图生图',
              'POST /v1/images/generations；兼容 image、images、imageUrls、image_urls、reference_images、referenceImages、image_refs（http(s) URL 或 data URI）。别名之间会去重，最多 9 张唯一参考图。',
            ],
            [
              'multipart 图生图',
              'POST /v1/images/edits；多图参考请重复字段名 image，最多 9 张；image 只传文件，不要填 URL。',
            ],
            [
              '同步返回格式',
              'response_format 可传 url（默认）或 b64_json；异步任务完成后返回图片 URL。',
            ],
            [
              'sync / async',
              'Gemini Banana 等推荐同步（勿传 async）；大图或多参考图可选用 async: true。异步轮询间隔建议 3–5 秒。',
            ],
            ['计费', '仅成功出图才计费；chat/completions 出图仍兼容但已弃用，响应带 Deprecation 头。'],
          ]}
        />
      </DocsSection>

      <DocsSection
        id='api-model-docs'
        title='单模型 API 说明'
        description='按供应商与能力分类；点击模型名查看该模型的接口地址、请求 JSON 与字段说明（与模型广场「查看文档」相同）。'
      >
        <ModelDocPicker siteOrigin={siteOrigin} capability='all' />
      </DocsSection>
    </DocsShell>
  )
}

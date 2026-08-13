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
  { id: 'api-audio-api', titleKey: 'apiDocs.nav.audioApi' },
  { id: 'api-model-docs', titleKey: 'apiDocs.nav.modelDocs' },
] as const

const PRICING_NOTE =
  '具体单价与计费方式（按次 / 按秒）以模型广场为准；失败任务通常不计费。'

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
        description='所有视频模型共用任务入口与状态生命周期；每个模型独立维护自己的请求字段、取值范围和素材能力。'
      >
        <p className='text-muted-foreground text-sm'>{PRICING_NOTE}</p>

        <p className='text-sm'>
          鉴权：
          <code className='text-sm'>Authorization: Bearer sk-你的令牌</code>
          。模型名与模型广场展示名一致。
        </p>

        <h3 className='text-lg font-semibold'>调用流程</h3>
        <DocsTable
          headers={['步骤', '方法', '说明']}
          rows={[
            [
              '1. 提交任务',
              'POST /v1/videos',
              'application/json；请求体只使用所选模型文档列出的字段',
            ],
            [
              '2. 轮询进度',
              'GET /v1/videos/{task_id}',
              'status: queued / in_progress / completed / failed',
            ],
            [
              '3. 下载成片',
              'GET /v1/videos/{task_id}/content',
              '或取响应 data[0].url',
            ],
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

        <h3 className='mt-8 text-lg font-semibold'>Canonical 字段</h3>
        <DocsTable
          headers={['字段', '说明']}
          rows={[
            ['model / prompt', '必填：模型广场 public 名与视频描述'],
            ['duration', '时长秒数'],
            [
              'aspect_ratio / resolution / size',
              '统一画幅、清晰度和像素尺寸字段',
            ],
            ['seed / generate_audio', '随机种子与音频开关'],
            ['reference_image_urls', '参考图 HTTPS URL 数组'],
            [
              'reference_videos / reference_audios',
              '参考视频与音频 HTTPS URL 数组',
            ],
            ['first_image_url / last_image_url', '首尾帧 HTTPS URL'],
          ]}
        />

        <p className='text-sm'>
          上表是统一入站 schema；提交 JSON
          时使用下方单模型文档声明的支持子集与完整示例，未在该模型文档列出的字段不要发送。
        </p>
        <CodeBlock
          title='轮询取片'
          code={`curl ${base}/videos/{task_id} \\
  -H "Authorization: Bearer sk-xxx"`}
        />

        <ul className='list-disc space-y-2 pl-5'>
          <li>
            视频生成通常 30 秒–5 分钟，轮询间隔建议 5–10 秒，客户端超时 ≥300 秒
          </li>
          <li>仅成功出片才计费；失败不扣费</li>
          <li>字段、范围、参考素材数量与生成模式以所选模型的独立文档为准</li>
        </ul>
      </DocsSection>

      <DocsSection
        id='api-image-api'
        title='图像生成 API'
        description='图像 API 统一使用 canonical JSON 字段：文生图提交 /v1/images/generations，参考图或蒙版编辑提交 /v1/images/edits。每个模型独立维护支持字段和取值范围。'
      >
        <p className='text-muted-foreground text-sm'>{PRICING_NOTE}</p>

        <p className='text-sm'>
          鉴权：
          <code className='text-sm'>Authorization: Bearer sk-你的令牌</code>
          。模型名与模型广场展示名一致。
        </p>

        <h3 className='text-lg font-semibold'>调用流程</h3>
        <DocsTable
          headers={['步骤', '方法', '说明']}
          rows={[
            [
              '1. 文生图',
              'POST /v1/images/generations',
              'application/json；只提交模型文档列出的 canonical 字段',
            ],
            [
              '1b. 参考图编辑',
              'POST /v1/images/edits',
              'application/json；images / mask 仅传 HTTPS URL',
            ],
            [
              '2. 轮询（仅 async）',
              'GET 对应创建端点/{task_id}',
              'status: queued / in_progress / completed / failed',
            ],
            [
              '3. 取图',
              '响应 data[0].url',
              '同步直接返回；异步 completed 后取；或 GET /v1/images/{task_id}/content',
            ],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>出图模式</h3>
        <DocsTable
          headers={['模式', '适用', '说明']}
          rows={[
            [
              '同步 sync',
              '模型文档标记为同步',
              '省略 async 或传 false；单次返回 data[].url',
            ],
            [
              '异步 async',
              '模型文档标记为异步',
              '传 async: true；先返回 task_id，再 GET 轮询',
            ],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>对外接口</h3>
        <DocsTable
          headers={['端点', '说明']}
          rows={[
            ['POST /v1/images/generations', '文生图（JSON，sync/async）'],
            [
              'POST /v1/images/edits',
              '参考图 / 蒙版编辑（URL-only JSON，sync/async）',
            ],
            ['GET /v1/images/generations/{task_id}', '异步任务查询'],
            ['GET /v1/images/edits/{task_id}', '异步编辑任务查询'],
            ['GET /v1/images/{task_id}/content', '下载图片（部分模型）'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>快速示例</h3>
        <CodeBlock
          title='同步文生图（canonical JSON）'
          code={`curl -X POST ${base}/images/generations \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "nano-banana-pro-4k",
    "prompt": "一只橘猫趴在窗台上晒太阳，水彩画风格",
    "size": "1:1",
    "n": 1,
    "response_format": "url",
    "stream": false
  }'`}
        />
        <CodeBlock
          title='参考图编辑（URL-only JSON）'
          code={`curl -X POST ${base}/images/edits \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "nano-banana-pro-4k",
    "prompt": "将参考图风格应用到新场景",
    "size": "16:9",
    "images": [
      "https://cdn.example.com/reference-1.png",
      "https://cdn.example.com/reference-2.png"
    ],
    "response_format": "url",
    "stream": false
  }'`}
        />
        <CodeBlock
          title='异步文生图'
          code={`curl -X POST ${base}/images/generations \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2-1k",
    "prompt": "一只橘猫趴在窗台上晒太阳，水彩画风格",
    "size": "1024x1024",
    "quality": "medium",
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

        <h3 className='mt-8 text-lg font-semibold'>字段规则</h3>
        <DocsTable
          headers={['场景', '说明']}
          rows={[
            [
              '模型独立文档',
              '每个模型独立声明 endpoints、示例、支持字段和取值范围；未列出的字段不要发送。',
            ],
            [
              '标准字段',
              '公共字段为 model、prompt、size、quality、n、background、output_format、output_compression、moderation、response_format、images、mask、async、stream。',
            ],
            [
              'size',
              '统一承载画幅比例或精确尺寸，例如 1:1、16:9、1024x1024；具体可选值以模型文档为准。',
            ],
            [
              '参考素材',
              '客户端先将本地图片直传对象存储，再向 /v1/images/edits 提交 images HTTPS URL 数组；mask 同样传 HTTPS URL。',
            ],
            [
              '字段边界',
              '只发送所选模型独立文档列出的字段；历史别名和渠道专属字段不会出现在新客户文档中。',
            ],
            ['返回格式', 'response_format 推荐 url，避免 base64 放大响应体。'],
            [
              'sync / async',
              '同步与异步使用相同字段；异步仅增加 async: true，并按创建端点轮询。',
            ],
            ['计费', '仅成功出图才计费；具体单价与计费单位以模型广场为准。'],
          ]}
        />
      </DocsSection>

      <DocsSection
        id='api-audio-api'
        title='音乐 / 音频生成 API'
        description='音频模型共用入口与任务生命周期；每个上架模型独立维护自己的完整请求文档。'
      >
        <p className='text-muted-foreground text-sm'>{PRICING_NOTE}</p>

        <p className='text-sm'>
          鉴权：
          <code className='text-sm'>Authorization: Bearer sk-你的令牌</code>
          。模型名与模型广场展示名一致。
        </p>

        <h3 className='text-lg font-semibold'>调用流程</h3>
        <DocsTable
          headers={['步骤', '方法', '说明']}
          rows={[
            [
              '1. 提交任务',
              'POST /v1/audio/generations',
              'application/json；字段与同步/异步方式以所选模型文档为准',
            ],
            [
              '2. 轮询进度',
              'GET /v1/audio/generations/{task_id}',
              'status: queued / in_progress / completed / failed',
            ],
            [
              '3. 下载音频',
              'GET data[0].url',
              '返回 first-party CDN 地址，无需再带 Authorization',
            ],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>对外接口</h3>
        <DocsTable
          headers={['端点', '说明']}
          rows={[
            [
              'POST /v1/audio/generations',
              '所有音乐模型统一入口（默认 async=true）',
            ],
            ['GET /v1/audio/generations/{task_id}', '查询任务状态与结果 URL'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>Canonical 字段</h3>
        <DocsTable
          headers={['字段', '说明']}
          rows={[
            ['model / prompt', '必填：模型广场 public 名与音频描述'],
            ['response_format', '统一返回格式字段'],
            ['async', '统一同步 / 异步开关'],
            ['stream', '统一流式开关；仅模型明确支持时使用'],
          ]}
        />

        <p className='text-sm'>
          上表是统一入站 schema；提交 JSON
          时使用下方单模型文档声明的支持子集与完整示例。
        </p>
        <CodeBlock
          title='轮询取音频'
          code={`curl ${base}/audio/generations/{task_id} \\
  -H "Authorization: Bearer sk-xxx"`}
        />

        <ul className='list-disc space-y-2 pl-5'>
          <li>音乐生成通常 30–60 秒，轮询间隔建议 5–10 秒</li>
          <li>仅成功出片才计费；失败不扣费</li>
          <li>结果 URL 为平台 CDN，不会暴露上游下载域</li>
          <li>字段、输出格式与同步/异步方式只以所选模型的独立文档为准</li>
        </ul>
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

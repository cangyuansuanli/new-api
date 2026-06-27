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
            ['1. 提交任务', 'POST /v1/videos', 'JSON 或 multipart；上游差异由平台内部适配'],
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
    "model": "Seedance2.0-720p",
    "prompt": "雨夜霓虹街道，镜头缓慢推进，电影感光影",
    "duration": 8,
    "metadata": {
      "ratio": "16:9",
      "resolution": "720p"
    }
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
          <li>Omni / Veo 等 Gemini 视频模型内容审查较严，避免真人正脸、版权 IP、敏感题材</li>
        </ul>
      </DocsSection>

      <DocsSection
        id='api-image-api'
        title='图像生成 API'
        description='图像分两种出图模式：画布及多数模型为异步（async: true + 轮询）；部分模型为同步单次返回。各模型具体模式与参数见下方弹窗。'
      >
        <p className='text-muted-foreground text-sm'>{PRICING_NOTE}</p>

        <p className='text-sm'>
          鉴权：<code className='text-sm'>Authorization: Bearer sk-你的令牌</code>。模型名与模型广场展示名一致。
        </p>

        <h3 className='text-lg font-semibold'>调用流程</h3>
        <DocsTable
          headers={['步骤', '方法', '说明']}
          rows={[
            ['1. 提交任务', 'POST /v1/images/generations', 'JSON body 中 async: true'],
            ['2. 轮询进度', 'GET /v1/images/generations/{task_id}', 'status: queued / in_progress / completed / failed'],
            ['3. 取图', '响应 data[0].url', '或 GET /v1/images/{task_id}/content'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>出图模式</h3>
        <DocsTable
          headers={['模式', '适用', '说明']}
          rows={[
            ['异步 async', '画布、多数图像模型', 'POST 带 async: true，再 GET 轮询 task_id'],
            ['同步 sync', '部分轻量模型', '单次 POST 直接返回 data.url，勿传 async'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>对外接口（异步）</h3>
        <DocsTable
          headers={['端点', '说明']}
          rows={[
            ['POST /v1/images/generations', '统一入口（async: true 走异步）'],
            ['GET /v1/images/generations/{task_id}', '查询任务状态'],
            ['GET /v1/images/{task_id}/content', '下载图片（部分模型）'],
          ]}
        />

        <h3 className='mt-8 text-lg font-semibold'>快速示例</h3>
        <CodeBlock
          title='异步文生图（画布默认）'
          code={`curl -X POST ${base}/images/generations \\
  -H "Authorization: Bearer sk-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "一只橘猫趴在窗台上晒太阳，水彩画风格",
    "size": "1024x1024",
    "n": 1,
    "async": true
  }'`}
        />
        <CodeBlock
          title='轮询取图'
          code={`curl ${base}/images/generations/{task_id} \\
  -H "Authorization: Bearer sk-xxx"`}
        />

        <ul className='list-disc space-y-2 pl-5'>
          <li>画布生图为异步任务，轮询间隔建议 3–5 秒，客户端超时建议 ≥120 秒（大图 ≥300 秒）</li>
          <li>completed 后从 data[0].url 取图；status 为 failed 时查看 error.message</li>
          <li>仅成功出图才计费</li>
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

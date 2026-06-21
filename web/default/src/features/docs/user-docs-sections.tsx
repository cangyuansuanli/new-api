/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { Link } from '@tanstack/react-router'
import { CopyBlock } from './components/copy-block'
import { DocsSection } from './components/docs-section'
import { DEFAULT_CANVAS_BASE_URL } from '@/features/canvas/lib/canvas-config'

type UserDocsSectionsProps = {
  siteOrigin: string
  siteName: string
}

const integrationTools = [
  {
    name: 'OpenAI Codex CLI',
    href: 'https://developers.openai.com/codex/cli',
    note: '在配置中填写 Base URL 与 API Key，模型名从模型广场选择。',
  },
  {
    name: 'Claude Code',
    href: 'https://docs.anthropic.com/en/docs/claude-code',
    note: '支持 Anthropic 兼容配置；也可通过 CC Switch 统一管理供应商。',
  },
  {
    name: 'Cherry Studio',
    href: 'https://www.cherry-ai.com/',
    note: '在「模型服务」中添加 OpenAI 兼容供应商，填入本站地址与 Key。',
  },
  {
    name: 'CC Switch',
    href: 'https://github.com/farion1231/cc-switch',
    note: '适合同时管理 Codex、Claude Code、Gemini CLI 等多工具的 API 配置。',
  },
]

export function UserDocsSections(props: UserDocsSectionsProps) {
  const apiTemplate = `Base URL: ${props.siteOrigin}\nAPI Key: sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

  return (
    <>
      <DocsSection
        id='overview'
        title='平台概览'
        description='沧元算力是统一的 AI 模型网关。注册并创建 API Key 后，可在控制台、第三方客户端或无限画布中使用同一套接口与计费。'
      >
        <p>
          大多数 OpenAI 兼容工具只需要两项信息：<strong>Base URL</strong>（接口地址）和{' '}
          <strong>API Key</strong>（你在控制台创建的令牌）。请求会经本站转发到对应模型渠道，费用按模型规则从账户余额扣除。
        </p>
        <CopyBlock label='客户端通用填写模板' value={apiTemplate} />
        <ul className='list-disc space-y-2 pl-5'>
          <li>
            Base URL 通常为本站域名，例如 <code className='text-sm'>{props.siteOrigin}</code>
          </li>
          <li>API Key 在控制台「API 密钥」页面创建，完整 Key 仅显示一次，请立即保存</li>
          <li>建议不同设备或不同软件使用不同 Key，便于统计用量与单独停用</li>
        </ul>
      </DocsSection>

      <DocsSection
        id='quick-start'
        title='快速开始'
        description='第一次使用建议按顺序完成以下步骤。'
      >
        <ol className='list-decimal space-y-3 pl-5'>
          <li>
            在首页点击「注册」或「登录」，完成账号验证（若站点开启了邮箱验证）。
          </li>
          <li>
            登录后进入{' '}
            <Link to='/dashboard' className='text-primary font-medium hover:underline'>
              控制台
            </Link>
            ，确认账户状态正常。
          </li>
          <li>
            打开{' '}
            <Link to='/keys' className='text-primary font-medium hover:underline'>
              API 密钥
            </Link>
            ，创建并保存你的第一个 Key。
          </li>
          <li>
            在{' '}
            <Link to='/wallet' className='text-primary font-medium hover:underline'>
              钱包
            </Link>{' '}
            充值或兑换额度，确保余额足够调用模型。
          </li>
          <li>
            浏览{' '}
            <Link to='/pricing' className='text-primary font-medium hover:underline'>
              模型广场
            </Link>
            ，了解可用模型与价格，再选择客户端或控制台功能开始创作。
          </li>
        </ol>
      </DocsSection>

      <DocsSection
        id='api-key'
        title='创建与管理 API Key'
        description='API Key 是调用模型的凭证，可在控制台独立配置权限与额度上限。'
      >
        <p>
          路径：登录后左侧菜单 → <strong>API 密钥</strong>，或直接访问{' '}
          <Link to='/keys' className='text-primary hover:underline'>
            /keys
          </Link>
          。
        </p>
        <h3 className='text-lg font-semibold'>创建 Key</h3>
        <ol className='list-decimal space-y-2 pl-5'>
          <li>点击「创建密钥」，填写名称（建议按用途命名，如「Cherry Studio」「Codex CLI」）。</li>
          <li>按需设置过期时间、剩余配额、可用模型范围、IP 白名单或分组。</li>
          <li>提交后立即复制完整 Key；关闭弹窗后无法再次查看完整内容。</li>
        </ol>
        <div className='bg-muted/40 border-border/50 rounded-lg border px-4 py-3 text-sm'>
          <strong>安全提示：</strong>请勿将 Key 提交到公开仓库、截图或发给他人。泄露后请立即删除旧 Key 并新建。
        </div>
        <h3 className='text-lg font-semibold'>编辑与删除</h3>
        <p>
          在列表中可编辑配额与模型限制；删除后 Key 立即失效且不可恢复。若某个软件不再使用，建议删除对应 Key 而非长期闲置。
        </p>
      </DocsSection>

      <DocsSection
        id='wallet'
        title='钱包与计费'
        description='账户余额用于支付模型调用；不同模型可能按 Token、按次或按秒计费。'
      >
        <p>
          在{' '}
          <Link to='/wallet' className='text-primary hover:underline'>
            钱包
          </Link>{' '}
          可查看余额、充值记录，并使用站点支持的支付方式或兑换码充值。
        </p>
        <ul className='list-disc space-y-2 pl-5'>
          <li>
            <strong>按 Token 计费：</strong>常见于文本对话模型，用量与输入/输出长度相关。
          </li>
          <li>
            <strong>按次计费：</strong>常见于生图、单次视频任务等，界面可能显示「/ 次」或「/ image」等单位。
          </li>
          <li>
            <strong>按秒计费：</strong>部分视频模型按生成时长计价，价格可能显示为「/ 秒」或带秒数预估。
          </li>
        </ul>
        <p>
          在模型广场与生成按钮上看到的预估费用来自当前模型定价；实际扣费以使用日志为准。可在控制台查看{' '}
          <Link to='/usage-logs' className='text-primary hover:underline'>
            使用日志
          </Link>{' '}
          核对每次请求。
        </p>
      </DocsSection>

      <DocsSection
        id='models'
        title='模型广场'
        description='在模型广场浏览可用模型、价格与说明，再复制模型名到客户端或控制台使用。'
      >
        <p>
          打开{' '}
          <Link to='/pricing' className='text-primary hover:underline'>
            模型广场
          </Link>
          ，可按厂商、能力或价格筛选。点击模型卡片可查看详细计费规则与支持的接口类型。
        </p>
        <ul className='list-disc space-y-2 pl-5'>
          <li>客户端里的「模型名称」应填写模型广场中显示的名称（或你的 Key 有权限访问的别名）。</li>
          <li>若 Key 设置了「模型限制」，只能调用列表内的模型。</li>
          <li>分组不同可能导致可用模型与价格倍率不同，以控制台实际展示为准。</li>
        </ul>
      </DocsSection>

      <DocsSection
        id='chat-image-video'
        title='对话、生图与视频'
        description='除第三方客户端外，也可直接在控制台使用部分 AI 能力。'
      >
        <ul className='list-disc space-y-2 pl-5'>
          <li>
            <strong>对话：</strong>若站点开启聊天功能，可在控制台进入对话页，选择模型后进行多轮问答。
          </li>
          <li>
            <strong>生图：</strong>支持 OpenAI 兼容生图接口的模型，可在支持该能力的客户端或集成应用中调用；部分站点首页提供免费生图体验入口。
          </li>
          <li>
            <strong>视频：</strong>视频模型通常耗时更长，提交后可在任务或使用记录中查看进度；不同模型对参考图、时长、比例的支持不同，以模型广场说明为准。
          </li>
        </ul>
        <p>
          第三方软件接入时，仍使用同一 Base URL 与 Key；只需在软件中选择对应能力（Chat、Images、Videos 等）并填写正确模型名即可。
        </p>
      </DocsSection>

      <DocsSection
        id='integrations'
        title='第三方工具接入'
        description='以下工具均通过 OpenAI 或 Anthropic 兼容方式连接本站，填写方式与上文模板相同。'
      >
        <div className='space-y-4'>
          {integrationTools.map((tool) => (
            <div key={tool.name} className='border-border/50 rounded-xl border p-4'>
              <h3 className='font-semibold'>
                <a
                  href={tool.href}
                  target='_blank'
                  rel='noopener noreferrer'
                  className='text-primary hover:underline'
                >
                  {tool.name}
                </a>
              </h3>
              <p className='text-muted-foreground mt-1 text-sm'>{tool.note}</p>
            </div>
          ))}
        </div>
        <p>
          安装与环境准备（Node.js、Git 等）请参考各工具官方文档。若同时维护多个 CLI，推荐使用 CC Switch 统一管理 Base URL 与 Key，避免重复手改配置。
        </p>
      </DocsSection>

      <DocsSection
        id='infinite-canvas'
        title='无限画布'
        description='无限画布是视觉创作工作台，可编排节点、生成图片与视频；接入本站时仍使用你在控制台创建的 API Key 计费。'
      >
        <p>
          无限画布是<strong>可选创作工具之一</strong>，与 Codex、Cherry Studio 等并列——并非唯一用法。适合需要节点编排、参考图编辑、批量生图/生视频的场景。
        </p>
        <h3 className='text-lg font-semibold'>从控制台打开</h3>
        <ol className='list-decimal space-y-2 pl-5'>
          <li>登录 {props.siteName} 控制台。</li>
          <li>在首页或顶部入口点击「打开无限画布」。</li>
          <li>选择要带入画布的 API Key（可按生图、视频等能力分别选择）。</li>
          <li>确认后在新标签页打开画布，网关地址与 Key 会自动填入配置。</li>
        </ol>
        <p>
          也可直接访问画布站点（默认{' '}
          <a
            href={DEFAULT_CANVAS_BASE_URL}
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary hover:underline'
          >
            {DEFAULT_CANVAS_BASE_URL}
          </a>
          ）。若已登录本站，会通过互信登录关联画布账号，<strong>不会</strong>自动携带 API Key，仍需在画布内选择 Key 或手动配置。
        </p>
        <p>
          画布详细操作见{' '}
          <a
            href={`${DEFAULT_CANVAS_BASE_URL}/docs/tutorials/quick-guide`}
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary hover:underline'
          >
            无限画布快速指南
          </a>
          ；Seedance 等视频模型说明见同站文档中的操作手册章节。
        </p>
      </DocsSection>

      <DocsSection id='faq' title='常见问题'>
        <div className='space-y-6'>
          <div>
            <h3 className='font-semibold'>提示 Key 无效或 401？</h3>
            <p className='text-muted-foreground mt-1'>
              检查 Key 是否复制完整、是否已删除或过期、是否超出配额；确认 Base URL 为本站地址且未多余添加路径。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>客户端里看不到模型？</h3>
            <p className='text-muted-foreground mt-1'>
              先在模型广场确认模型名称；若 Key 启用了模型限制，只能使用允许列表内的模型。部分客户端需手动填写模型名而非下拉选择。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>余额充足但仍无法调用？</h3>
            <p className='text-muted-foreground mt-1'>
              查看 Key 剩余配额、分组是否可用、模型是否维护中；在使用日志中查看具体错误信息。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>视频或生图等很久没结果？</h3>
            <p className='text-muted-foreground mt-1'>
              长任务属于正常现象，请勿重复提交相同请求；可在使用记录或客户端任务列表中查看状态。参考图需为可访问的 HTTPS 链接时，请确保素材 URL 有效。
            </p>
          </div>
          <div>
            <h3 className='font-semibold'>需要部署或二次开发文档？</h3>
            <p className='text-muted-foreground mt-1'>
              本站文档面向终端用户。New API 开源项目的安装、API 与管理接口说明请参阅{' '}
              <a
                href='https://docs.newapi.pro/zh/docs'
                target='_blank'
                rel='noopener noreferrer'
                className='text-primary hover:underline'
              >
                官方开发文档
              </a>
              （自建站管理员适用）。
            </p>
          </div>
        </div>
      </DocsSection>
    </>
  )
}

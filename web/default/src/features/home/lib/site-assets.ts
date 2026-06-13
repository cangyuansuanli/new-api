/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

const CDN_BASE = (
  import.meta.env.VITE_STATIC_CDN || 'https://assets.cangyuansuanli.cn'
).replace(/\/$/, '')

export function siteAsset(path: string) {
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${CDN_BASE}${normalized}`
}

export const SITE_BRAND = {
  name: '沧元算力',
  logo: siteAsset('/site/logo.png'),
  domain: 'cangyuansuanli.cn',
} as const

export const INSPIRATION_SLIDES = [
  {
    id: '01',
    image: siteAsset('/home/inspiration/01-city-poster.jpg'),
    title: '城市文字旅行海报',
    tags: '海报 / 城市 / 排版',
  },
  {
    id: '02',
    image: siteAsset('/home/inspiration/02-portrait.jpg'),
    title: '便利店写真人像',
    tags: '写真 / 人物 / 氛围',
  },
  {
    id: '03',
    image: siteAsset('/home/inspiration/03-product-blueprint.jpg'),
    title: '产品结构蓝图',
    tags: '产品 / 科技 / 蓝图',
  },
  {
    id: '04',
    image: siteAsset('/home/inspiration/04-campaign-kv.jpg'),
    title: '视觉活动主 KV',
    tags: '商业 / 创意 / 风格',
  },
  {
    id: '05',
    image: siteAsset('/home/inspiration/05-architecture.jpg'),
    title: '空间结构概念图',
    tags: '建筑 / 空间 / 概念',
  },
  {
    id: '06',
    image: siteAsset('/home/inspiration/06-document-cover.jpg'),
    title: '文档封面设计',
    tags: '封面 / 文档 / 质感',
  },
  {
    id: '07',
    image: siteAsset('/home/inspiration/07-gallery-wall.jpg'),
    title: '作品集展示墙',
    tags: '合集 / 灵感 / 展示',
  },
  {
    id: '08',
    image: siteAsset('/home/inspiration/08-ui-experiment.jpg'),
    title: '界面视觉实验',
    tags: '界面 / 科技 / 交互',
  },
] as const

export const SITE_ASSETS = {
  logo: SITE_BRAND.logo,
  tools: {
    claudeCode: siteAsset('/home/tools/claude-code.png'),
    codexCli: siteAsset('/home/tools/codex-cli.png'),
    geminiCli: siteAsset('/home/tools/gemini-cli.png'),
    imageApi: siteAsset('/home/tools/image-api.png'),
  },
} as const

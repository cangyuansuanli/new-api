/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your option)
any later version.
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { PricingModel } from '../types'
import { buildModelApiDoc } from './model-api-doc'

type MediaKind = 'image' | 'video' | 'audio'

function mediaModel(kind: MediaKind, params: string[]): PricingModel {
  const endpoint = {
    image: 'openai-image',
    video: 'openai-video',
    audio: 'openai-audio',
  }[kind]
  return {
    id: 1,
    model_name: `${kind}-model`,
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    supported_endpoint_types: [endpoint],
    api_doc: {
      dispatch_mode: kind === 'image' ? 'sync' : 'async',
      intro: `${kind} model-owned documentation`,
      endpoints: [
        {
          method: 'POST',
          path: `{{base}}/${kind}/owned-endpoint`,
          description: 'model-owned endpoint',
        },
      ],
      request_json: Object.fromEntries(params.map((name) => [name, 'value'])),
      params: params.map((name) => ({
        name,
        description: `${name} model-owned note`,
      })),
      create_response_json: { ok: true },
    },
  }
}

describe('buildModelApiDoc media ownership', () => {
  for (const [kind, ownParams, forbiddenParams] of [
    ['image', ['model', 'prompt', 'size'], ['quality', 'images', 'async']],
    [
      'video',
      ['model', 'prompt', 'duration'],
      ['resolution', 'reference_image_urls'],
    ],
    ['audio', ['model', 'prompt'], ['async', 'response_format', 'stream']],
  ] as const) {
    test(`${kind} renders only its own api_doc`, () => {
      const doc = buildModelApiDoc(
        mediaModel(kind, [...ownParams]),
        'https://api.example.com'
      )
      assert.ok(doc)
      const variant = doc.variants[0]

      assert.deepEqual(
        variant.params.map((row) => row.name),
        ownParams
      )
      assert.deepEqual(Object.keys(JSON.parse(variant.requestJson)), ownParams)
      assert.equal(
        variant.endpoints[0].path,
        `https://api.example.com/v1/${kind}/owned-endpoint`
      )
      for (const name of forbiddenParams) {
        assert.equal(
          variant.params.some((row) => row.name === name),
          false
        )
      }
    })

    test(`${kind} without api_doc has no generated documentation`, () => {
      const model = mediaModel(kind, [...ownParams])
      model.api_doc = undefined
      assert.equal(buildModelApiDoc(model), null)
    })
  }
})

test('async image docs never expose stream or multipart compatibility', () => {
  const model = mediaModel('image', ['model', 'prompt', 'async'])
  model.api_doc = {
    dispatch_mode: 'async',
    intro: 'canonical JSON only',
    endpoints: [
      { method: 'POST', path: '{{base}}/images/generations', description: 'JSON' },
    ],
    request_json: { model: 'banana-pro-2k', prompt: 'test', async: true },
    params: [
      { name: 'model', description: '' },
      { name: 'prompt', description: '' },
      { name: 'async', description: '' },
    ],
    create_response_json: { status: 'queued' },
  }
  const doc = buildModelApiDoc(model, 'https://api.example.com')
  assert.ok(doc)
  const variant = doc.variants[0]
  const rendered = JSON.stringify(variant)
  assert.equal(rendered.includes('stream'), false)
  assert.equal(rendered.toLowerCase().includes('multipart'), false)
})

test('video first/last frame docs say HTTPS URL, not JSON object', () => {
  const model = mediaModel('video', [
    'model',
    'prompt',
    'first_image_url',
    'last_image_url',
  ])
  model.api_doc = {
    dispatch_mode: 'async',
    intro: 'omni frame',
    endpoints: [
      { method: 'POST', path: '{{base}}/videos', description: 'create' },
    ],
    request_json: {
      model: 'omni-fast-no-water',
      prompt: 'test',
      first_image_url: 'https://cdn.example.com/first.png',
      last_image_url: 'https://cdn.example.com/last.png',
    },
    params: [
      { name: 'model', description: '必填' },
      { name: 'prompt', description: '描述' },
      {
        name: 'first_image_url',
        description: '首帧参考图（JSON；可与 last_image_url 单独或成对使用）。',
      },
      { name: 'last_image_url', description: '末帧参考图（JSON）。' },
    ],
    create_response_json: { status: 'queued' },
  }
  const doc = buildModelApiDoc(model, 'https://api.example.com')
  assert.ok(doc)
  const byName = Object.fromEntries(
    doc.variants[0].params.map((row) => [row.name, row.description])
  )
  assert.equal(
    byName.first_image_url,
    '首帧 HTTPS URL 字符串；可单独使用，也可与 last_image_url 成对使用。'
  )
  assert.equal(byName.last_image_url, '末帧 HTTPS URL 字符串。')
  assert.equal(byName.first_image_url.toLowerCase().includes('json'), false)
  assert.equal(byName.last_image_url.toLowerCase().includes('json'), false)
})

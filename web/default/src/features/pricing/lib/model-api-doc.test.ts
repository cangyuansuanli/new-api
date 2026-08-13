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

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

function videoModel(): PricingModel {
  return {
    id: 1,
    model_name: 'grok-video-1.5',
    quota_type: 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    supported_endpoint_types: ['openai-video'],
    video_ui_params: {
      params: {
        duration: { enabled: true, numericOptions: [4, 8] },
        ratio: { enabled: true, options: [{ value: '16:9' }] },
        resolution: { enabled: true, options: [{ value: '720p' }] },
      },
      referenceLimits: { images: 1, videos: 0, audios: 0 },
    },
    api_doc: {
      intro: 'model capability',
      doc_params_json: [
        { name: 'seconds', description: 'legacy duration alias' },
        { name: 'images', description: 'legacy image alias' },
        { name: 'image_url', description: 'legacy image alias' },
        { name: 'input_reference', description: 'legacy multipart alias' },
        { name: 'size', description: 'unsupported fallback' },
      ],
    },
  }
}

describe('buildModelApiDoc video contract', () => {
  test('renders only canonical profile-supported request fields', () => {
    const doc = buildModelApiDoc(videoModel(), 'https://api.example.com')
    const variant = doc.variants[0]

    assert.deepEqual(
      variant.params.map((row) => row.name),
      [
        'model',
        'prompt',
        'duration',
        'aspect_ratio',
        'resolution',
        'reference_image_urls',
      ]
    )
    assert.deepEqual(Object.keys(JSON.parse(variant.requestJson)), [
      'model',
      'prompt',
      'duration',
      'aspect_ratio',
      'resolution',
      'reference_image_urls',
    ])
  })

  test('does not invent optional parameters when the profile disables them', () => {
    const model = videoModel()
    model.video_ui_params = { params: {}, referenceLimits: {} }
    const variant = buildModelApiDoc(model).variants[0]

    assert.deepEqual(
      variant.params.map((row) => row.name),
      ['model', 'prompt']
    )
    assert.deepEqual(Object.keys(JSON.parse(variant.requestJson)), [
      'model',
      'prompt',
    ])
  })
})

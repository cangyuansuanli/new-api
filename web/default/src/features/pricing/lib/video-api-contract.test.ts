/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your option)
any later version.
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { mergeVideoCapabilityParams } from './video-api-contract'

describe('mergeVideoCapabilityParams', () => {
  test('does not append legacy aliases or model-specific fallback fields', () => {
    const result = mergeVideoCapabilityParams(
      [
        { name: 'model', description: 'model' },
        { name: 'prompt', description: 'prompt' },
        { name: 'duration', description: 'duration' },
        {
          name: 'reference_image_urls',
          description: 'canonical images',
        },
      ],
      [
        { name: 'seconds', description: 'legacy duration' },
        { name: 'images', description: 'legacy images' },
        { name: 'image_url', description: 'legacy image' },
        { name: 'input_reference', description: 'legacy multipart' },
        { name: 'vendor_option', description: 'provider option' },
      ]
    )

    assert.deepEqual(
      result.map((row) => row.name),
      ['model', 'prompt', 'duration', 'reference_image_urls']
    )
  })

  test('only overrides descriptions for canonical fields already generated', () => {
    const result = mergeVideoCapabilityParams(
      [
        { name: 'model', description: 'model' },
        { name: 'prompt', description: 'prompt' },
        { name: 'resolution', description: 'generated resolution' },
      ],
      [
        { name: 'resolution', description: 'model resolution limits' },
        { name: 'seed', description: 'must not be appended' },
      ]
    )

    assert.deepEqual(result, [
      { name: 'model', description: 'model' },
      { name: 'prompt', description: 'prompt' },
      { name: 'resolution', description: 'model resolution limits' },
    ])
  })

  test('allows explicitly declared canonical frame controls', () => {
    const result = mergeVideoCapabilityParams(
      [
        { name: 'model', description: 'model' },
        { name: 'prompt', description: 'prompt' },
      ],
      [
        { name: 'first_image_url', description: 'first frame' },
        { name: 'last_image_url', description: 'last frame' },
      ]
    )

    assert.deepEqual(
      result.map((row) => row.name),
      ['model', 'prompt', 'first_image_url', 'last_image_url']
    )
  })
})

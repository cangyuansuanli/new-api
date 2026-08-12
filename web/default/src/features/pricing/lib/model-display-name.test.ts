import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { PricingModel } from '../types'
import {
  formatModelDisplayName,
  groupPricingModelsByDisplayName,
} from './model-display-name'

describe('Happy House marketplace names', () => {
  test('keeps the model family in public names', () => {
    assert.equal(formatModelDisplayName('happyhouse-1.0'), 'happyhouse-1.0')
    assert.equal(formatModelDisplayName('happyhouse-1.1'), 'happyhouse-1.1')
  })

  test('keeps Happy House versions as separate marketplace entries', () => {
    const models = [
      { model_name: 'happyhouse-1.0' },
      { model_name: 'happyhouse-1.1' },
    ] as PricingModel[]

    assert.deepEqual(
      groupPricingModelsByDisplayName(models).map(
        (model) => model.display_name
      ),
      ['happyhouse-1.0', 'happyhouse-1.1']
    )
  })
})

describe('Seedance public route prefixes', () => {
  test('keeps sd5, sd6 and sd7 in marketplace and API document names', () => {
    assert.equal(formatModelDisplayName('sd5-seedance-2.0'), 'sd5-seedance-2.0')
    assert.equal(
      formatModelDisplayName('sd6-seedance-2.0-720p'),
      'sd6-seedance-2.0-720p'
    )
    assert.equal(
      formatModelDisplayName('sd6-seedance-2.0-1080p'),
      'sd6-seedance-2.0-1080p'
    )
    assert.equal(
      formatModelDisplayName('sd7-seedance-2.0-720p'),
      'sd7-seedance-2.0-720p'
    )
    assert.equal(
      formatModelDisplayName('sd7-seedance-2.0-1080p'),
      'sd7-seedance-2.0-1080p'
    )
  })

  test('does not merge sd6 products into unprefixed Seedance entries', () => {
    const models = [
      { model_name: 'seedance-2.0-720p' },
      { model_name: 'sd6-seedance-2.0-720p' },
      { model_name: 'sd6-seedance-2.0-1080p' },
    ] as PricingModel[]

    assert.deepEqual(
      groupPricingModelsByDisplayName(models).map(
        (model) => model.display_name
      ),
      ['sd6-seedance-2.0-1080p', 'sd6-seedance-2.0-720p', 'seedance-2.0-720p']
    )
  })
})

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import type { PricingModel } from '../types'
import {
  formatModelDisplayName,
  groupPricingModelsByDisplayName,
} from './model-display-name'

describe('Public model names', () => {
  test('marketplace display uses pricing public names without stripping prefixes', () => {
    const models = [
      {
        model_name: 'nano-banana-pro',
        display_name: 'nano-banana-pro',
      },
    ] as PricingModel[]

    assert.deepEqual(groupPricingModelsByDisplayName(models), [
      {
        ...models[0],
        display_name: 'nano-banana-pro',
        model_aliases: ['nano-banana-pro'],
        enable_groups: [],
      },
    ])
  })
})

describe('Nano Banana public names', () => {
  test('keeps nano- prefix in marketplace display names', () => {
    assert.equal(formatModelDisplayName('nano-banana-pro'), 'nano-banana-pro')
    assert.equal(formatModelDisplayName('nano-banana'), 'nano-banana')
    assert.equal(formatModelDisplayName('nano-banana-pro-4k'), 'nano-banana-pro-4k')
  })
})

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
  test('keeps sd{N} route prefixes in marketplace and API document names', () => {
    assert.equal(formatModelDisplayName('sd4-seedance-2.0'), 'sd4-seedance-2.0')
    assert.equal(formatModelDisplayName('sd4-seedance-2.0-fast'), 'sd4-seedance-2.0-fast')
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
    assert.equal(formatModelDisplayName('sd8-seedance-2.0'), 'sd8-seedance-2.0')
    assert.equal(
      formatModelDisplayName('sd8-seedance-2.0-fast'),
      'sd8-seedance-2.0-fast'
    )
    assert.equal(formatModelDisplayName('sd99-seedance-2.0'), 'sd99-seedance-2.0')
  })

  test('does not merge sd4 products into legacy seedance entries', () => {
    const models = [
      { model_name: 'seedance-2.0' },
      { model_name: 'sd4-seedance-2.0' },
      { model_name: 'sd4-seedance-2.0-fast' },
    ] as PricingModel[]

    assert.deepEqual(
      groupPricingModelsByDisplayName(models).map(
        (model) => model.display_name
      ),
      ['sd4-seedance-2.0', 'sd4-seedance-2.0-fast', 'seedance-2.0']
    )
  })

  test('does not merge sd8 products into unprefixed Seedance entries', () => {
    const models = [
      { model_name: 'seedance-2.0' },
      { model_name: 'sd8-seedance-2.0' },
      { model_name: 'sd8-seedance-2.0-fast' },
    ] as PricingModel[]

    assert.deepEqual(
      groupPricingModelsByDisplayName(models).map(
        (model) => model.display_name
      ),
      ['sd8-seedance-2.0', 'sd8-seedance-2.0-fast', 'seedance-2.0']
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

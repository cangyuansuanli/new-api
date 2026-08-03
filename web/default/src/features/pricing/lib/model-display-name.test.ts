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

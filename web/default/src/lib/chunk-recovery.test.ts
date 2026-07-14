import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { isChunkLoadError } from './chunk-recovery'

describe('chunk load error detection', () => {
  test('detects Rspack chunk load failures', () => {
    const error = new Error(
      'Loading chunk 4990 failed. (missing: /static/js/async/4990.js)'
    )
    error.name = 'ChunkLoadError'

    assert.equal(isChunkLoadError(error), true)
  })

  test('detects dynamic import and module MIME failures', () => {
    assert.equal(
      isChunkLoadError(
        new TypeError('Failed to fetch dynamically imported module')
      ),
      true
    )
    assert.equal(
      isChunkLoadError(
        new TypeError(
          'Expected a JavaScript-or-Wasm module script but the server responded with text/html'
        )
      ),
      true
    )
  })

  test('walks nested causes', () => {
    assert.equal(
      isChunkLoadError({
        message: 'Route failed',
        cause: new Error('Importing a module script failed'),
      }),
      true
    )
  })

  test('does not classify ordinary application errors as chunk failures', () => {
    assert.equal(isChunkLoadError(new Error('Cannot read property x')), false)
    assert.equal(isChunkLoadError({ response: { status: 500 } }), false)
  })
})

import { describe, expect, it } from 'vitest'

import { displayCanvasSize } from './photoProcessing'

describe('displayCanvasSize', () => {
  it.each([
    {
      label: 'portrait',
      source: { width: 1200, height: 1600 },
      expected: { width: 720, height: 960 }
    },
    {
      label: 'square',
      source: { width: 1600, height: 1600 },
      expected: { width: 960, height: 960 }
    },
    {
      label: 'landscape',
      source: { width: 1600, height: 1200 },
      expected: { width: 1280, height: 960 }
    },
    {
      label: 'wide',
      source: { width: 2400, height: 1200 },
      expected: { width: 1280, height: 640 }
    },
    {
      label: 'small',
      source: { width: 600, height: 900 },
      expected: { width: 600, height: 900 }
    }
  ])('preserves the $label source aspect ratio within display bounds', ({ source, expected }) => {
    expect(displayCanvasSize(source)).toEqual(expected)
  })
})

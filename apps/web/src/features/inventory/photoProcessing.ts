export type ProcessedPhotoVariant = {
  blob: Blob
  contentType: 'image/webp'
  sizeBytes: number
  width: number
  height: number
}

export type ProcessedInventoryPhoto = {
  full: ProcessedPhotoVariant
  display: ProcessedPhotoVariant
}

type ImageDimensions = {
  width: number
  height: number
}

type CanvasSize = {
  width: number
  height: number
}

const webpContentType = 'image/webp'
const fullMaxLongestEdge = 3840
const fullMaxSizeBytes = 10 * 1024 * 1024
const displayMaxWidth = 1280
const displayMaxHeight = 960
const displayMaxSizeBytes = 2 * 1024 * 1024
const fullQualities = [0.86, 0.82, 0.78]
const displayQualities = [0.82, 0.78]

export async function processInventoryPhoto(file: File): Promise<ProcessedInventoryPhoto> {
  validateSourceFile(file)

  const image = await createImageBitmap(file)
  try {
    const dimensions = imageDimensions(image)
    return {
      full: await createFullVariant(image, dimensions),
      display: await createDisplayVariant(image, dimensions)
    }
  } finally {
    image.close()
  }
}

function validateSourceFile(file: File): void {
  const acceptedTypes = ['image/jpeg', 'image/png', 'image/webp']
  if (!acceptedTypes.includes(file.type)) {
    throw new Error('Unsupported photo type')
  }
}

function imageDimensions(image: ImageBitmap): ImageDimensions {
  if (image.width <= 0 || image.height <= 0) {
    throw new Error('Photo dimensions are invalid')
  }

  return {
    width: image.width,
    height: image.height
  }
}

async function createFullVariant(
  image: ImageBitmap,
  dimensions: ImageDimensions
): Promise<ProcessedPhotoVariant> {
  const canvasSize = fullCanvasSize(dimensions)

  return encodeVariant(image, dimensions, canvasSize, fullQualities, fullMaxSizeBytes)
}

async function createDisplayVariant(
  image: ImageBitmap,
  dimensions: ImageDimensions
): Promise<ProcessedPhotoVariant> {
  const canvasSize = displayCanvasSize(dimensions)

  return encodeVariant(image, dimensions, canvasSize, displayQualities, displayMaxSizeBytes)
}

function fullCanvasSize(dimensions: ImageDimensions): CanvasSize {
  const longestEdge = Math.max(dimensions.width, dimensions.height)
  if (longestEdge <= fullMaxLongestEdge) {
    return {
      width: dimensions.width,
      height: dimensions.height
    }
  }

  const scale = fullMaxLongestEdge / longestEdge
  return {
    width: Math.max(1, Math.round(dimensions.width * scale)),
    height: Math.max(1, Math.round(dimensions.height * scale))
  }
}

export function displayCanvasSize(dimensions: ImageDimensions): CanvasSize {
  const scale = Math.min(
    1,
    displayMaxWidth / dimensions.width,
    displayMaxHeight / dimensions.height
  )

  return {
    width: Math.max(1, Math.round(dimensions.width * scale)),
    height: Math.max(1, Math.round(dimensions.height * scale))
  }
}

async function encodeVariant(
  image: ImageBitmap,
  sourceDimensions: ImageDimensions,
  canvasSize: CanvasSize,
  qualities: number[],
  maxSizeBytes: number
): Promise<ProcessedPhotoVariant> {
  const canvas = document.createElement('canvas')
  canvas.width = canvasSize.width
  canvas.height = canvasSize.height

  const context = canvas.getContext('2d')
  if (context === null) {
    throw new Error('Photo canvas context is unavailable')
  }

  context.drawImage(
    image,
    0,
    0,
    sourceDimensions.width,
    sourceDimensions.height,
    0,
    0,
    canvasSize.width,
    canvasSize.height
  )

  const blob = await encodeWebPWithinSize(canvas, qualities, maxSizeBytes)
  return {
    blob,
    contentType: webpContentType,
    sizeBytes: blob.size,
    width: canvasSize.width,
    height: canvasSize.height
  }
}

async function encodeWebPWithinSize(
  canvas: HTMLCanvasElement,
  qualities: number[],
  maxSizeBytes: number
): Promise<Blob> {
  let latestBlob: Blob | null = null
  for (const quality of qualities) {
    const blob = await canvasToWebPBlob(canvas, quality)
    latestBlob = blob
    if (blob.size <= maxSizeBytes) {
      return blob
    }
  }

  if (latestBlob === null) {
    throw new Error('Photo conversion failed')
  }

  throw new Error(`Converted photo exceeds ${maxSizeBytes} bytes`)
}

function canvasToWebPBlob(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob === null) {
          reject(new Error('Photo conversion failed'))
          return
        }

        if (blob.type !== webpContentType) {
          reject(new Error('Browser did not produce WebP photo output'))
          return
        }

        resolve(blob)
      },
      webpContentType,
      quality
    )
  })
}

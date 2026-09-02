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

type SourceCrop = {
  sourceX: number
  sourceY: number
  sourceWidth: number
  sourceHeight: number
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
  const crop = fullSourceCrop(dimensions)

  return encodeVariant(image, crop, canvasSize, fullQualities, fullMaxSizeBytes)
}

async function createDisplayVariant(
  image: ImageBitmap,
  dimensions: ImageDimensions
): Promise<ProcessedPhotoVariant> {
  const crop = displaySourceCrop(dimensions)
  const canvasSize = displayCanvasSize(crop)

  return encodeVariant(image, crop, canvasSize, displayQualities, displayMaxSizeBytes)
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

function fullSourceCrop(dimensions: ImageDimensions): SourceCrop {
  return {
    sourceX: 0,
    sourceY: 0,
    sourceWidth: dimensions.width,
    sourceHeight: dimensions.height
  }
}

function displaySourceCrop(dimensions: ImageDimensions): SourceCrop {
  const targetRatio = 4 / 3
  const sourceRatio = dimensions.width / dimensions.height

  if (sourceRatio > targetRatio) {
    const sourceWidth = Math.round(dimensions.height * targetRatio)
    return {
      sourceX: Math.round((dimensions.width - sourceWidth) / 2),
      sourceY: 0,
      sourceWidth,
      sourceHeight: dimensions.height
    }
  }

  const sourceHeight = Math.round(dimensions.width / targetRatio)
  return {
    sourceX: 0,
    sourceY: Math.round((dimensions.height - sourceHeight) / 2),
    sourceWidth: dimensions.width,
    sourceHeight
  }
}

function displayCanvasSize(crop: SourceCrop): CanvasSize {
  const scale = Math.min(1, displayMaxWidth / crop.sourceWidth, displayMaxHeight / crop.sourceHeight)
  const scaledWidth = Math.max(4, Math.floor(crop.sourceWidth * scale))
  const aspectUnits = Math.max(1, Math.floor(scaledWidth / 4))

  return {
    width: aspectUnits * 4,
    height: aspectUnits * 3
  }
}

async function encodeVariant(
  image: ImageBitmap,
  crop: SourceCrop,
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
    crop.sourceX,
    crop.sourceY,
    crop.sourceWidth,
    crop.sourceHeight,
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

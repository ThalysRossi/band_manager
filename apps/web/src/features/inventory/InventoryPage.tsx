import { useEffect, useRef, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import type { UseFormReturn } from 'react-hook-form'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, ChevronUp, Pencil, Plus, Save, Trash2, Upload, X } from 'lucide-react'
import type { TranslationKey } from 'i18n'
import { z } from 'zod'

import { ApiError } from '@/shared/api/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table'
import { getCurrentAccount } from '../auth/api'
import {
  createInventoryPhotoUploadRequest,
  createInventoryProduct,
  createInventoryVariant,
  deleteInventoryProduct,
  deleteInventoryVariant,
  listInventory,
  updateInventoryProduct,
  updateInventoryVariant,
  uploadInventoryPhotoVariant
} from './api'
import type {
  CreateInventoryProductRequest,
  CreateInventoryVariantRequest,
  InventoryCategory,
  InventoryPhoto,
  InventoryPhotoManifest,
  InventoryPhotoUploadResponse,
  InventoryProduct,
  InventorySize,
  InventoryVariant,
  InventoryVariantRequest,
  PhotoUploadVariantRequest,
  UpdateInventoryProductRequest,
  UpdateInventoryVariantRequest
} from './api'
import { processInventoryPhoto } from './photoProcessing'
import type { ProcessedInventoryPhoto } from './photoProcessing'

type Translate = (key: TranslationKey) => string

type InventoryPageProps = {
  accessToken: string
  translate: Translate
}

type InventoryProductFormValues = {
  name: string
  category: InventoryCategory
  variants: InventoryColourFormValues[]
}

type InventoryColourFormValues = {
  colour: string
  priceAmount: number
  costAmount: number
  sizes: InventorySizeStockFormValues[]
}

type InventorySizeStockFormValues = {
  size: InventorySize
  quantity: number
}

type InventoryProductEditFormValues = {
  name: string
  category: InventoryCategory
}

type InventoryVariantEditFormValues = InventoryVariantFormValues

type InventoryVariantFormValues = {
  size: InventorySize
  colour: string
  priceAmount: number
  costAmount: number
  quantity: number
}

type PhotoFormState =
  | {
      status: 'empty'
    }
  | {
      status: 'processing'
      fileName: string
    }
  | {
      status: 'ready'
      fileName: string
      photo: ProcessedInventoryPhoto
    }
  | {
      status: 'failed'
      message: string
    }

const inventoryCategories: InventoryCategory[] = [
  'shirt',
  'hoodie',
  'tote_bag',
  'patch',
  'sticker',
  'vinyl',
  'cd',
  'cassette',
  'accessory'
]

const clothingSizes: InventorySize[] = ['pp', 'p', 'm', 'g', 'gg', 'xgg']

const inventoryProductSchema = z.object({
  name: z.string().trim().min(1),
  category: z.enum([
    'shirt',
    'hoodie',
    'tote_bag',
    'patch',
    'sticker',
    'vinyl',
    'cd',
    'cassette',
    'accessory'
  ]),
  variants: z
    .array(
      z.object({
        colour: z.string().trim().min(1),
        priceAmount: z.number().finite().min(0),
        costAmount: z.number().finite().min(0),
        sizes: z
          .array(
            z.object({
              size: z.enum(['not_applicable', 'one_size', 'pp', 'p', 'm', 'g', 'gg', 'xgg']),
              quantity: z.number().int().min(0)
            })
          )
          .min(1)
      })
    )
    .min(1)
})

const inventoryProductEditSchema = z.object({
  name: z.string().trim().min(1),
  category: z.enum([
    'shirt',
    'hoodie',
    'tote_bag',
    'patch',
    'sticker',
    'vinyl',
    'cd',
    'cassette',
    'accessory'
  ])
})

const inventoryVariantEditSchema = z.object({
  size: z.enum(['not_applicable', 'one_size', 'pp', 'p', 'm', 'g', 'gg', 'xgg']),
  colour: z.string().trim().min(1),
  priceAmount: z.number().finite().min(0),
  costAmount: z.number().finite().min(0),
  quantity: z.number().int().min(0)
})

type UpdateProductMutationInput = {
  productID: string
  request: UpdateInventoryProductRequest
}

type UpdateVariantMutationInput = {
  variantID: string
  request: UpdateInventoryVariantRequest
}

type CreateVariantMutationInput = {
  productID: string
  request: CreateInventoryVariantRequest
}

type InventoryVariantSelection = {
  product: InventoryProduct
  variant: InventoryVariant
}

type InventoryColourGroup = {
  id: string
  colour: string
  photo: InventoryPhoto
  sizes: InventoryVariant[]
}

export function InventoryPage(props: InventoryPageProps) {
  const queryClient = useQueryClient()
  const [formStatus, setFormStatus] = useState<string>('')
  const [photoStates, setPhotoStates] = useState<Record<string, PhotoFormState>>({})
  const [previewURLs, setPreviewURLs] = useState<Record<string, string>>({})
  const [photoUploadPending, setPhotoUploadPending] = useState<boolean>(false)
  const previewURLsRef = useRef<Record<string, string>>({})
  const [editingProductID, setEditingProductID] = useState<string>('')
  const [editingVariantID, setEditingVariantID] = useState<string>('')
  const [addingVariantProductID, setAddingVariantProductID] = useState<string>('')
  const [createPanelOpen, setCreatePanelOpen] = useState<boolean | null>(null)

  const accountQuery = useQuery({
    queryKey: ['account', 'current', props.accessToken],
    queryFn: () => getCurrentAccount(props.accessToken)
  })

  const inventoryQuery = useQuery({
    queryKey: ['inventory', props.accessToken],
    queryFn: () => listInventory(props.accessToken)
  })

  const form = useForm<InventoryProductFormValues>({
    defaultValues: createInitialProductValues()
  })

  const variantFields = useFieldArray({
    control: form.control,
    name: 'variants'
  })
  const categoryField = form.register('category', { required: true })
  const createCategory = form.watch('category')

  const createMutation = useMutation({
    mutationFn: (request: CreateInventoryProductRequest) =>
      createInventoryProduct(props.accessToken, request),
    onSuccess: () => {
      form.reset(createInitialProductValues())
      Object.values(previewURLsRef.current).forEach(URL.revokeObjectURL)
      setPhotoStates({})
      setPreviewURLs({})
      setCreatePanelOpen(false)
      setFormStatus(props.translate('inventory.createSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(inventoryMutationMessage(error, props.translate))
    }
  })

  const updateProductMutation = useMutation({
    mutationFn: (input: UpdateProductMutationInput) =>
      updateInventoryProduct(props.accessToken, input.productID, input.request),
    onSuccess: () => {
      setEditingProductID('')
      setFormStatus(props.translate('inventory.updateSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(inventoryMutationMessage(error, props.translate))
    }
  })

  const createVariantMutation = useMutation({
    mutationFn: (input: CreateVariantMutationInput) =>
      createInventoryVariant(props.accessToken, input.productID, input.request),
    onSuccess: () => {
      setAddingVariantProductID('')
      setFormStatus(props.translate('inventory.addVariantSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(inventoryMutationMessage(error, props.translate))
    }
  })

  const deleteProductMutation = useMutation({
    mutationFn: (productID: string) => deleteInventoryProduct(props.accessToken, productID),
    onSuccess: () => {
      setEditingProductID('')
      setFormStatus(props.translate('inventory.deleteSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(inventoryMutationMessage(error, props.translate))
    }
  })

  const updateVariantMutation = useMutation({
    mutationFn: (input: UpdateVariantMutationInput) =>
      updateInventoryVariant(props.accessToken, input.variantID, input.request),
    onSuccess: () => {
      setEditingVariantID('')
      setFormStatus(props.translate('inventory.updateVariantSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(inventoryMutationMessage(error, props.translate))
    }
  })

  const deleteVariantMutation = useMutation({
    mutationFn: (variantID: string) => deleteInventoryVariant(props.accessToken, variantID),
    onSuccess: () => {
      setEditingVariantID('')
      setFormStatus(props.translate('inventory.deleteVariantSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(inventoryMutationMessage(error, props.translate))
    }
  })

  useEffect(() => {
    previewURLsRef.current = previewURLs
  }, [previewURLs])

  useEffect(
    () => () => {
      Object.values(previewURLsRef.current).forEach(URL.revokeObjectURL)
    },
    []
  )

  const account = accountQuery.data
  const products = inventoryQuery.data
  const canMutate = account?.activeBand.canWrite === true
  const showCreatePanel = createPanelOpen ?? products?.length === 0
  const editingProduct = findProductByID(products, editingProductID)
  const editingVariant = findInventoryVariant(products, editingVariantID)
  const addingVariantProduct = findProductByID(products, addingVariantProductID)
  const createPending =
    createMutation.isPending ||
    photoUploadPending ||
    Object.values(photoStates).some((photoState) => photoState.status === 'processing')
  const productMutationPending = updateProductMutation.isPending || deleteProductMutation.isPending
  const variantMutationPending =
    createVariantMutation.isPending ||
    updateVariantMutation.isPending ||
    deleteVariantMutation.isPending

  async function handlePhotoChange(fieldID: string, fileList: FileList | null): Promise<void> {
    const file = fileList?.[0]
    if (file === null || file === undefined) {
      setPhotoStates((current) => ({ ...current, [fieldID]: { status: 'empty' } }))
      return
    }

    setFormStatus('')
    setPhotoStates((current) => ({
      ...current,
      [fieldID]: { status: 'processing', fileName: file.name }
    }))
    try {
      const photo = await processInventoryPhoto(file)
      const oldPreviewURL = previewURLsRef.current[fieldID]
      if (oldPreviewURL !== undefined) {
        URL.revokeObjectURL(oldPreviewURL)
      }
      setPhotoStates((current) => ({
        ...current,
        [fieldID]: { status: 'ready', fileName: file.name, photo }
      }))
      setPreviewURLs((current) => ({
        ...current,
        [fieldID]: URL.createObjectURL(photo.display.blob)
      }))
    } catch {
      setPhotoStates((current) => ({
        ...current,
        [fieldID]: {
          status: 'failed',
          message: props.translate('inventory.photoInvalid')
        }
      }))
    }
  }

  async function handleCreate(values: InventoryProductFormValues): Promise<void> {
    setFormStatus('')
    const parsedValues = inventoryProductSchema.safeParse(values)
    if (!parsedValues.success) {
      setFormStatus(props.translate('inventory.formInvalid'))
      return
    }

    const colourKeys = new Set<string>()
    for (const colour of parsedValues.data.variants) {
      const normalizedColour = colour.colour.trim().toLowerCase().replace(/\s+/g, ' ')
      if (
        colourKeys.has(normalizedColour) ||
        new Set(colour.sizes.map((stock) => stock.size)).size !== colour.sizes.length
      ) {
        setFormStatus(props.translate('inventory.formInvalid'))
        return
      }
      colourKeys.add(normalizedColour)
      const sized =
        parsedValues.data.category === 'shirt' || parsedValues.data.category === 'hoodie'
      if (
        colour.sizes.some((stock) =>
          sized
            ? stock.size === 'not_applicable' || stock.size === 'one_size'
            : stock.size !== 'not_applicable'
        )
      ) {
        setFormStatus(props.translate('inventory.formInvalid'))
        return
      }
    }

    const uploadedPhotos: InventoryPhotoManifest[] = []
    setPhotoUploadPending(true)
    try {
      for (const [index] of parsedValues.data.variants.entries()) {
        const fieldID = variantFields.fields[index]?.id
        const photoState = fieldID === undefined ? undefined : photoStates[fieldID]
        if (photoState?.status !== 'ready') {
          setFormStatus(props.translate('inventory.photoRequired'))
          return
        }
        let uploadRequest: InventoryPhotoUploadResponse
        try {
          uploadRequest = await createInventoryPhotoUploadRequest(props.accessToken, {
            full: toPhotoUploadVariantRequest(photoState.photo.full),
            display: toPhotoUploadVariantRequest(photoState.photo.display)
          })
        } catch {
          setFormStatus(props.translate('inventory.photoUploadRequestFailed'))
          return
        }
        try {
          await Promise.all([
            uploadInventoryPhotoVariant(uploadRequest.uploads.full, photoState.photo.full.blob),
            uploadInventoryPhotoVariant(
              uploadRequest.uploads.display,
              photoState.photo.display.blob
            )
          ])
        } catch {
          setFormStatus(props.translate('inventory.photoUploadFailed'))
          return
        }
        uploadedPhotos.push(toPhotoManifest(uploadRequest.photo))
      }
    } finally {
      setPhotoUploadPending(false)
    }

    const variants: InventoryVariantRequest[] = parsedValues.data.variants.flatMap(
      (colour, index) =>
        colour.sizes.map((stock) => ({
          size: stock.size,
          colour: colour.colour.trim(),
          photo: uploadedPhotos[index],
          price: { amount: amountToCents(colour.priceAmount), currency: 'BRL' },
          cost: { amount: amountToCents(colour.costAmount), currency: 'BRL' },
          quantity: stock.quantity
        }))
    )
    createMutation.mutate({
      name: parsedValues.data.name.trim(),
      category: parsedValues.data.category,
      photo: uploadedPhotos[0],
      variants
    })
  }

  function handleUpdateProduct(
    product: InventoryProduct,
    values: InventoryProductEditFormValues
  ): void {
    setFormStatus('')
    const parsedValues = inventoryProductEditSchema.safeParse(values)
    if (!parsedValues.success) {
      setFormStatus(props.translate('inventory.formInvalid'))
      return
    }

    updateProductMutation.mutate({
      productID: product.id,
      request: {
        name: parsedValues.data.name.trim(),
        category: parsedValues.data.category,
        photo: toPhotoManifest(product.photo)
      }
    })
  }

  function handleDeleteProduct(product: InventoryProduct): void {
    setFormStatus('')
    if (!window.confirm(props.translate('inventory.deleteConfirm'))) {
      return
    }

    deleteProductMutation.mutate(product.id)
  }

  async function handleCreateVariant(
    product: InventoryProduct,
    values: InventoryVariantFormValues,
    file: File | null
  ): Promise<void> {
    setFormStatus('')
    const parsedValues = inventoryVariantEditSchema.safeParse(values)
    if (!parsedValues.success) {
      setFormStatus(props.translate('inventory.formInvalid'))
      return
    }

    const existingColour = product.variants.find(
      (variant) => normalizeColour(variant.colour) === normalizeColour(parsedValues.data.colour)
    )
    let photo: InventoryPhotoManifest
    if (existingColour !== undefined) {
      photo = toPhotoManifest(existingColour.photo)
    } else {
      if (file === null) {
        setFormStatus(props.translate('inventory.photoRequired'))
        return
      }
      try {
        photo = await uploadPhotoFile(props.accessToken, file)
      } catch (error) {
        setFormStatus(
          error instanceof Error ? error.message : props.translate('inventory.photoUploadFailed')
        )
        return
      }
    }
    const request = toVariantRequest(parsedValues.data, photo)
    if (existingColour !== undefined) {
      request.price = existingColour.price
      request.cost = existingColour.cost
    }
    createVariantMutation.mutate({ productID: product.id, request })
  }

  async function handleUpdateVariant(
    selection: InventoryVariantSelection,
    values: InventoryVariantEditFormValues,
    file: File | null
  ): Promise<void> {
    setFormStatus('')
    const parsedValues = inventoryVariantEditSchema.safeParse(values)
    if (!parsedValues.success) {
      setFormStatus(props.translate('inventory.formInvalid'))
      return
    }

    let photo = toPhotoManifest(selection.variant.photo)
    if (file !== null) {
      try {
        photo = await uploadPhotoFile(props.accessToken, file)
      } catch (error) {
        setFormStatus(
          error instanceof Error ? error.message : props.translate('inventory.photoUploadFailed')
        )
        return
      }
    }
    updateVariantMutation.mutate({
      variantID: selection.variant.id,
      request: toVariantRequest(parsedValues.data, photo)
    })
  }

  function handleDeleteVariant(selection: InventoryVariantSelection): void {
    setFormStatus('')
    if (!window.confirm(props.translate('inventory.deleteVariantConfirm'))) {
      return
    }

    deleteVariantMutation.mutate(selection.variant.id)
  }

  if (accountQuery.isLoading || inventoryQuery.isLoading) {
    return <StatusPanel message={props.translate('inventory.loading')} />
  }

  if (
    accountQuery.isError ||
    inventoryQuery.isError ||
    account === undefined ||
    products === undefined
  ) {
    return <StatusPanel message={props.translate('inventory.error')} />
  }

  return (
    <section className="grid gap-ui-32">
      <header className="flex flex-wrap items-start justify-between gap-ui-16">
        <div className="grid gap-ui-8">
          <h2 className="m-0 text-[1.75rem] leading-[1.15]">{props.translate('nav.inventory')}</h2>
          <p className="m-0 text-base text-white-300">
            {props.translate('inventory.productCount')}: {products.length}
          </p>
        </div>
        {canMutate ? (
          <Button
            type="button"
            variant="outline"
            aria-expanded={showCreatePanel}
            aria-controls="inventory-create-panel"
            onClick={() => setCreatePanelOpen(!showCreatePanel)}
          >
            {showCreatePanel ? (
              <ChevronUp aria-hidden="true" />
            ) : (
              <ChevronDown aria-hidden="true" />
            )}
            {props.translate('inventory.addProduct')}
          </Button>
        ) : null}
      </header>

      {formStatus === '' ? null : (
        <p className="m-0 text-sm text-white-300" role="status">
          {formStatus}
        </p>
      )}

      {canMutate ? (
        <Card
          id="inventory-create-panel"
          aria-labelledby="inventory-create-title"
          hidden={!showCreatePanel}
          className={showCreatePanel ? undefined : 'hidden'}
        >
          <CardHeader>
            <h3 id="inventory-create-title" className="m-0 text-base leading-tight">
              {props.translate('inventory.createTitle')}
            </h3>
          </CardHeader>
          <CardContent>
            <form
              className="grid gap-ui-24"
              onSubmit={(event) => {
                void form.handleSubmit(handleCreate)(event)
              }}
            >
              <div className="grid gap-ui-16 min-[800px]:grid-cols-2">
                <div className="grid gap-ui-8">
                  <Label htmlFor="inventory-product-name">
                    {props.translate('inventory.nameLabel')}
                  </Label>
                  <Input
                    id="inventory-product-name"
                    {...form.register('name', { required: true })}
                  />
                </div>

                <div className="grid gap-ui-8">
                  <Label htmlFor="inventory-product-category">
                    {props.translate('inventory.categoryLabel')}
                  </Label>
                  <select
                    id="inventory-product-category"
                    className="h-9 w-full rounded-md border border-input bg-background px-ui-12 text-sm"
                    {...categoryField}
                    onChange={(event) => {
                      void categoryField.onChange(event)
                      const category = event.currentTarget.value as InventoryCategory
                      variantFields.fields.forEach((_, index) => {
                        form.setValue(`variants.${index}.sizes`, [
                          createInitialSizeStockValues(category)
                        ])
                      })
                    }}
                  >
                    {inventoryCategories.map((category) => (
                      <option key={category} value={category}>
                        {props.translate(categoryLabelKey(category))}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid gap-ui-16">
                <div className="flex items-center justify-between gap-ui-16">
                  <h3 className="m-0 text-base leading-tight">
                    {props.translate('inventory.variantsTitle')}
                  </h3>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => variantFields.append(createInitialColourValues(createCategory))}
                  >
                    <Plus aria-hidden="true" />
                    {props.translate('inventory.addVariant')}
                  </Button>
                </div>

                <div className="grid gap-ui-16">
                  {variantFields.fields.map((field, index) => (
                    <ColourVariantCreateFields
                      key={field.id}
                      fieldID={field.id}
                      index={index}
                      form={form}
                      category={createCategory}
                      translate={props.translate}
                      photoState={photoStates[field.id] ?? { status: 'empty' }}
                      previewURL={previewURLs[field.id] ?? ''}
                      canRemove={variantFields.fields.length > 1}
                      onPhotoChange={(files) => void handlePhotoChange(field.id, files)}
                      onRemove={() => {
                        const previewURL = previewURLsRef.current[field.id]
                        if (previewURL !== undefined) URL.revokeObjectURL(previewURL)
                        setPhotoStates((current) =>
                          Object.fromEntries(
                            Object.entries(current).filter(([key]) => key !== field.id)
                          )
                        )
                        setPreviewURLs((current) =>
                          Object.fromEntries(
                            Object.entries(current).filter(([key]) => key !== field.id)
                          )
                        )
                        variantFields.remove(index)
                      }}
                    />
                  ))}
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-ui-16">
                <Button type="submit" disabled={createPending}>
                  <Save aria-hidden="true" />
                  {props.translate('inventory.createSubmit')}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      ) : null}

      {editingProduct === undefined ? null : (
        <ProductEditForm
          product={editingProduct}
          translate={props.translate}
          disabled={productMutationPending}
          onCancel={() => setEditingProductID('')}
          onSubmit={handleUpdateProduct}
        />
      )}

      {addingVariantProduct === undefined ? null : (
        <VariantCreateForm
          product={addingVariantProduct}
          translate={props.translate}
          disabled={variantMutationPending}
          onCancel={() => setAddingVariantProductID('')}
          onSubmit={handleCreateVariant}
        />
      )}

      {editingVariant === undefined ? null : (
        <VariantEditForm
          selection={editingVariant}
          translate={props.translate}
          disabled={variantMutationPending}
          onCancel={() => setEditingVariantID('')}
          onSubmit={handleUpdateVariant}
        />
      )}

      <InventoryList
        products={products}
        translate={props.translate}
        canMutate={canMutate}
        productMutationPending={productMutationPending}
        variantMutationPending={variantMutationPending}
        onEdit={(product) => {
          setFormStatus('')
          setAddingVariantProductID('')
          setEditingVariantID('')
          setEditingProductID(product.id)
        }}
        onDelete={handleDeleteProduct}
        onAddVariant={(product) => {
          setFormStatus('')
          setEditingProductID('')
          setEditingVariantID('')
          setAddingVariantProductID(product.id)
        }}
        onEditVariant={(selection) => {
          setFormStatus('')
          setAddingVariantProductID('')
          setEditingProductID('')
          setEditingVariantID(selection.variant.id)
        }}
        onDeleteVariant={handleDeleteVariant}
      />
    </section>
  )
}

function ColourVariantCreateFields(props: {
  fieldID: string
  index: number
  form: UseFormReturn<InventoryProductFormValues>
  category: InventoryCategory
  translate: Translate
  photoState: PhotoFormState
  previewURL: string
  canRemove: boolean
  onPhotoChange: (files: FileList | null) => void
  onRemove: () => void
}) {
  const stockFields = useFieldArray({
    control: props.form.control,
    name: `variants.${props.index}.sizes` as const
  })
  const sized = props.category === 'shirt' || props.category === 'hoodie'

  return (
    <div className="grid gap-ui-16 rounded-md border border-border p-ui-16">
      <div className="grid gap-ui-12 min-[900px]:grid-cols-[1fr_1fr_1fr_auto]">
        <div className="grid gap-ui-8">
          <Label htmlFor={`inventory-variant-colour-${props.fieldID}`}>
            {props.translate('inventory.colourLabel')}
          </Label>
          <Input
            id={`inventory-variant-colour-${props.fieldID}`}
            {...props.form.register(`variants.${props.index}.colour`, { required: true })}
          />
        </div>
        <div className="grid gap-ui-8">
          <Label htmlFor={`inventory-variant-price-${props.fieldID}`}>
            {props.translate('inventory.priceLabel')}
          </Label>
          <Input
            id={`inventory-variant-price-${props.fieldID}`}
            type="number"
            step="0.01"
            min="0"
            {...props.form.register(`variants.${props.index}.priceAmount`, { valueAsNumber: true })}
          />
        </div>
        <div className="grid gap-ui-8">
          <Label htmlFor={`inventory-variant-cost-${props.fieldID}`}>
            {props.translate('inventory.costLabel')}
          </Label>
          <Input
            id={`inventory-variant-cost-${props.fieldID}`}
            type="number"
            step="0.01"
            min="0"
            {...props.form.register(`variants.${props.index}.costAmount`, { valueAsNumber: true })}
          />
        </div>
        <div className="flex items-end">
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label={props.translate('inventory.removeVariant')}
            disabled={!props.canRemove}
            onClick={props.onRemove}
          >
            <Trash2 aria-hidden="true" />
          </Button>
        </div>
      </div>

      <div className="grid gap-ui-8">
        <Label htmlFor={`inventory-variant-photo-${props.fieldID}`}>
          {props.translate('inventory.photoLabel')}
        </Label>
        <div className="grid gap-ui-12 min-[800px]:grid-cols-[minmax(0,1fr)_160px]">
          <Input
            id={`inventory-variant-photo-${props.fieldID}`}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            onChange={(event) => props.onPhotoChange(event.currentTarget.files)}
          />
          {props.previewURL === '' ? null : (
            <img
              src={props.previewURL}
              alt={props.translate('inventory.photoPreviewAlt')}
              className="h-[120px] w-[160px] rounded-md border border-border bg-muted object-cover"
            />
          )}
        </div>
        <PhotoStatus photoState={props.photoState} translate={props.translate} />
      </div>

      <div className="grid gap-ui-12">
        <div className="flex items-center justify-between gap-ui-12">
          <h4 className="m-0 text-sm font-semibold">{props.translate('inventory.sizesTitle')}</h4>
          {sized ? (
            <Button
              type="button"
              variant="outline"
              onClick={() => stockFields.append(createInitialSizeStockValues(props.category))}
            >
              <Plus aria-hidden="true" />
              {props.translate('inventory.addSize')}
            </Button>
          ) : null}
        </div>
        {stockFields.fields.map((stockField, sizeIndex) => (
          <div key={stockField.id} className="flex flex-wrap items-end gap-ui-12">
            {sized ? (
              <div className="grid min-w-32 gap-ui-8">
                <Label htmlFor={`inventory-variant-size-${stockField.id}`}>
                  {props.translate('inventory.sizeLabel')}
                </Label>
                <select
                  id={`inventory-variant-size-${stockField.id}`}
                  className="h-9 rounded-md border border-input bg-background px-ui-12 text-sm"
                  {...props.form.register(`variants.${props.index}.sizes.${sizeIndex}.size`, {
                    required: true
                  })}
                >
                  {clothingSizes.map((size) => (
                    <option key={size} value={size}>
                      {props.translate(sizeLabelKey(size))}
                    </option>
                  ))}
                </select>
              </div>
            ) : null}
            <div className="grid min-w-32 gap-ui-8">
              <Label htmlFor={`inventory-variant-quantity-${stockField.id}`}>
                {props.translate('inventory.quantityLabel')}
              </Label>
              <Input
                id={`inventory-variant-quantity-${stockField.id}`}
                type="number"
                step="1"
                min="0"
                {...props.form.register(`variants.${props.index}.sizes.${sizeIndex}.quantity`, {
                  valueAsNumber: true
                })}
              />
            </div>
            {sized ? (
              <Button
                type="button"
                variant="outline"
                size="icon"
                aria-label={props.translate('inventory.removeSize')}
                disabled={stockFields.fields.length === 1}
                onClick={() => stockFields.remove(sizeIndex)}
              >
                <Trash2 aria-hidden="true" />
              </Button>
            ) : null}
          </div>
        ))}
      </div>
    </div>
  )
}

function ProductEditForm(props: {
  product: InventoryProduct
  translate: Translate
  disabled: boolean
  onCancel: () => void
  onSubmit: (product: InventoryProduct, values: InventoryProductEditFormValues) => void
}) {
  const form = useForm<InventoryProductEditFormValues>({
    defaultValues: {
      name: props.product.name,
      category: props.product.category
    }
  })

  return (
    <Card aria-labelledby="inventory-edit-title">
      <CardHeader>
        <h3 id="inventory-edit-title" className="m-0 text-base leading-tight">
          {props.translate('inventory.editTitle')}
        </h3>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            void form.handleSubmit((values) => props.onSubmit(props.product, values))(event)
          }}
        >
          <div className="grid gap-ui-16 min-[800px]:grid-cols-2">
            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-edit-product-name-${props.product.id}`}>
                {props.translate('inventory.editNameLabel')}
              </Label>
              <Input
                id={`inventory-edit-product-name-${props.product.id}`}
                {...form.register('name', { required: true })}
              />
            </div>

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-edit-product-category-${props.product.id}`}>
                {props.translate('inventory.editCategoryLabel')}
              </Label>
              <select
                id={`inventory-edit-product-category-${props.product.id}`}
                className="h-9 w-full rounded-md border border-input bg-background px-ui-12 text-sm"
                {...form.register('category', { required: true })}
              >
                {inventoryCategories.map((category) => (
                  <option key={category} value={category}>
                    {props.translate(categoryLabelKey(category))}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-ui-16">
            <Button type="submit" disabled={props.disabled}>
              <Save aria-hidden="true" />
              {props.translate('inventory.updateSubmit')}
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={props.disabled}
              onClick={props.onCancel}
            >
              <X aria-hidden="true" />
              {props.translate('inventory.cancelEdit')}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

function VariantCreateForm(props: {
  product: InventoryProduct
  translate: Translate
  disabled: boolean
  onCancel: () => void
  onSubmit: (
    product: InventoryProduct,
    values: InventoryVariantFormValues,
    file: File | null
  ) => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const form = useForm<InventoryVariantFormValues>({
    defaultValues: createInitialVariantValues(props.product.category)
  })
  const selectedColour = form.watch('colour')
  const existingColour = props.product.variants.find(
    (variant) => normalizeColour(variant.colour) === normalizeColour(selectedColour)
  )

  return (
    <Card aria-labelledby="inventory-create-variant-title">
      <CardHeader>
        <h3 id="inventory-create-variant-title" className="m-0 text-base leading-tight">
          {props.translate('inventory.addVariant')}
        </h3>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            void form.handleSubmit((values) => props.onSubmit(props.product, values, file))(event)
          }}
        >
          <div className="grid gap-ui-12 min-[900px]:grid-cols-5">
            {props.product.category === 'shirt' || props.product.category === 'hoodie' ? (
              <div className="grid gap-ui-8">
                <Label htmlFor={`inventory-create-variant-size-${props.product.id}`}>
                  {props.translate('inventory.newVariantSizeLabel')}
                </Label>
                <select
                  id={`inventory-create-variant-size-${props.product.id}`}
                  className="h-9 w-full rounded-md border border-input bg-background px-ui-12 text-sm"
                  {...form.register('size', { required: true })}
                >
                  {clothingSizes.map((size) => (
                    <option key={size} value={size}>
                      {props.translate(sizeLabelKey(size))}
                    </option>
                  ))}
                </select>
              </div>
            ) : null}

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-create-variant-colour-${props.product.id}`}>
                {props.translate('inventory.newVariantColourLabel')}
              </Label>
              <Input
                id={`inventory-create-variant-colour-${props.product.id}`}
                {...form.register('colour')}
              />
            </div>

            {existingColour === undefined ? (
              <div className="grid gap-ui-8">
                <Label htmlFor={`inventory-create-variant-price-${props.product.id}`}>
                  {props.translate('inventory.newVariantPriceLabel')}
                </Label>
                <Input
                  id={`inventory-create-variant-price-${props.product.id}`}
                  type="number"
                  step="0.01"
                  min="0"
                  {...form.register('priceAmount', { valueAsNumber: true })}
                />
              </div>
            ) : null}

            {existingColour === undefined ? (
              <div className="grid gap-ui-8">
                <Label htmlFor={`inventory-create-variant-cost-${props.product.id}`}>
                  {props.translate('inventory.newVariantCostLabel')}
                </Label>
                <Input
                  id={`inventory-create-variant-cost-${props.product.id}`}
                  type="number"
                  step="0.01"
                  min="0"
                  {...form.register('costAmount', { valueAsNumber: true })}
                />
              </div>
            ) : null}

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-create-variant-quantity-${props.product.id}`}>
                {props.translate('inventory.newVariantQuantityLabel')}
              </Label>
              <Input
                id={`inventory-create-variant-quantity-${props.product.id}`}
                type="number"
                step="1"
                min="0"
                {...form.register('quantity', { valueAsNumber: true })}
              />
            </div>
          </div>

          {existingColour === undefined ? (
            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-create-variant-photo-${props.product.id}`}>
                {props.translate('inventory.photoLabel')}
              </Label>
              <Input
                id={`inventory-create-variant-photo-${props.product.id}`}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                onChange={(event) => setFile(event.currentTarget.files?.[0] ?? null)}
              />
            </div>
          ) : (
            <p className="m-0 text-sm text-white-300">
              {props.translate('inventory.existingColourSizeNote')}
            </p>
          )}

          <div className="flex flex-wrap items-center gap-ui-16">
            <Button type="submit" disabled={props.disabled}>
              <Plus aria-hidden="true" />
              {props.translate('inventory.createVariantSubmit')}
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={props.disabled}
              onClick={props.onCancel}
            >
              <X aria-hidden="true" />
              {props.translate('inventory.cancelEdit')}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

function VariantEditForm(props: {
  selection: InventoryVariantSelection
  translate: Translate
  disabled: boolean
  onCancel: () => void
  onSubmit: (
    selection: InventoryVariantSelection,
    values: InventoryVariantEditFormValues,
    file: File | null
  ) => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const form = useForm<InventoryVariantEditFormValues>({
    defaultValues: toVariantFormValues(props.selection.variant)
  })

  return (
    <Card aria-labelledby="inventory-edit-variant-title">
      <CardHeader>
        <h3 id="inventory-edit-variant-title" className="m-0 text-base leading-tight">
          {props.translate('inventory.editVariantTitle')}
        </h3>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-ui-16"
          onSubmit={(event) => {
            void form.handleSubmit((values) => props.onSubmit(props.selection, values, file))(event)
          }}
        >
          <div className="grid gap-ui-12 min-[900px]:grid-cols-5">
            {props.selection.product.category === 'shirt' ||
            props.selection.product.category === 'hoodie' ? (
              <div className="grid gap-ui-8">
                <Label htmlFor={`inventory-edit-variant-size-${props.selection.variant.id}`}>
                  {props.translate('inventory.editVariantSizeLabel')}
                </Label>
                <select
                  id={`inventory-edit-variant-size-${props.selection.variant.id}`}
                  className="h-9 w-full rounded-md border border-input bg-background px-ui-12 text-sm"
                  {...form.register('size', { required: true })}
                >
                  {clothingSizes.map((size) => (
                    <option key={size} value={size}>
                      {props.translate(sizeLabelKey(size))}
                    </option>
                  ))}
                </select>
              </div>
            ) : null}

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-edit-variant-colour-${props.selection.variant.id}`}>
                {props.translate('inventory.editVariantColourLabel')}
              </Label>
              <Input
                id={`inventory-edit-variant-colour-${props.selection.variant.id}`}
                {...form.register('colour')}
              />
            </div>

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-edit-variant-price-${props.selection.variant.id}`}>
                {props.translate('inventory.editVariantPriceLabel')}
              </Label>
              <Input
                id={`inventory-edit-variant-price-${props.selection.variant.id}`}
                type="number"
                step="0.01"
                min="0"
                {...form.register('priceAmount', { valueAsNumber: true })}
              />
            </div>

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-edit-variant-cost-${props.selection.variant.id}`}>
                {props.translate('inventory.editVariantCostLabel')}
              </Label>
              <Input
                id={`inventory-edit-variant-cost-${props.selection.variant.id}`}
                type="number"
                step="0.01"
                min="0"
                {...form.register('costAmount', { valueAsNumber: true })}
              />
            </div>

            <div className="grid gap-ui-8">
              <Label htmlFor={`inventory-edit-variant-quantity-${props.selection.variant.id}`}>
                {props.translate('inventory.editVariantQuantityLabel')}
              </Label>
              <Input
                id={`inventory-edit-variant-quantity-${props.selection.variant.id}`}
                type="number"
                step="1"
                min="0"
                {...form.register('quantity', { valueAsNumber: true })}
              />
            </div>
          </div>

          <div className="grid gap-ui-8">
            <Label htmlFor={`inventory-edit-variant-photo-${props.selection.variant.id}`}>
              {props.translate('inventory.photoLabel')}
            </Label>
            <Input
              id={`inventory-edit-variant-photo-${props.selection.variant.id}`}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onChange={(event) => setFile(event.currentTarget.files?.[0] ?? null)}
            />
          </div>

          <div className="flex flex-wrap items-center gap-ui-16">
            <Button type="submit" disabled={props.disabled}>
              <Save aria-hidden="true" />
              {props.translate('inventory.updateVariantSubmit')}
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={props.disabled}
              onClick={props.onCancel}
            >
              <X aria-hidden="true" />
              {props.translate('inventory.cancelEdit')}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

function PhotoStatus(props: { photoState: PhotoFormState; translate: Translate }) {
  if (props.photoState.status === 'empty') {
    return null
  }

  if (props.photoState.status === 'processing') {
    return (
      <p className="m-0 flex items-center gap-ui-8 text-sm text-white-300" role="status">
        <Upload aria-hidden="true" size={16} />
        {props.translate('inventory.photoProcessing')} {props.photoState.fileName}
      </p>
    )
  }

  if (props.photoState.status === 'failed') {
    return (
      <p className="m-0 text-sm text-red-100" role="status">
        {props.photoState.message}
      </p>
    )
  }

  return (
    <p className="m-0 text-sm text-white-200" role="status">
      {props.translate('inventory.photoReady')} {props.photoState.fileName}
    </p>
  )
}

function InventoryList(props: {
  products: InventoryProduct[]
  translate: Translate
  canMutate: boolean
  productMutationPending: boolean
  variantMutationPending: boolean
  onEdit: (product: InventoryProduct) => void
  onDelete: (product: InventoryProduct) => void
  onAddVariant: (product: InventoryProduct) => void
  onEditVariant: (selection: InventoryVariantSelection) => void
  onDeleteVariant: (selection: InventoryVariantSelection) => void
}) {
  if (props.products.length === 0) {
    return <StatusPanel message={props.translate('inventory.empty')} />
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{props.translate('inventory.productHeader')}</TableHead>
          <TableHead>{props.translate('inventory.categoryHeader')}</TableHead>
          <TableHead>{props.translate('inventory.variantsHeader')}</TableHead>
          <TableHead>{props.translate('inventory.stockHeader')}</TableHead>
          <TableHead>{props.translate('inventory.statusHeader')}</TableHead>
          {props.canMutate ? (
            <TableHead>{props.translate('inventory.actionsHeader')}</TableHead>
          ) : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {props.products.map((product) => (
          <TableRow key={product.id}>
            <TableCell>
              <div className="flex items-center gap-ui-12">
                <span className="font-medium text-white-100">{product.name}</span>
              </div>
            </TableCell>
            <TableCell>{props.translate(categoryLabelKey(product.category))}</TableCell>
            <TableCell>
              <div className="grid gap-ui-8">
                <span>
                  {variantCountLabel(
                    groupInventoryVariants(product.variants).length,
                    props.translate
                  )}
                </span>
                {groupInventoryVariants(product.variants).map((group) => (
                  <div
                    key={group.id}
                    className="grid gap-ui-8 rounded-md border border-border p-ui-8 text-sm"
                  >
                    <div className="flex items-center gap-ui-8">
                      <img
                        src={group.photo.display.publicUrl}
                        alt={`${product.name} ${group.colour}`}
                        className="h-12 w-16 rounded-md border border-border bg-muted object-cover"
                      />
                      <span className="font-medium">{group.colour}</span>
                    </div>
                    {group.sizes.map((variant) => {
                      const selection: InventoryVariantSelection = { product, variant }
                      const label = variantLabel(variant, props.translate)
                      return (
                        <div key={variant.id} className="flex flex-wrap items-center gap-ui-8">
                          {product.category === 'shirt' || product.category === 'hoodie' ? (
                            <span>{props.translate(sizeLabelKey(variant.size))}</span>
                          ) : null}
                          <span className="text-white-300">
                            {variantStockLabel(variant, props.translate)}
                          </span>
                          {props.canMutate ? (
                            <div className="flex items-center gap-ui-4">
                              <Button
                                type="button"
                                variant="outline"
                                size="icon"
                                aria-label={`${props.translate('inventory.editVariant')} ${product.name} ${label}`}
                                disabled={props.variantMutationPending}
                                onClick={() => props.onEditVariant(selection)}
                              >
                                <Pencil aria-hidden="true" />
                              </Button>
                              <Button
                                type="button"
                                variant="outline"
                                size="icon"
                                aria-label={`${props.translate('inventory.deleteVariant')} ${product.name} ${label}`}
                                title={
                                  product.variants.length === 1
                                    ? props.translate('inventory.lastVariantDeleteDisabled')
                                    : undefined
                                }
                                disabled={
                                  props.variantMutationPending || product.variants.length === 1
                                }
                                onClick={() => props.onDeleteVariant(selection)}
                              >
                                <Trash2 aria-hidden="true" />
                              </Button>
                            </div>
                          ) : null}
                        </div>
                      )
                    })}
                  </div>
                ))}
              </div>
            </TableCell>
            <TableCell>{totalStock(product)}</TableCell>
            <TableCell>
              <Badge variant={isSoldOut(product) ? 'secondary' : 'default'}>
                {props.translate(isSoldOut(product) ? 'inventory.soldOut' : 'inventory.inStock')}
              </Badge>
            </TableCell>
            {props.canMutate ? (
              <TableCell>
                <div className="flex items-center gap-ui-8">
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label={`${props.translate('inventory.addVariant')} ${product.name}`}
                    disabled={props.variantMutationPending}
                    onClick={() => props.onAddVariant(product)}
                  >
                    <Plus aria-hidden="true" />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label={`${props.translate('inventory.editProduct')} ${product.name}`}
                    disabled={props.productMutationPending}
                    onClick={() => props.onEdit(product)}
                  >
                    <Pencil aria-hidden="true" />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label={`${props.translate('inventory.deleteProduct')} ${product.name}`}
                    disabled={props.productMutationPending}
                    onClick={() => props.onDelete(product)}
                  >
                    <Trash2 aria-hidden="true" />
                  </Button>
                </div>
              </TableCell>
            ) : null}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function StatusPanel(props: { message: string }) {
  return (
    <div className="rounded-md border border-border p-ui-16">
      <p className="m-0 text-white-300" role="status">
        {props.message}
      </p>
    </div>
  )
}

function createInitialProductValues(): InventoryProductFormValues {
  return {
    name: '',
    category: 'shirt',
    variants: [createInitialColourValues('shirt')]
  }
}

function createInitialColourValues(category: InventoryCategory): InventoryColourFormValues {
  return {
    colour: '',
    priceAmount: 0,
    costAmount: 0,
    sizes: [createInitialSizeStockValues(category)]
  }
}

function createInitialSizeStockValues(category: InventoryCategory): InventorySizeStockFormValues {
  return {
    size: category === 'shirt' || category === 'hoodie' ? 'm' : 'not_applicable',
    quantity: 0
  }
}

function createInitialVariantValues(category: InventoryCategory): InventoryVariantFormValues {
  return {
    size: category === 'shirt' || category === 'hoodie' ? 'm' : 'not_applicable',
    colour: '',
    priceAmount: 0,
    costAmount: 0,
    quantity: 0
  }
}

function toPhotoUploadVariantRequest(
  variant: ProcessedInventoryPhoto['full']
): PhotoUploadVariantRequest {
  return {
    contentType: variant.contentType,
    sizeBytes: variant.sizeBytes,
    width: variant.width,
    height: variant.height
  }
}

function toPhotoManifest(photo: InventoryPhoto): InventoryPhotoManifest {
  return {
    full: {
      objectKey: photo.full.objectKey,
      contentType: photo.full.contentType,
      sizeBytes: photo.full.sizeBytes,
      width: photo.full.width,
      height: photo.full.height
    },
    display: {
      objectKey: photo.display.objectKey,
      contentType: photo.display.contentType,
      sizeBytes: photo.display.sizeBytes,
      width: photo.display.width,
      height: photo.display.height
    }
  }
}

function toVariantRequest(
  values: InventoryVariantFormValues,
  photo: InventoryPhotoManifest
): InventoryVariantRequest {
  return {
    size: values.size,
    colour: values.colour.trim(),
    photo,
    price: {
      amount: amountToCents(values.priceAmount),
      currency: 'BRL'
    },
    cost: {
      amount: amountToCents(values.costAmount),
      currency: 'BRL'
    },
    quantity: values.quantity
  }
}

async function uploadPhotoFile(accessToken: string, file: File): Promise<InventoryPhotoManifest> {
  const photo = await processInventoryPhoto(file)
  const uploadRequest = await createInventoryPhotoUploadRequest(accessToken, {
    full: toPhotoUploadVariantRequest(photo.full),
    display: toPhotoUploadVariantRequest(photo.display)
  })
  await Promise.all([
    uploadInventoryPhotoVariant(uploadRequest.uploads.full, photo.full.blob),
    uploadInventoryPhotoVariant(uploadRequest.uploads.display, photo.display.blob)
  ])
  return toPhotoManifest(uploadRequest.photo)
}

function normalizeColour(colour: string): string {
  return colour.trim().toLowerCase().replace(/\s+/g, ' ')
}

function toVariantFormValues(variant: InventoryVariant): InventoryVariantEditFormValues {
  return {
    size: variant.size,
    colour: variant.colour,
    priceAmount: variant.price.amount / 100,
    costAmount: variant.cost.amount / 100,
    quantity: variant.quantity
  }
}

function amountToCents(value: number): number {
  return Math.round(value * 100)
}

function inventoryMutationMessage(error: unknown, translate: Translate): string {
  if (error instanceof ApiError) {
    if (error.code === 'duplicate_product') return translate('inventory.duplicateProduct')
    if (error.code === 'duplicate_variant') return translate('inventory.duplicateVariant')
  }
  return error instanceof Error ? error.message : translate('inventory.error')
}

function totalStock(product: InventoryProduct): number {
  return product.variants.reduce((total, variant) => total + variant.quantity, 0)
}

function groupInventoryVariants(variants: InventoryVariant[]): InventoryColourGroup[] {
  const groups = new Map<string, InventoryColourGroup>()
  for (const variant of variants) {
    const existing = groups.get(variant.colourVariantId)
    if (existing === undefined) {
      groups.set(variant.colourVariantId, {
        id: variant.colourVariantId,
        colour: variant.colour,
        photo: variant.photo,
        sizes: [variant]
      })
    } else {
      existing.sizes.push(variant)
    }
  }
  return Array.from(groups.values())
}

function isSoldOut(product: InventoryProduct): boolean {
  return totalStock(product) === 0
}

function findProductByID(
  products: InventoryProduct[] | undefined,
  productID: string
): InventoryProduct | undefined {
  if (products === undefined || productID === '') {
    return undefined
  }

  return products.find((product) => product.id === productID)
}

function findInventoryVariant(
  products: InventoryProduct[] | undefined,
  variantID: string
): InventoryVariantSelection | undefined {
  if (products === undefined || variantID === '') {
    return undefined
  }

  for (const product of products) {
    const variant = product.variants.find((inventoryVariant) => inventoryVariant.id === variantID)
    if (variant !== undefined) {
      return { product, variant }
    }
  }

  return undefined
}

function variantLabel(variant: InventoryVariant, translate: Translate): string {
  return `${translate(sizeLabelKey(variant.size))} / ${variant.colour}`
}

function variantCountLabel(count: number, translate: Translate): string {
  return count === 1
    ? `1 ${translate('inventory.variantSingular')}`
    : `${count} ${translate('inventory.variantsPlural')}`
}

function variantStockLabel(variant: InventoryVariant, translate: Translate): string {
  return variant.quantity === 0
    ? translate('inventory.soldOut')
    : `${variant.quantity} ${translate('inventory.inStockCountSuffix')}`
}

function categoryLabelKey(category: InventoryCategory): TranslationKey {
  return `inventory.category.${category}`
}

function sizeLabelKey(size: InventorySize): TranslationKey {
  return `inventory.size.${size}`
}

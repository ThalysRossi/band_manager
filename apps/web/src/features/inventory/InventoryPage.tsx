import { useEffect, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Save, Trash2, Upload, X } from 'lucide-react'
import type { TranslationKey } from 'i18n'
import { z } from 'zod'

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
  deleteInventoryProduct,
  listInventory,
  updateInventoryProduct,
  uploadInventoryPhotoVariant
} from './api'
import type {
  CreateInventoryProductRequest,
  InventoryCategory,
  InventoryPhoto,
  InventoryPhotoManifest,
  InventoryPhotoUploadResponse,
  InventoryProduct,
  InventorySize,
  InventoryVariantRequest,
  PhotoUploadVariantRequest,
  UpdateInventoryProductRequest
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
  variants: InventoryVariantFormValues[]
}

type InventoryProductEditFormValues = {
  name: string
  category: InventoryCategory
}

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

const inventorySizes: InventorySize[] = [
  'not_applicable',
  'one_size',
  'pp',
  'p',
  'm',
  'g',
  'gg',
  'xgg'
]

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
        size: z.enum(['not_applicable', 'one_size', 'pp', 'p', 'm', 'g', 'gg', 'xgg']),
        colour: z.string(),
        priceAmount: z.number().finite().min(0),
        costAmount: z.number().finite().min(0),
        quantity: z.number().int().min(0)
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

type UpdateProductMutationInput = {
  productID: string
  request: UpdateInventoryProductRequest
}

export function InventoryPage(props: InventoryPageProps) {
  const queryClient = useQueryClient()
  const [formStatus, setFormStatus] = useState<string>('')
  const [photoState, setPhotoState] = useState<PhotoFormState>({ status: 'empty' })
  const [previewURL, setPreviewURL] = useState<string>('')
  const [editingProductID, setEditingProductID] = useState<string>('')

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

  const createMutation = useMutation({
    mutationFn: (request: CreateInventoryProductRequest) =>
      createInventoryProduct(props.accessToken, request),
    onSuccess: () => {
      form.reset(createInitialProductValues())
      setPhotoState({ status: 'empty' })
      setPreviewURL('')
      setFormStatus(props.translate('inventory.createSuccess'))
      void queryClient.invalidateQueries({ queryKey: ['inventory', props.accessToken] })
    },
    onError: (error) => {
      setFormStatus(error instanceof Error ? error.message : props.translate('inventory.error'))
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
      setFormStatus(error instanceof Error ? error.message : props.translate('inventory.error'))
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
      setFormStatus(error instanceof Error ? error.message : props.translate('inventory.error'))
    }
  })

  useEffect(() => {
    return () => {
      if (previewURL !== '') {
        URL.revokeObjectURL(previewURL)
      }
    }
  }, [previewURL])

  const account = accountQuery.data
  const products = inventoryQuery.data
  const canMutate = account?.activeBand.canWrite === true
  const editingProduct = findProductByID(products, editingProductID)
  const createPending = createMutation.isPending || photoState.status === 'processing'
  const productMutationPending = updateProductMutation.isPending || deleteProductMutation.isPending

  async function handlePhotoChange(fileList: FileList | null): Promise<void> {
    const file = fileList?.[0]
    if (file === null || file === undefined) {
      setPhotoState({ status: 'empty' })
      return
    }

    setFormStatus('')
    setPhotoState({ status: 'processing', fileName: file.name })
    try {
      const photo = await processInventoryPhoto(file)
      setPhotoState({ status: 'ready', fileName: file.name, photo })
      setPreviewURL(URL.createObjectURL(photo.display.blob))
    } catch {
      setPhotoState({
        status: 'failed',
        message: props.translate('inventory.photoInvalid')
      })
    }
  }

  async function handleCreate(values: InventoryProductFormValues): Promise<void> {
    setFormStatus('')
    const parsedValues = inventoryProductSchema.safeParse(values)
    if (!parsedValues.success) {
      setFormStatus(props.translate('inventory.formInvalid'))
      return
    }

    if (photoState.status !== 'ready') {
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
        uploadInventoryPhotoVariant(uploadRequest.uploads.display, photoState.photo.display.blob)
      ])
    } catch {
      setFormStatus(props.translate('inventory.photoUploadFailed'))
      return
    }

    createMutation.mutate({
      name: parsedValues.data.name.trim(),
      category: parsedValues.data.category,
      photo: toPhotoManifest(uploadRequest.photo),
      variants: parsedValues.data.variants.map(toVariantRequest)
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
          <h2 className="m-0 text-[1.75rem] leading-[1.15]">
            {props.translate('nav.inventory')}
          </h2>
          <p className="m-0 text-base text-white-300">
            {props.translate('inventory.productCount')}: {products.length}
          </p>
        </div>
      </header>

      {canMutate ? (
        <Card aria-labelledby="inventory-create-title">
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

              <div className="grid gap-ui-12">
                <Label htmlFor="inventory-product-photo">
                  {props.translate('inventory.photoLabel')}
                </Label>
                <div className="grid gap-ui-12 min-[800px]:grid-cols-[minmax(0,1fr)_160px]">
                  <Input
                    id="inventory-product-photo"
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    onChange={(event) => {
                      void handlePhotoChange(event.currentTarget.files)
                    }}
                  />
                  {previewURL === '' ? null : (
                    <img
                      src={previewURL}
                      alt={props.translate('inventory.photoPreviewAlt')}
                      className="h-[120px] w-[160px] rounded-md border border-border object-cover"
                    />
                  )}
                </div>
                <PhotoStatus photoState={photoState} translate={props.translate} />
              </div>

              <div className="grid gap-ui-16">
                <div className="flex items-center justify-between gap-ui-16">
                  <h3 className="m-0 text-base leading-tight">
                    {props.translate('inventory.variantsTitle')}
                  </h3>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => variantFields.append(createInitialVariantValues())}
                  >
                    <Plus aria-hidden="true" />
                    {props.translate('inventory.addVariant')}
                  </Button>
                </div>

                <div className="grid gap-ui-16">
                  {variantFields.fields.map((field, index) => (
                    <div
                      key={field.id}
                      className="grid gap-ui-12 rounded-md border border-border p-ui-12 min-[900px]:grid-cols-[1fr_1fr_1fr_1fr_1fr_auto]"
                    >
                      <div className="grid gap-ui-8">
                        <Label htmlFor={`inventory-variant-size-${field.id}`}>
                          {props.translate('inventory.sizeLabel')}
                        </Label>
                        <select
                          id={`inventory-variant-size-${field.id}`}
                          className="h-9 w-full rounded-md border border-input bg-background px-ui-12 text-sm"
                          {...form.register(`variants.${index}.size`, { required: true })}
                        >
                          {inventorySizes.map((size) => (
                            <option key={size} value={size}>
                              {props.translate(sizeLabelKey(size))}
                            </option>
                          ))}
                        </select>
                      </div>

                      <div className="grid gap-ui-8">
                        <Label htmlFor={`inventory-variant-colour-${field.id}`}>
                          {props.translate('inventory.colourLabel')}
                        </Label>
                        <Input
                          id={`inventory-variant-colour-${field.id}`}
                          {...form.register(`variants.${index}.colour`)}
                        />
                      </div>

                      <div className="grid gap-ui-8">
                        <Label htmlFor={`inventory-variant-price-${field.id}`}>
                          {props.translate('inventory.priceLabel')}
                        </Label>
                        <Input
                          id={`inventory-variant-price-${field.id}`}
                          type="number"
                          step="0.01"
                          min="0"
                          {...form.register(`variants.${index}.priceAmount`, {
                            valueAsNumber: true
                          })}
                        />
                      </div>

                      <div className="grid gap-ui-8">
                        <Label htmlFor={`inventory-variant-cost-${field.id}`}>
                          {props.translate('inventory.costLabel')}
                        </Label>
                        <Input
                          id={`inventory-variant-cost-${field.id}`}
                          type="number"
                          step="0.01"
                          min="0"
                          {...form.register(`variants.${index}.costAmount`, {
                            valueAsNumber: true
                          })}
                        />
                      </div>

                      <div className="grid gap-ui-8">
                        <Label htmlFor={`inventory-variant-quantity-${field.id}`}>
                          {props.translate('inventory.quantityLabel')}
                        </Label>
                        <Input
                          id={`inventory-variant-quantity-${field.id}`}
                          type="number"
                          step="1"
                          min="0"
                          {...form.register(`variants.${index}.quantity`, {
                            valueAsNumber: true
                          })}
                        />
                      </div>

                      <div className="flex items-end">
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          aria-label={props.translate('inventory.removeVariant')}
                          disabled={variantFields.fields.length === 1}
                          onClick={() => variantFields.remove(index)}
                        >
                          <Trash2 aria-hidden="true" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-ui-16">
                <Button type="submit" disabled={createPending}>
                  <Save aria-hidden="true" />
                  {props.translate('inventory.createSubmit')}
                </Button>
                {formStatus === '' ? null : (
                  <p className="m-0 text-sm text-white-300" role="status">
                    {formStatus}
                  </p>
                )}
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

      <InventoryList
        products={products}
        translate={props.translate}
        canMutate={canMutate}
        mutationPending={productMutationPending}
        onEdit={(product) => {
          setFormStatus('')
          setEditingProductID(product.id)
        }}
        onDelete={handleDeleteProduct}
      />
    </section>
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
            <Button type="button" variant="outline" disabled={props.disabled} onClick={props.onCancel}>
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
    <p className="m-0 text-sm text-green-100" role="status">
      {props.translate('inventory.photoReady')} {props.photoState.fileName}
    </p>
  )
}

function InventoryList(props: {
  products: InventoryProduct[]
  translate: Translate
  canMutate: boolean
  mutationPending: boolean
  onEdit: (product: InventoryProduct) => void
  onDelete: (product: InventoryProduct) => void
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
          {props.canMutate ? <TableHead>{props.translate('inventory.actionsHeader')}</TableHead> : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {props.products.map((product) => (
          <TableRow key={product.id}>
            <TableCell>
              <div className="flex items-center gap-ui-12">
                <img
                  src={product.photo.display.publicUrl}
                  alt={product.name}
                  className="h-12 w-16 rounded-md border border-border object-cover"
                />
                <span className="font-medium text-white-100">{product.name}</span>
              </div>
            </TableCell>
            <TableCell>{props.translate(categoryLabelKey(product.category))}</TableCell>
            <TableCell>{product.variants.length}</TableCell>
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
                    aria-label={`${props.translate('inventory.editProduct')} ${product.name}`}
                    disabled={props.mutationPending}
                    onClick={() => props.onEdit(product)}
                  >
                    <Pencil aria-hidden="true" />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label={`${props.translate('inventory.deleteProduct')} ${product.name}`}
                    disabled={props.mutationPending}
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
    variants: [createInitialVariantValues()]
  }
}

function createInitialVariantValues(): InventoryVariantFormValues {
  return {
    size: 'm',
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

function toVariantRequest(values: InventoryVariantFormValues): InventoryVariantRequest {
  return {
    size: values.size,
    colour: values.colour.trim(),
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

function amountToCents(value: number): number {
  return Math.round(value * 100)
}

function totalStock(product: InventoryProduct): number {
  return product.variants.reduce((total, variant) => total + variant.quantity, 0)
}

function isSoldOut(product: InventoryProduct): boolean {
  return totalStock(product) === 0
}

function findProductByID(products: InventoryProduct[] | undefined, productID: string): InventoryProduct | undefined {
  if (products === undefined || productID === '') {
    return undefined
  }

  return products.find((product) => product.id === productID)
}

function categoryLabelKey(category: InventoryCategory): TranslationKey {
  return `inventory.category.${category}`
}

function sizeLabelKey(size: InventorySize): TranslationKey {
  return `inventory.size.${size}`
}

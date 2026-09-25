export type Locale = 'en' | 'pt-BR'

export type TranslationKey =
  | 'app.title'
  | 'auth.bandNameLabel'
  | 'auth.emailInstruction'
  | 'auth.emailLabel'
  | 'auth.loginSubmit'
  | 'auth.loginTitle'
  | 'auth.onboardingSubmit'
  | 'auth.onboardingTitle'
  | 'auth.passwordReset'
  | 'auth.passwordResetSent'
  | 'auth.passwordResetSubmit'
  | 'auth.passwordLabel'
  | 'auth.passwordUpdateSubmit'
  | 'auth.passwordUpdateFailed'
  | 'auth.passwordUpdateTitle'
  | 'auth.passwordUpdated'
  | 'auth.signupSubmit'
  | 'auth.signupTitle'
  | 'auth.timezoneLabel'
  | 'auth.emailVerificationRequired'
  | 'auth.genericError'
  | 'auth.signupCreated'
  | 'auth.loginReady'
  | 'auth.logout'
  | 'auth.logoutFailed'
  | 'auth.recoveryLinkError'
  | 'account.acceptLoading'
  | 'account.acceptLoginPrompt'
  | 'account.acceptMissingToken'
  | 'account.acceptSuccess'
  | 'account.acceptTitle'
  | 'account.actionsHeader'
  | 'account.copyFailed'
  | 'account.copyInviteLink'
  | 'account.copySuccess'
  | 'account.createInviteSubmit'
  | 'account.createInviteTitle'
  | 'account.emailHeader'
  | 'account.emailLabel'
  | 'account.expiresAtHeader'
  | 'account.genericError'
  | 'account.inviteCreated'
  | 'account.inviteEmailInvalid'
  | 'account.invitesTitle'
  | 'account.inviteStatus.accepted'
  | 'account.inviteStatus.expired'
  | 'account.inviteStatus.pending'
  | 'account.inviteStatus.revoked'
  | 'account.joinedAtHeader'
  | 'account.loading'
  | 'account.loginRequired'
  | 'account.membersTitle'
  | 'account.noInvites'
  | 'account.noMembers'
  | 'account.revokeInvite'
  | 'account.role.admin'
  | 'account.role.member'
  | 'account.role.owner'
  | 'account.role.viewer'
  | 'account.roleHeader'
  | 'account.statusHeader'
  | 'account.title'
  | 'inventory.addVariant'
  | 'inventory.addProduct'
  | 'inventory.addSize'
  | 'inventory.addVariantSuccess'
  | 'inventory.actionsHeader'
  | 'inventory.cancelEdit'
  | 'inventory.category.accessory'
  | 'inventory.category.cassette'
  | 'inventory.category.cd'
  | 'inventory.category.hoodie'
  | 'inventory.category.patch'
  | 'inventory.category.shirt'
  | 'inventory.category.sticker'
  | 'inventory.category.tote_bag'
  | 'inventory.category.vinyl'
  | 'inventory.categoryHeader'
  | 'inventory.categoryLabel'
  | 'inventory.colourLabel'
  | 'inventory.costLabel'
  | 'inventory.createSubmit'
  | 'inventory.createSuccess'
  | 'inventory.createTitle'
  | 'inventory.createVariantSubmit'
  | 'inventory.deleteConfirm'
  | 'inventory.deleteProduct'
  | 'inventory.deleteSuccess'
  | 'inventory.deleteVariant'
  | 'inventory.deleteVariantConfirm'
  | 'inventory.deleteVariantSuccess'
  | 'inventory.duplicateProduct'
  | 'inventory.duplicateVariant'
  | 'inventory.editCategoryLabel'
  | 'inventory.editNameLabel'
  | 'inventory.editProduct'
  | 'inventory.editTitle'
  | 'inventory.editVariant'
  | 'inventory.editVariantColourLabel'
  | 'inventory.editVariantCostLabel'
  | 'inventory.editVariantPriceLabel'
  | 'inventory.editVariantQuantityLabel'
  | 'inventory.editVariantSizeLabel'
  | 'inventory.editVariantTitle'
  | 'inventory.existingColourSizeNote'
  | 'inventory.empty'
  | 'inventory.error'
  | 'inventory.formInvalid'
  | 'inventory.inStock'
  | 'inventory.inStockCountSuffix'
  | 'inventory.loading'
  | 'inventory.lastVariantDeleteDisabled'
  | 'inventory.nameLabel'
  | 'inventory.newVariantColourLabel'
  | 'inventory.newVariantCostLabel'
  | 'inventory.newVariantPriceLabel'
  | 'inventory.newVariantQuantityLabel'
  | 'inventory.newVariantSizeLabel'
  | 'inventory.photoInvalid'
  | 'inventory.photoLabel'
  | 'inventory.photoPreviewAlt'
  | 'inventory.photoProcessing'
  | 'inventory.photoReady'
  | 'inventory.photoRequired'
  | 'inventory.photoUploadFailed'
  | 'inventory.photoUploadRequestFailed'
  | 'inventory.priceLabel'
  | 'inventory.productCount'
  | 'inventory.productHeader'
  | 'inventory.quantityLabel'
  | 'inventory.removeVariant'
  | 'inventory.removeSize'
  | 'inventory.size.g'
  | 'inventory.size.gg'
  | 'inventory.size.m'
  | 'inventory.size.not_applicable'
  | 'inventory.size.one_size'
  | 'inventory.size.p'
  | 'inventory.size.pp'
  | 'inventory.size.xgg'
  | 'inventory.sizeLabel'
  | 'inventory.sizesTitle'
  | 'inventory.soldOut'
  | 'inventory.statusHeader'
  | 'inventory.stockHeader'
  | 'inventory.updateSubmit'
  | 'inventory.updateSuccess'
  | 'inventory.updateVariantSubmit'
  | 'inventory.updateVariantSuccess'
  | 'inventory.variantSingular'
  | 'inventory.variantsPlural'
  | 'inventory.variantsHeader'
  | 'inventory.variantsTitle'
  | 'merchBooth.addToCart'
  | 'merchBooth.backToBooth'
  | 'merchBooth.cartEmpty'
  | 'merchBooth.cartPersistenceError'
  | 'merchBooth.cartReadOnly'
  | 'merchBooth.cartReconciled'
  | 'merchBooth.cartTitle'
  | 'merchBooth.cashCheckout'
  | 'merchBooth.cashCheckoutSuccess'
  | 'merchBooth.checkoutError'
  | 'merchBooth.decreaseQuantity'
  | 'merchBooth.empty'
  | 'merchBooth.error'
  | 'merchBooth.increaseQuantity'
  | 'merchBooth.itemCount'
  | 'merchBooth.loading'
  | 'merchBooth.openCart'
  | 'merchBooth.orderSummary'
  | 'merchBooth.photoAlt'
  | 'merchBooth.selectSize'
  | 'merchBooth.removeFromCart'
  | 'merchBooth.total'
  | 'nav.inventory'
  | 'nav.merchBooth'
  | 'nav.reports'
  | 'nav.calendar'
  | 'nav.account'
  | 'status.backendReady'

export type TranslationDictionary = Record<TranslationKey, string>

export const translations: Record<Locale, TranslationDictionary> = {
  en: {
    'app.title': 'Band Manager',
    'auth.bandNameLabel': 'Band name',
    'auth.emailInstruction': 'Verify your email before setting up the band workspace.',
    'auth.emailLabel': 'Email',
    'auth.loginSubmit': 'Log in',
    'auth.loginTitle': 'Log in',
    'auth.onboardingSubmit': 'Create band workspace',
    'auth.onboardingTitle': 'Set up your band',
    'auth.passwordLabel': 'Password',
    'auth.passwordReset': 'Reset password',
    'auth.passwordResetSent': 'Check your email for the password reset link.',
    'auth.passwordResetSubmit': 'Send reset link',
    'auth.passwordUpdateSubmit': 'Update password',
    'auth.passwordUpdateFailed':
      'Password update failed. Request a new recovery link and try again.',
    'auth.passwordUpdateTitle': 'Choose a new password',
    'auth.passwordUpdated': 'Password updated.',
    'auth.signupSubmit': 'Create account',
    'auth.signupTitle': 'Create account',
    'auth.timezoneLabel': 'Band timezone',
    'auth.emailVerificationRequired':
      'Check your email, verify the account, then log in to finish setup.',
    'auth.genericError': 'Authentication failed. Check the fields and try again.',
    'auth.signupCreated': 'Account created.',
    'auth.loginReady': 'Login successful.',
    'auth.logout': 'Log out',
    'auth.logoutFailed': 'Logout failed. Try again.',
    'auth.recoveryLinkError': 'This recovery link is invalid or expired. Request a new one.',
    'account.acceptLoading': 'Accepting invite...',
    'account.acceptLoginPrompt': 'Log in with the invited email to accept this invite.',
    'account.acceptMissingToken': 'Invite token is missing.',
    'account.acceptSuccess': 'Invite accepted for',
    'account.acceptTitle': 'Accept invite',
    'account.actionsHeader': 'Actions',
    'account.copyFailed': 'Invite link could not be copied.',
    'account.copyInviteLink': 'Copy invite link',
    'account.copySuccess': 'Invite link copied.',
    'account.createInviteSubmit': 'Create invite',
    'account.createInviteTitle': 'Invite viewer',
    'account.emailHeader': 'Email',
    'account.emailLabel': 'Viewer email',
    'account.expiresAtHeader': 'Expires',
    'account.genericError': 'Account request failed.',
    'account.inviteCreated': 'Invite created.',
    'account.inviteEmailInvalid': 'Enter a valid email address.',
    'account.invitesTitle': 'Invites',
    'account.inviteStatus.accepted': 'Accepted',
    'account.inviteStatus.expired': 'Expired',
    'account.inviteStatus.pending': 'Pending',
    'account.inviteStatus.revoked': 'Revoked',
    'account.joinedAtHeader': 'Joined',
    'account.loading': 'Loading account...',
    'account.loginRequired': 'Log in to manage account access.',
    'account.membersTitle': 'Members',
    'account.noInvites': 'No invites yet.',
    'account.noMembers': 'No members yet.',
    'account.revokeInvite': 'Revoke',
    'account.role.admin': 'Admin',
    'account.role.member': 'Member',
    'account.role.owner': 'Owner',
    'account.role.viewer': 'Viewer',
    'account.roleHeader': 'Role',
    'account.statusHeader': 'Status',
    'account.title': 'Account',
    'inventory.addVariant': 'Add variant',
    'inventory.addProduct': 'Add product',
    'inventory.addSize': 'Add size',
    'inventory.addVariantSuccess': 'Variant created.',
    'inventory.actionsHeader': 'Actions',
    'inventory.cancelEdit': 'Cancel',
    'inventory.category.accessory': 'Accessory',
    'inventory.category.cassette': 'Cassette',
    'inventory.category.cd': 'CD',
    'inventory.category.hoodie': 'Hoodie',
    'inventory.category.patch': 'Patch',
    'inventory.category.shirt': 'Shirt',
    'inventory.category.sticker': 'Sticker',
    'inventory.category.tote_bag': 'Tote bag',
    'inventory.category.vinyl': 'Vinyl',
    'inventory.categoryHeader': 'Category',
    'inventory.categoryLabel': 'Category',
    'inventory.colourLabel': 'Colour',
    'inventory.costLabel': 'Cost (BRL)',
    'inventory.createSubmit': 'Create product',
    'inventory.createSuccess': 'Product created.',
    'inventory.createTitle': 'Create product',
    'inventory.createVariantSubmit': 'Create variant',
    'inventory.deleteConfirm': 'Delete this product from inventory?',
    'inventory.deleteProduct': 'Delete product',
    'inventory.deleteSuccess': 'Product deleted.',
    'inventory.deleteVariant': 'Delete variant',
    'inventory.deleteVariantConfirm': 'Delete this variant from inventory?',
    'inventory.deleteVariantSuccess': 'Variant deleted.',
    'inventory.duplicateProduct': 'A product with this category and name already exists.',
    'inventory.duplicateVariant':
      'This colour and size already exist, or the colour has different photo, price, or cost.',
    'inventory.editCategoryLabel': 'Edit category',
    'inventory.editNameLabel': 'Edit product name',
    'inventory.editProduct': 'Edit product',
    'inventory.editTitle': 'Edit product',
    'inventory.editVariant': 'Edit variant',
    'inventory.editVariantColourLabel': 'Edit variant colour',
    'inventory.editVariantCostLabel': 'Edit variant cost (BRL)',
    'inventory.editVariantPriceLabel': 'Edit variant price (BRL)',
    'inventory.editVariantQuantityLabel': 'Edit variant quantity',
    'inventory.editVariantSizeLabel': 'Edit variant size',
    'inventory.editVariantTitle': 'Edit variant',
    'inventory.existingColourSizeNote':
      'This size uses the colour’s existing photo, price, and cost.',
    'inventory.empty': 'No inventory products yet.',
    'inventory.error': 'Inventory request failed.',
    'inventory.formInvalid': 'Check the inventory form and try again.',
    'inventory.inStock': 'In stock',
    'inventory.inStockCountSuffix': 'in stock',
    'inventory.loading': 'Loading inventory...',
    'inventory.lastVariantDeleteDisabled': 'Delete the product to remove its final variant.',
    'inventory.nameLabel': 'Product name',
    'inventory.newVariantColourLabel': 'New variant colour',
    'inventory.newVariantCostLabel': 'New variant cost (BRL)',
    'inventory.newVariantPriceLabel': 'New variant price (BRL)',
    'inventory.newVariantQuantityLabel': 'New variant quantity',
    'inventory.newVariantSizeLabel': 'New variant size',
    'inventory.photoInvalid': 'Photo could not be processed.',
    'inventory.photoLabel': 'Photo',
    'inventory.photoPreviewAlt': 'Product photo preview',
    'inventory.photoProcessing': 'Processing photo',
    'inventory.photoReady': 'Photo ready',
    'inventory.photoRequired': 'Photo is required.',
    'inventory.photoUploadFailed': 'Photo upload failed.',
    'inventory.photoUploadRequestFailed':
      'Photo upload request failed. Restart the API and check VITE_API_BASE_URL.',
    'inventory.priceLabel': 'Price (BRL)',
    'inventory.productCount': 'Products',
    'inventory.productHeader': 'Product',
    'inventory.quantityLabel': 'Quantity',
    'inventory.removeVariant': 'Remove variant',
    'inventory.removeSize': 'Remove size',
    'inventory.size.g': 'G',
    'inventory.size.gg': 'GG',
    'inventory.size.m': 'M',
    'inventory.size.not_applicable': 'Not applicable',
    'inventory.size.one_size': 'One size',
    'inventory.size.p': 'P',
    'inventory.size.pp': 'PP',
    'inventory.size.xgg': 'XGG',
    'inventory.sizeLabel': 'Size',
    'inventory.sizesTitle': 'Sizes and stock',
    'inventory.soldOut': 'Sold out',
    'inventory.statusHeader': 'Status',
    'inventory.stockHeader': 'Stock',
    'inventory.updateSubmit': 'Save product',
    'inventory.updateSuccess': 'Product updated.',
    'inventory.updateVariantSubmit': 'Save variant',
    'inventory.updateVariantSuccess': 'Variant updated.',
    'inventory.variantSingular': 'variant',
    'inventory.variantsPlural': 'variants',
    'inventory.variantsHeader': 'Variants',
    'inventory.variantsTitle': 'Variants',
    'merchBooth.addToCart': 'Add to cart',
    'merchBooth.backToBooth': 'Back to merch booth',
    'merchBooth.cartEmpty': 'Your cart is empty.',
    'merchBooth.cartPersistenceError':
      'The cart could not be restored or saved in this browser session.',
    'merchBooth.cartReadOnly': 'Your role does not allow merch booth checkout.',
    'merchBooth.cartReconciled': 'The cart was updated to match current stock.',
    'merchBooth.cartTitle': 'Cart',
    'merchBooth.cashCheckout': 'Complete cash sale',
    'merchBooth.cashCheckoutSuccess': 'Cash sale completed.',
    'merchBooth.checkoutError': 'Cash checkout failed. Review stock and try again.',
    'merchBooth.decreaseQuantity': 'Decrease quantity for',
    'merchBooth.empty': 'No inventory variants are available for sale.',
    'merchBooth.error': 'Merch booth request failed.',
    'merchBooth.increaseQuantity': 'Increase quantity for',
    'merchBooth.itemCount': 'Items',
    'merchBooth.loading': 'Loading merch booth...',
    'merchBooth.openCart': 'Open cart',
    'merchBooth.orderSummary': 'Order summary',
    'merchBooth.photoAlt': 'product photo',
    'merchBooth.selectSize': 'Select size',
    'merchBooth.removeFromCart': 'Remove from cart',
    'merchBooth.total': 'Total',
    'nav.inventory': 'Inventory',
    'nav.merchBooth': 'Merch Booth',
    'nav.reports': 'Reports',
    'nav.calendar': 'Calendar',
    'nav.account': 'Account',
    'status.backendReady': 'Backend foundation is ready'
  },
  'pt-BR': {
    'app.title': 'Band Manager',
    'auth.bandNameLabel': 'Nome da banda',
    'auth.emailInstruction': 'Verifique seu email antes de configurar o espaco da banda.',
    'auth.emailLabel': 'Email',
    'auth.loginSubmit': 'Entrar',
    'auth.loginTitle': 'Entrar',
    'auth.onboardingSubmit': 'Criar espaco da banda',
    'auth.onboardingTitle': 'Configurar sua banda',
    'auth.passwordLabel': 'Senha',
    'auth.passwordReset': 'Redefinir senha',
    'auth.passwordResetSent': 'Verifique seu email para acessar o link de redefinicao.',
    'auth.passwordResetSubmit': 'Enviar link de redefinicao',
    'auth.passwordUpdateSubmit': 'Atualizar senha',
    'auth.passwordUpdateFailed':
      'A atualizacao da senha falhou. Solicite um novo link de recuperacao e tente novamente.',
    'auth.passwordUpdateTitle': 'Escolha uma nova senha',
    'auth.passwordUpdated': 'Senha atualizada.',
    'auth.signupSubmit': 'Criar conta',
    'auth.signupTitle': 'Criar conta',
    'auth.timezoneLabel': 'Fuso horario da banda',
    'auth.emailVerificationRequired':
      'Verifique seu email e depois entre para concluir a configuracao.',
    'auth.genericError': 'A autenticacao falhou. Confira os campos e tente novamente.',
    'auth.signupCreated': 'Conta criada.',
    'auth.loginReady': 'Login realizado.',
    'auth.logout': 'Sair',
    'auth.logoutFailed': 'Nao foi possivel sair. Tente novamente.',
    'auth.recoveryLinkError':
      'Este link de recuperacao e invalido ou expirou. Solicite um novo link.',
    'account.acceptLoading': 'Aceitando convite...',
    'account.acceptLoginPrompt': 'Entre com o email convidado para aceitar este convite.',
    'account.acceptMissingToken': 'O token do convite esta ausente.',
    'account.acceptSuccess': 'Convite aceito para',
    'account.acceptTitle': 'Aceitar convite',
    'account.actionsHeader': 'Acoes',
    'account.copyFailed': 'Nao foi possivel copiar o link do convite.',
    'account.copyInviteLink': 'Copiar link do convite',
    'account.copySuccess': 'Link do convite copiado.',
    'account.createInviteSubmit': 'Criar convite',
    'account.createInviteTitle': 'Convidar visualizador',
    'account.emailHeader': 'Email',
    'account.emailLabel': 'Email do visualizador',
    'account.expiresAtHeader': 'Expira',
    'account.genericError': 'A requisicao de conta falhou.',
    'account.inviteCreated': 'Convite criado.',
    'account.inviteEmailInvalid': 'Informe um email valido.',
    'account.invitesTitle': 'Convites',
    'account.inviteStatus.accepted': 'Aceito',
    'account.inviteStatus.expired': 'Expirado',
    'account.inviteStatus.pending': 'Pendente',
    'account.inviteStatus.revoked': 'Revogado',
    'account.joinedAtHeader': 'Entrada',
    'account.loading': 'Carregando conta...',
    'account.loginRequired': 'Entre para gerenciar o acesso da conta.',
    'account.membersTitle': 'Membros',
    'account.noInvites': 'Nenhum convite ainda.',
    'account.noMembers': 'Nenhum membro ainda.',
    'account.revokeInvite': 'Revogar',
    'account.role.admin': 'Admin',
    'account.role.member': 'Membro',
    'account.role.owner': 'Dono',
    'account.role.viewer': 'Visualizador',
    'account.roleHeader': 'Papel',
    'account.statusHeader': 'Status',
    'account.title': 'Conta',
    'inventory.addVariant': 'Adicionar variante',
    'inventory.addProduct': 'Adicionar produto',
    'inventory.addSize': 'Adicionar tamanho',
    'inventory.addVariantSuccess': 'Variante criada.',
    'inventory.actionsHeader': 'Acoes',
    'inventory.cancelEdit': 'Cancelar',
    'inventory.category.accessory': 'Acessorio',
    'inventory.category.cassette': 'Fita',
    'inventory.category.cd': 'CD',
    'inventory.category.hoodie': 'Moletom',
    'inventory.category.patch': 'Patch',
    'inventory.category.shirt': 'Camiseta',
    'inventory.category.sticker': 'Adesivo',
    'inventory.category.tote_bag': 'Ecobag',
    'inventory.category.vinyl': 'Vinil',
    'inventory.categoryHeader': 'Categoria',
    'inventory.categoryLabel': 'Categoria',
    'inventory.colourLabel': 'Cor',
    'inventory.costLabel': 'Custo (BRL)',
    'inventory.createSubmit': 'Criar produto',
    'inventory.createSuccess': 'Produto criado.',
    'inventory.createTitle': 'Criar produto',
    'inventory.createVariantSubmit': 'Criar variante',
    'inventory.deleteConfirm': 'Remover este produto do estoque?',
    'inventory.deleteProduct': 'Remover produto',
    'inventory.deleteSuccess': 'Produto removido.',
    'inventory.deleteVariant': 'Remover variante',
    'inventory.deleteVariantConfirm': 'Remover esta variante do estoque?',
    'inventory.deleteVariantSuccess': 'Variante removida.',
    'inventory.duplicateProduct': 'Ja existe um produto com esta categoria e nome.',
    'inventory.duplicateVariant':
      'Esta cor e tamanho ja existem, ou a cor tem foto, preco ou custo diferente.',
    'inventory.editCategoryLabel': 'Editar categoria',
    'inventory.editNameLabel': 'Editar nome do produto',
    'inventory.editProduct': 'Editar produto',
    'inventory.editTitle': 'Editar produto',
    'inventory.editVariant': 'Editar variante',
    'inventory.editVariantColourLabel': 'Editar cor da variante',
    'inventory.editVariantCostLabel': 'Editar custo da variante (BRL)',
    'inventory.editVariantPriceLabel': 'Editar preco da variante (BRL)',
    'inventory.editVariantQuantityLabel': 'Editar quantidade da variante',
    'inventory.editVariantSizeLabel': 'Editar tamanho da variante',
    'inventory.editVariantTitle': 'Editar variante',
    'inventory.existingColourSizeNote':
      'Este tamanho usa a foto, o preco e o custo existentes da cor.',
    'inventory.empty': 'Nenhum produto no estoque ainda.',
    'inventory.error': 'A requisicao do estoque falhou.',
    'inventory.formInvalid': 'Confira o formulario do estoque e tente novamente.',
    'inventory.inStock': 'Em estoque',
    'inventory.inStockCountSuffix': 'em estoque',
    'inventory.loading': 'Carregando estoque...',
    'inventory.lastVariantDeleteDisabled': 'Remova o produto para excluir sua ultima variante.',
    'inventory.nameLabel': 'Nome do produto',
    'inventory.newVariantColourLabel': 'Cor da nova variante',
    'inventory.newVariantCostLabel': 'Custo da nova variante (BRL)',
    'inventory.newVariantPriceLabel': 'Preco da nova variante (BRL)',
    'inventory.newVariantQuantityLabel': 'Quantidade da nova variante',
    'inventory.newVariantSizeLabel': 'Tamanho da nova variante',
    'inventory.photoInvalid': 'Nao foi possivel processar a foto.',
    'inventory.photoLabel': 'Foto',
    'inventory.photoPreviewAlt': 'Previa da foto do produto',
    'inventory.photoProcessing': 'Processando foto',
    'inventory.photoReady': 'Foto pronta',
    'inventory.photoRequired': 'A foto e obrigatoria.',
    'inventory.photoUploadFailed': 'O upload da foto falhou.',
    'inventory.photoUploadRequestFailed':
      'A requisicao de upload da foto falhou. Reinicie a API e confira VITE_API_BASE_URL.',
    'inventory.priceLabel': 'Preco (BRL)',
    'inventory.productCount': 'Produtos',
    'inventory.productHeader': 'Produto',
    'inventory.quantityLabel': 'Quantidade',
    'inventory.removeVariant': 'Remover variante',
    'inventory.removeSize': 'Remover tamanho',
    'inventory.size.g': 'G',
    'inventory.size.gg': 'GG',
    'inventory.size.m': 'M',
    'inventory.size.not_applicable': 'Nao aplicavel',
    'inventory.size.one_size': 'Tamanho unico',
    'inventory.size.p': 'P',
    'inventory.size.pp': 'PP',
    'inventory.size.xgg': 'XGG',
    'inventory.sizeLabel': 'Tamanho',
    'inventory.sizesTitle': 'Tamanhos e estoque',
    'inventory.soldOut': 'Esgotado',
    'inventory.statusHeader': 'Status',
    'inventory.stockHeader': 'Estoque',
    'inventory.updateSubmit': 'Salvar produto',
    'inventory.updateSuccess': 'Produto atualizado.',
    'inventory.updateVariantSubmit': 'Salvar variante',
    'inventory.updateVariantSuccess': 'Variante atualizada.',
    'inventory.variantSingular': 'variante',
    'inventory.variantsPlural': 'variantes',
    'inventory.variantsHeader': 'Variantes',
    'inventory.variantsTitle': 'Variantes',
    'merchBooth.addToCart': 'Adicionar ao carrinho',
    'merchBooth.backToBooth': 'Voltar para a banca',
    'merchBooth.cartEmpty': 'Seu carrinho esta vazio.',
    'merchBooth.cartPersistenceError':
      'O carrinho nao pode ser restaurado ou salvo nesta sessao do navegador.',
    'merchBooth.cartReadOnly': 'Seu perfil nao permite concluir vendas na banca.',
    'merchBooth.cartReconciled': 'O carrinho foi atualizado de acordo com o estoque atual.',
    'merchBooth.cartTitle': 'Carrinho',
    'merchBooth.cashCheckout': 'Concluir venda em dinheiro',
    'merchBooth.cashCheckoutSuccess': 'Venda em dinheiro concluida.',
    'merchBooth.checkoutError': 'A venda em dinheiro falhou. Confira o estoque e tente novamente.',
    'merchBooth.decreaseQuantity': 'Diminuir quantidade de',
    'merchBooth.empty': 'Nenhuma variante esta disponivel para venda.',
    'merchBooth.error': 'A requisicao da banca falhou.',
    'merchBooth.increaseQuantity': 'Aumentar quantidade de',
    'merchBooth.itemCount': 'Itens',
    'merchBooth.loading': 'Carregando banca...',
    'merchBooth.openCart': 'Abrir carrinho',
    'merchBooth.orderSummary': 'Resumo do pedido',
    'merchBooth.photoAlt': 'foto do produto',
    'merchBooth.selectSize': 'Selecione o tamanho',
    'merchBooth.removeFromCart': 'Remover do carrinho',
    'merchBooth.total': 'Total',
    'nav.inventory': 'Estoque',
    'nav.merchBooth': 'Banca',
    'nav.reports': 'Relatorios',
    'nav.calendar': 'Calendario',
    'nav.account': 'Conta',
    'status.backendReady': 'Base do backend pronta'
  }
}

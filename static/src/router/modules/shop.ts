import { AppRouteRecord } from '@/types/router'

export const shopRoutes: AppRouteRecord = {
  path: '/shop',
  name: 'ShopRoot',
  component: '/index/index',
  meta: {
    title: 'menus.shop.title',
    icon: 'ri:shopping-bag-line',
    login: true
  },
  children: [
    {
      path: 'browse',
      name: 'Shop',
      component: '/shop/browse',
      meta: {
        title: 'menus.shop.browse',
        keepAlive: true
      }
    },
    {
      path: 'manage',
      name: 'ShopManage',
      component: '/shop/manage',
      meta: {
        title: 'menus.shop.manage',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        authList: [
          { title: 'authActions.shop.addProduct', authMark: 'add_product' },
          { title: 'authActions.shop.editProduct', authMark: 'edit_product' },
          { title: 'authActions.shop.deleteProduct', authMark: 'delete_product' }
        ]
      }
    },
    {
      path: 'order-manage',
      name: 'ShopOrderManage',
      component: '/shop/order-manage',
      meta: {
        title: 'menus.shop.orderManage',
        keepAlive: true,
        roles: ['super_admin', 'admin', 'shop_order_manage'],
        authList: [{ title: 'authActions.shop.approveOrder', authMark: 'approve_order' }]
      }
    },
    {
      path: 'wallet',
      name: 'Wallet',
      component: '/shop/wallet',
      meta: {
        title: 'menus.shop.wallet',
        keepAlive: true
      }
    }
  ]
}

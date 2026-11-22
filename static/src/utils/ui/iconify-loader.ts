/**
 * 离线图标加载器
 *
 * 将项目实际使用的 Iconify 图标集在本地注册，避免运行时向 api.iconify.design 发外网请求。
 * 未包含在离线集合中的图标仍由 @iconify/vue 自动回退在线拉取。
 *
 * @module utils/ui/iconify-loader
 */

import { addCollection, type IconifyJSON } from '@iconify/vue'
import offlineCollections from '@/assets/icons/offline-iconify.json'

for (const collection of offlineCollections as unknown as IconifyJSON[]) {
  addCollection(collection)
}

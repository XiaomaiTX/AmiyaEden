import { AppRouteRecord } from '@/types/router'

export const systemRoutes: AppRouteRecord = {
  path: '/system',
  name: 'System',
  component: '/index/index',
  meta: {
    title: 'menus.system.title',
    icon: 'ri:user-3-line',
    roles: ['super_admin', 'admin'],
    corpCapabilitiesAny: [
      'system.manage',
      'system.task.read',
      'system.wallet.read',
      'system.audit.read',
      'system.basic_config.read'
    ]
  },
  children: [
    {
      path: 'user',
      name: 'User',
      component: '/system/user',
      meta: {
        title: 'menus.system.user',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAny: ['system.manage'],
        authList: [
          { title: 'authActions.system.deleteUser', authMark: 'delete_user' },
          { title: 'authActions.system.assignRole', authMark: 'assign_role' }
        ]
      }
    },
    {
      path: 'task-manager',
      name: 'TaskManager',
      component: '/system/task-manager',
      meta: {
        title: 'menus.system.taskManager',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAny: ['system.task.read'],
        authList: [
          { title: 'authActions.system.executeTask', authMark: 'execute_task' },
          { title: 'authActions.system.updateSchedule', authMark: 'update_schedule' }
        ]
      }
    },
    {
      path: 'wallet',
      name: 'SystemWallet',
      component: '/system/wallet',
      meta: {
        title: 'menus.system.wallet',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAny: ['system.wallet.read'],
        authList: [
          { title: 'authActions.system.adjustBalance', authMark: 'adjust_balance' },
          { title: 'authActions.system.viewLog', authMark: 'view_log' }
        ]
      }
    },
    {
      path: 'audit',
      name: 'SystemAudit',
      component: '/system/audit',
      meta: {
        title: 'menus.system.audit',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAny: ['system.audit.read'],
        authList: [{ title: 'authActions.system.viewAuditDetail', authMark: 'view_audit_detail' }]
      }
    },
    {
      path: 'pap-exchange',
      name: 'PAPExchange',
      component: '/system/pap-exchange',
      meta: {
        title: 'menus.system.papExchange',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAny: ['system.manage'],
        authList: [{ title: 'authActions.system.editExchangeRate', authMark: 'edit_exchange_rate' }]
      }
    },
    {
      path: 'pap',
      name: 'AlliancePAP',
      component: '/system/pap',
      meta: {
        title: 'menus.system.alliancePap',
        keepAlive: true,
        roles: ['super_admin', 'admin'],
        corpCapabilitiesAny: ['system.manage'],
        authList: [{ title: 'authActions.system.manualFetch', authMark: 'manual_fetch' }]
      }
    },
    {
      path: 'auto-role',
      name: 'AutoRole',
      component: '/system/auto-role',
      meta: {
        title: 'menus.system.autoRole',
        keepAlive: true,
        roles: ['super_admin'],
        corpCapabilitiesAny: ['system.basic_config.read']
      }
    },
    {
      path: 'user-center',
      name: 'UserCenter',
      component: '/system/user-center',
      meta: {
        title: 'menus.system.userCenter',
        isHide: true,
        keepAlive: true,
        isHideTab: true
      }
    },
    {
      path: 'webhook',
      name: 'WebhookSettings',
      component: '/system/webhook',
      meta: {
        title: 'menus.system.webhook',
        keepAlive: true,
        roles: ['super_admin'],
        corpCapabilitiesAny: ['system.manage']
      }
    },
    {
      path: 'qq-governance',
      name: 'QQGovernance',
      component: '/system/qq-governance',
      meta: {
        title: 'menus.system.qqGovernance',
        keepAlive: true,
        roles: ['super_admin'],
        corpCapabilitiesAny: ['system.manage']
      }
    },
    {
      path: 'basic-config',
      name: 'BasicConfig',
      component: '/system/basic-config',
      meta: {
        title: 'menus.system.basicConfig',
        keepAlive: true,
        roles: ['super_admin'],
        corpCapabilitiesAny: ['system.basic_config.read']
      }
    }
  ]
}

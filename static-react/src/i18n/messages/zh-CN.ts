const zhCN = {
  common: {
    backHome: '返回首页',
    switchLocale: '切换语言',
    confirm: '确认',
    cancel: '取消',
  },
  nav: {
    home: '首页',
    permissionDemo: '权限路由示例',
  },
  shell: {
    runtime: 'React 壳层',
    unnamedPage: '未命名页面',
    globalHostPlaceholder: 'GlobalHost 占位：消息、弹窗、抽屉将挂载于此',
  },
  home: {
    title: 'AmiyaEden React Shell',
    description: '壳层能力已切到 React：Sidebar、Header、PageContent、GlobalHost。',
    permissionRoute: '进入权限路由示例',
    open500: '打开 500 页面',
    open404: '触发 404 页面',
    showSuccessToast: '显示成功消息',
    showErrorToast: '显示错误消息',
    openConfirmDialog: '打开确认弹窗',
    mock401: '模拟 401 未授权',
  },
  auth: {
    loginTitle: 'EVE SSO 登录（模拟）',
    loginDescription: '当前为 React 迁移阶段登录占位页。',
    loginRedirectTo: '登录后回跳',
    mockLogin: '模拟登录',
    alreadyLoggedIn: '当前已登录，可直接进入业务路由。',
  },
  errors: {
    forbiddenTitle: '403 Forbidden',
    forbiddenDesc: '当前账号无权访问该页面。',
    notFoundTitle: '404 Not Found',
    notFoundDesc: '未匹配到页面路由。',
    serverErrorTitle: '500 Server Error',
    serverErrorDesc: '服务异常，请稍后重试。',
  },
  feedback: {
    unauthorized: '登录态已失效，请重新登录。',
    successDemo: '操作成功。',
    errorDemo: '操作失败，请稍后重试。',
    confirmTitle: '确认操作',
    confirmMessage: '是否继续执行该操作？',
    confirmed: '已确认。',
    cancelled: '已取消。',
  },
} as const

export default zhCN

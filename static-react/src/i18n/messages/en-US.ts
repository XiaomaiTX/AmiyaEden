const enUS = {
  common: {
    backHome: 'Back Home',
    switchLocale: 'Switch Locale',
    confirm: 'Confirm',
    cancel: 'Cancel',
  },
  nav: {
    home: 'Home',
    permissionDemo: 'Permission Demo',
  },
  shell: {
    runtime: 'React Shell',
    unnamedPage: 'Untitled Page',
    globalHostPlaceholder: 'GlobalHost placeholder: message, modal, and drawer mounts.',
  },
  home: {
    title: 'AmiyaEden React Shell',
    description: 'Shell capabilities are now migrated to React: Sidebar, Header, PageContent, GlobalHost.',
    permissionRoute: 'Open Permission Route Demo',
    open500: 'Open 500 Page',
    open404: 'Trigger 404 Route',
    showSuccessToast: 'Show Success Toast',
    showErrorToast: 'Show Error Toast',
    openConfirmDialog: 'Open Confirm Dialog',
    mock401: 'Simulate 401 Unauthorized',
  },
  auth: {
    loginTitle: 'EVE SSO Login (Mock)',
    loginDescription: 'This is a placeholder login page during React migration.',
    loginRedirectTo: 'Redirect after login',
    mockLogin: 'Mock Login',
    alreadyLoggedIn: 'You are already logged in and can enter protected routes.',
  },
  errors: {
    forbiddenTitle: '403 Forbidden',
    forbiddenDesc: 'Your account is not allowed to access this page.',
    notFoundTitle: '404 Not Found',
    notFoundDesc: 'The route does not match any page.',
    serverErrorTitle: '500 Server Error',
    serverErrorDesc: 'Service error, please try again later.',
  },
  feedback: {
    unauthorized: 'Session expired, please sign in again.',
    successDemo: 'Operation succeeded.',
    errorDemo: 'Operation failed, please try again later.',
    confirmTitle: 'Confirm Action',
    confirmMessage: 'Do you want to continue this action?',
    confirmed: 'Confirmed.',
    cancelled: 'Cancelled.',
  },
} as const

export default enUS

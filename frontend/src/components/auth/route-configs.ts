export const adminRoute = {
  shouldShow: (isAuth: boolean, isAdmin: boolean) => isAuth && isAdmin,
  redirectPath: '/login',
  withNext: true,
  showForbidden: true,
}

export const protectedRoute = {
  shouldShow: (isAuth: boolean) => isAuth,
  redirectPath: '/login',
  withNext: true,
}

export const publicRoute = {
  shouldShow: (isAuth: boolean) => !isAuth,
  redirectPath: '/dashboard',
  showSpinnerWhen: (isAuth: boolean) => !isAuth,
}

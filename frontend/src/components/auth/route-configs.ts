export const adminRoute = {
  shouldShow: (isAuth: boolean, isAdmin: boolean) => isAuth && isAdmin,
  redirectPath: "/dashboard",
};

export const protectedRoute = {
  shouldShow: (isAuth: boolean) => isAuth,
  redirectPath: "/login",
};

export const publicRoute = {
  shouldShow: (isAuth: boolean) => !isAuth,
  redirectPath: "/dashboard",
  showSpinnerWhen: (isAuth: boolean) => !isAuth,
};